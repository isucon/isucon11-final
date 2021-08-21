use actix_web::web;
use actix_web::HttpResponse;
use futures::future;
use futures::StreamExt as _;
use sqlx::Arguments as _;
use sqlx::Executor as _;
use tokio::io::AsyncWriteExt as _;

const SQL_DIRECTORY: &str = "../sql/";
const ASSIGNMENTS_DIRECTORY: &str = "../assignments/";
const SESSION_NAME: &str = "session";
const MYSQL_ERR_NUM_DUPLICATE_ENTRY: u16 = 1062;

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    env_logger::Builder::from_env(env_logger::Env::default().default_filter_or("info,sqlx=warn"))
        .init();

    let pool = sqlx::mysql::MySqlPoolOptions::new()
        .max_connections(10)
        .after_connect(|conn| {
            Box::pin(async move {
                conn.execute("set time_zone = '+00:00'").await?;
                Ok(())
            })
        })
        .connect_with(
            sqlx::mysql::MySqlConnectOptions::new()
                .host("127.0.0.1")
                .port(3306)
                .database("isucholar")
                .username("isucon")
                .password("isucon")
                .ssl_mode(sqlx::mysql::MySqlSslMode::Disabled),
        )
        .await
        .expect("failed to connect db");

    let mut session_key = b"trapnomura".to_vec();
    session_key.resize(32, 0);

    let server = actix_web::HttpServer::new(move || {
        let users_api = web::scope("/users")
            .route("/me", web::get().to(get_me))
            .route("/me/courses", web::get().to(get_registered_courses))
            .route("/me/courses", web::put().to(register_courses))
            .route("/me/grades", web::get().to(get_grades));

        let syllabus_api = web::scope("/syllabus")
            .route("", web::get().to(search_courses))
            .route("/{course_id}", web::get().to(get_course_detail));

        let courses_admin_api = web::scope("/courses")
            .wrap(IsAdmin)
            .route("", web::post().to(add_course))
            .route("/{course_id}/status", web::put().to(set_course_status))
            .route("/{course_id}/classes", web::get().to(get_classes))
            .route("/{course_id}/classes", web::get().to(add_class))
            .route(
                "/{course_id}/classes/{class_id}/assignments",
                web::post().to(register_scores),
            )
            .route(
                "/{course_id}/classes/{class_id}/assignments/export",
                web::get().to(download_submitted_assignments),
            );
        let courses_api = web::scope("/courses").route(
            "/{course_id}/classes/{class_id}/assignment",
            web::post().to(submit_assignment),
        );

        let announcements_api = web::scope("/announcements")
            .route("", web::get().to(get_announcement_list))
            .route("/{announcement_id}", web::get().to(get_announcement_detail));
        let announcements_admin_api = web::scope("/announcements")
            .wrap(IsAdmin)
            .route("", web::post().to(add_announcement));

        actix_web::App::new()
            .app_data(web::Data::new(pool.clone()))
            .wrap(actix_web::middleware::Logger::default())
            .wrap(
                actix_session::CookieSession::signed(&session_key)
                    .secure(false)
                    .name(SESSION_NAME)
                    .max_age(3600),
            )
            .route("/initialize", web::post().to(initialize))
            .route("/login", web::post().to(login))
            .service(
                web::scope("/api")
                    .wrap(IsLoggedIn)
                    .service(users_api)
                    .service(syllabus_api)
                    .service(courses_admin_api)
                    .service(courses_api)
                    .service(announcements_admin_api)
                    .service(announcements_api),
            )
    });
    if let Some(l) = listenfd::ListenFd::from_env().take_tcp_listener(0)? {
        server.listen(l)?
    } else {
        server.bind("127.0.0.1:7000")?
    }
    .run()
    .await
}

#[derive(Debug)]
struct SqlxError(sqlx::Error);
impl std::fmt::Display for SqlxError {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        self.0.fmt(f)
    }
}
impl actix_web::ResponseError for SqlxError {
    fn error_response(&self) -> HttpResponse {
        log::error!("{}", self);
        HttpResponse::InternalServerError()
            .content_type(mime::TEXT_PLAIN)
            .body(format!("SQLx error: {:?}", self.0))
    }
}

#[derive(Debug, serde::Serialize)]
#[serde(transparent)]
struct UuidInDb(uuid::Uuid);
impl sqlx::Type<sqlx::MySql> for UuidInDb {
    fn type_info() -> sqlx::mysql::MySqlTypeInfo {
        str::type_info()
    }

    fn compatible(ty: &sqlx::mysql::MySqlTypeInfo) -> bool {
        <&str>::compatible(ty)
    }
}
impl<'r> sqlx::Decode<'r, sqlx::MySql> for UuidInDb {
    fn decode(
        value: sqlx::mysql::MySqlValueRef<'r>,
    ) -> Result<Self, Box<dyn std::error::Error + Sync + Send>> {
        let hyphenated_uuid = <&'r str>::decode(value)?;
        Ok(Self(uuid::Uuid::parse_str(hyphenated_uuid)?))
    }
}
impl<'q> sqlx::Encode<'q, sqlx::MySql> for UuidInDb {
    fn encode_by_ref(&self, buf: &mut Vec<u8>) -> sqlx::encode::IsNull {
        self.0.to_hyphenated().to_string().encode_by_ref(buf)
    }
}

#[derive(Debug, serde::Serialize)]
struct InitializeResponse {
    language: &'static str,
}

// 初期化エンドポイント
async fn initialize(pool: web::Data<sqlx::MySqlPool>) -> actix_web::Result<HttpResponse> {
    let files = ["1_schema.sql", "2_init.sql"];
    for file in files {
        let data = tokio::fs::read_to_string(format!("{}{}", SQL_DIRECTORY, file)).await?;
        let mut stream = pool.execute_many(data.as_str());
        while let Some(result) = stream.next().await {
            result.map_err(SqlxError)?;
        }
    }

    if !tokio::process::Command::new("rm")
        .arg("-rf")
        .arg(ASSIGNMENTS_DIRECTORY)
        .status()
        .await?
        .success()
    {
        return Err(actix_web::error::ErrorInternalServerError(""));
    }
    if !tokio::process::Command::new("mkdir")
        .arg(ASSIGNMENTS_DIRECTORY)
        .status()
        .await?
        .success()
    {
        return Err(actix_web::error::ErrorInternalServerError(""));
    }

    Ok(HttpResponse::Ok().json(InitializeResponse { language: "rust" }))
}

// ログイン確認用middleware
struct IsLoggedIn;
impl<S, B> actix_web::dev::Transform<S, actix_web::dev::ServiceRequest> for IsLoggedIn
where
    S: actix_web::dev::Service<
        actix_web::dev::ServiceRequest,
        Response = actix_web::dev::ServiceResponse<B>,
        Error = actix_web::error::Error,
    >,
{
    type Response = actix_web::dev::ServiceResponse<B>;
    type Error = actix_web::Error;
    type Transform = IsLoggedInMiddleware<S>;
    type InitError = ();
    type Future = future::Ready<Result<Self::Transform, Self::InitError>>;

    fn new_transform(&self, service: S) -> Self::Future {
        future::ok(IsLoggedInMiddleware { service })
    }
}
struct IsLoggedInMiddleware<S> {
    service: S,
}
impl<S, B> actix_web::dev::Service<actix_web::dev::ServiceRequest> for IsLoggedInMiddleware<S>
where
    S: actix_web::dev::Service<
        actix_web::dev::ServiceRequest,
        Response = actix_web::dev::ServiceResponse<B>,
        Error = actix_web::error::Error,
    >,
{
    type Response = actix_web::dev::ServiceResponse<B>;
    type Error = actix_web::Error;
    type Future = future::Either<S::Future, future::Ready<Result<Self::Response, Self::Error>>>;

    actix_web::dev::forward_ready!(service);

    fn call(&self, req: actix_web::dev::ServiceRequest) -> Self::Future {
        use actix_session::UserSession as _;
        use futures::FutureExt as _;

        match req.get_session().get::<String>("userID") {
            Ok(Some(_)) => self.service.call(req).left_future(),
            Ok(None) => future::err(actix_web::error::ErrorForbidden("You are not logged in."))
                .right_future(),
            Err(e) => future::err(e).right_future(),
        }
    }
}

// admin確認用middleware
struct IsAdmin;
impl<S, B> actix_web::dev::Transform<S, actix_web::dev::ServiceRequest> for IsAdmin
where
    S: actix_web::dev::Service<
        actix_web::dev::ServiceRequest,
        Response = actix_web::dev::ServiceResponse<B>,
        Error = actix_web::error::Error,
    >,
{
    type Response = actix_web::dev::ServiceResponse<B>;
    type Error = actix_web::Error;
    type Transform = IsAdminMiddleware<S>;
    type InitError = ();
    type Future = future::Ready<Result<Self::Transform, Self::InitError>>;

    fn new_transform(&self, service: S) -> Self::Future {
        future::ok(IsAdminMiddleware { service })
    }
}
struct IsAdminMiddleware<S> {
    service: S,
}
impl<S, B> actix_web::dev::Service<actix_web::dev::ServiceRequest> for IsAdminMiddleware<S>
where
    S: actix_web::dev::Service<
        actix_web::dev::ServiceRequest,
        Response = actix_web::dev::ServiceResponse<B>,
        Error = actix_web::error::Error,
    >,
{
    type Response = actix_web::dev::ServiceResponse<B>;
    type Error = actix_web::Error;
    type Future = future::Either<S::Future, future::Ready<Result<Self::Response, Self::Error>>>;

    actix_web::dev::forward_ready!(service);

    fn call(&self, req: actix_web::dev::ServiceRequest) -> Self::Future {
        use actix_session::UserSession as _;
        use futures::FutureExt as _;

        match req.get_session().get::<bool>("isAdmin") {
            Ok(Some(true)) => self.service.call(req).left_future(),
            Ok(Some(false)) | Ok(None) => {
                future::err(actix_web::error::ErrorForbidden("You are not admin user."))
                    .right_future()
            }
            Err(e) => future::err(e).right_future(),
        }
    }
}

fn get_user_id(session: actix_session::Session) -> actix_web::Result<(uuid::Uuid, bool)> {
    let user_id = session.get("userID")?;
    if user_id.is_none() {
        return Err(actix_web::error::ErrorInternalServerError(
            "failed to get userID from session",
        ));
    }
    let is_admin = session.get("isAdmin")?;
    if is_admin.is_none() {
        return Err(actix_web::error::ErrorInternalServerError(
            "failed to get isAdmin from session",
        ));
    }
    Ok((user_id.unwrap(), is_admin.unwrap()))
}

#[derive(Debug, PartialEq, Eq)]
enum UserType {
    Student,
    Teacher,
}
impl sqlx::Type<sqlx::MySql> for UserType {
    fn type_info() -> sqlx::mysql::MySqlTypeInfo {
        str::type_info()
    }

    fn compatible(ty: &sqlx::mysql::MySqlTypeInfo) -> bool {
        <&str>::compatible(ty)
    }
}
impl<'r> sqlx::Decode<'r, sqlx::MySql> for UserType {
    fn decode(
        value: sqlx::mysql::MySqlValueRef<'r>,
    ) -> Result<Self, Box<dyn std::error::Error + Sync + Send>> {
        match <&'r str>::decode(value)? {
            "student" => Ok(Self::Student),
            "teacher" => Ok(Self::Teacher),
            v => Err(format!("Unknown enum variant: {}", v).into()),
        }
    }
}
impl<'q> sqlx::Encode<'q, sqlx::MySql> for UserType {
    fn encode_by_ref(&self, buf: &mut Vec<u8>) -> sqlx::encode::IsNull {
        match *self {
            Self::Teacher => "teacher",
            Self::Student => "student",
        }
        .encode_by_ref(buf)
    }
}

#[derive(Debug, sqlx::FromRow)]
struct User {
    id: UuidInDb,
    code: String,
    name: String,
    hashed_password: Vec<u8>,
    #[sqlx(rename = "type")]
    type_: UserType,
}

#[derive(Debug, serde::Deserialize)]
#[serde(rename_all = "kebab-case")]
enum CourseType {
    LiberalArts,
    MajorSubjects,
}
impl sqlx::Type<sqlx::MySql> for CourseType {
    fn type_info() -> sqlx::mysql::MySqlTypeInfo {
        str::type_info()
    }

    fn compatible(ty: &sqlx::mysql::MySqlTypeInfo) -> bool {
        <&str>::compatible(ty)
    }
}
impl<'r> sqlx::Decode<'r, sqlx::MySql> for CourseType {
    fn decode(
        value: sqlx::mysql::MySqlValueRef<'r>,
    ) -> Result<Self, Box<dyn std::error::Error + Sync + Send>> {
        match <&'r str>::decode(value)? {
            "liberal-arts" => Ok(Self::LiberalArts),
            "major-subjects" => Ok(Self::MajorSubjects),
            v => Err(format!("Unknown enum variant: {}", v).into()),
        }
    }
}
impl<'q> sqlx::Encode<'q, sqlx::MySql> for CourseType {
    fn encode_by_ref(&self, buf: &mut Vec<u8>) -> sqlx::encode::IsNull {
        match *self {
            Self::LiberalArts => "liberal-arts",
            Self::MajorSubjects => "major-subjects",
        }
        .encode_by_ref(buf)
    }
}

#[derive(Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "lowercase")]
enum DayOfWeek {
    Sunday,
    Monday,
    Tuesday,
    Wednesday,
    Thursday,
    Friday,
    Saturday,
}
impl sqlx::Type<sqlx::MySql> for DayOfWeek {
    fn type_info() -> sqlx::mysql::MySqlTypeInfo {
        str::type_info()
    }

    fn compatible(ty: &sqlx::mysql::MySqlTypeInfo) -> bool {
        <&str>::compatible(ty)
    }
}
impl<'r> sqlx::Decode<'r, sqlx::MySql> for DayOfWeek {
    fn decode(
        value: sqlx::mysql::MySqlValueRef<'r>,
    ) -> Result<Self, Box<dyn std::error::Error + Sync + Send>> {
        match <&'r str>::decode(value)? {
            "sunday" => Ok(Self::Sunday),
            "monday" => Ok(Self::Monday),
            "tuesday" => Ok(Self::Tuesday),
            "wednesday" => Ok(Self::Wednesday),
            "thursday" => Ok(Self::Thursday),
            "friday" => Ok(Self::Friday),
            "saturday" => Ok(Self::Saturday),
            v => Err(format!("Unknown enum variant: {}", v).into()),
        }
    }
}
impl<'q> sqlx::Encode<'q, sqlx::MySql> for DayOfWeek {
    fn encode_by_ref(&self, buf: &mut Vec<u8>) -> sqlx::encode::IsNull {
        match *self {
            Self::Sunday => "sunday",
            Self::Monday => "monday",
            Self::Tuesday => "tuesday",
            Self::Wednesday => "wednesday",
            Self::Thursday => "thursday",
            Self::Friday => "friday",
            Self::Saturday => "saturday",
        }
        .encode_by_ref(buf)
    }
}

#[derive(Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "kebab-case")]
enum CourseStatus {
    Registration,
    InProgress,
    Closed,
}
impl sqlx::Type<sqlx::MySql> for CourseStatus {
    fn type_info() -> sqlx::mysql::MySqlTypeInfo {
        str::type_info()
    }

    fn compatible(ty: &sqlx::mysql::MySqlTypeInfo) -> bool {
        <&str>::compatible(ty)
    }
}
impl<'r> sqlx::Decode<'r, sqlx::MySql> for CourseStatus {
    fn decode(
        value: sqlx::mysql::MySqlValueRef<'r>,
    ) -> Result<Self, Box<dyn std::error::Error + Sync + Send>> {
        match <&'r str>::decode(value)? {
            "registration" => Ok(Self::Registration),
            "in-progress" => Ok(Self::InProgress),
            "closed" => Ok(Self::Closed),
            v => Err(format!("Unknown enum variant: {}", v).into()),
        }
    }
}
impl<'q> sqlx::Encode<'q, sqlx::MySql> for CourseStatus {
    fn encode_by_ref(&self, buf: &mut Vec<u8>) -> sqlx::encode::IsNull {
        match *self {
            Self::Registration => "registration",
            Self::InProgress => "in-progress",
            Self::Closed => "closed",
        }
        .encode_by_ref(buf)
    }
}

#[derive(Debug, sqlx::FromRow)]
struct Course {
    id: UuidInDb,
    code: String,
    #[sqlx(rename = "type")]
    type_: CourseType,
    name: String,
    description: String,
    credit: u8,
    period: u8,
    day_of_week: DayOfWeek,
    teacher_id: UuidInDb,
    keywords: String,
    status: CourseStatus,
}

#[derive(Debug, serde::Deserialize)]
struct LoginRequest {
    code: String,
    password: String,
}

// ログイン
async fn login(
    session: actix_session::Session,
    pool: web::Data<sqlx::MySqlPool>,
    req: web::Json<LoginRequest>,
) -> actix_web::Result<HttpResponse> {
    let user: Option<User> = sqlx::query_as("SELECT * FROM `users` WHERE `code` = ?")
        .bind(&req.code)
        .fetch_optional(pool.as_ref())
        .await
        .map_err(SqlxError)?;
    if user.is_none() {
        return Err(actix_web::error::ErrorUnauthorized(
            "Code or Password is wrong.",
        ));
    }
    let user = user.unwrap();

    if !bcrypt::verify(
        &req.password,
        &String::from_utf8_lossy(&user.hashed_password),
    )
    .map_err(actix_web::error::ErrorInternalServerError)?
    {
        return Err(actix_web::error::ErrorUnauthorized(
            "Code or Password is wrong.",
        ));
    }

    if let Some(user_id) = session.get::<uuid::Uuid>("userID")? {
        if user_id == user.id.0 {
            return Err(actix_web::error::ErrorBadRequest(
                "You are already logged in.",
            ));
        }
    }

    session.insert("userID", user.id.0)?;
    session.insert("userName", user.name)?;
    session.insert("isAdmin", user.type_ == UserType::Teacher)?;
    Ok(HttpResponse::Ok().finish())
}

#[derive(Debug, serde::Serialize)]
struct GetMeResponse {
    code: String,
    is_admin: bool,
}

async fn get_me(
    pool: web::Data<sqlx::MySqlPool>,
    session: actix_session::Session,
) -> actix_web::Result<HttpResponse> {
    let (user_id, is_admin) = get_user_id(session)?;

    let user_code = sqlx::query_scalar("SELECT `code` FROM `users` WHERE `id` = ?")
        .bind(UuidInDb(user_id))
        .fetch_one(pool.as_ref())
        .await
        .map_err(SqlxError)?;

    Ok(HttpResponse::Ok().json(GetMeResponse {
        code: user_code,
        is_admin,
    }))
}

#[derive(Debug, serde::Serialize)]
struct GetRegisteredCourseResponseContent {
    id: uuid::Uuid,
    name: String,
    teacher: String,
    period: u8,
    day_of_week: DayOfWeek,
}

// 履修中の科目一覧取得
async fn get_registered_courses(
    pool: web::Data<sqlx::MySqlPool>,
    session: actix_session::Session,
) -> actix_web::Result<HttpResponse> {
    let (user_id, _) = get_user_id(session)?;

    let courses: Vec<Course> = sqlx::query_as(concat!(
        "SELECT `courses`.*",
        " FROM `courses`",
        " JOIN `registrations` ON `courses`.`id` = `registrations`.`course_id`",
        " WHERE `courses`.`status` != ? AND `registrations`.`user_id` = ?",
    ))
    .bind(CourseStatus::Closed)
    .bind(UuidInDb(user_id))
    .fetch_all(pool.as_ref())
    .await
    .map_err(SqlxError)?;

    // 履修科目が0件の時は空配列を返却
    let mut res = Vec::with_capacity(courses.len());
    for course in courses {
        let teacher: User = sqlx::query_as("SELECT * FROM `users` WHERE `id` = ?")
            .bind(&course.teacher_id)
            .fetch_one(pool.as_ref())
            .await
            .map_err(SqlxError)?;

        res.push(GetRegisteredCourseResponseContent {
            id: course.id.0,
            name: course.name,
            teacher: teacher.name,
            period: course.period,
            day_of_week: course.day_of_week,
        });
    }

    Ok(HttpResponse::Ok().json(res))
}

#[derive(Debug, serde::Deserialize)]
struct RegisterCourseRequestContent {
    id: String,
}

#[derive(Debug, Default, serde::Serialize)]
struct RegisterCoursesErrorResponse {
    #[serde(skip_serializing_if = "Vec::is_empty")]
    course_not_found: Vec<String>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    not_registrable_status: Vec<uuid::Uuid>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    schedule_conflict: Vec<uuid::Uuid>,
}

async fn register_courses(
    pool: web::Data<sqlx::MySqlPool>,
    session: actix_session::Session,
    req: web::Json<Vec<RegisterCourseRequestContent>>,
) -> actix_web::Result<HttpResponse> {
    let (user_id, _) = get_user_id(session)?;

    let mut req = req.into_inner();
    req.sort_by(|x, y| x.id.cmp(&y.id));

    let mut tx = pool.begin().await.map_err(SqlxError)?;

    let mut errors = RegisterCoursesErrorResponse::default();
    let mut newly_added = Vec::new();
    for course_req in req {
        let course_id = uuid::Uuid::parse_str(&course_req.id);
        if course_id.is_err() {
            errors.course_not_found.push(course_req.id);
            continue;
        }
        let course_id = course_id.unwrap();

        let course: Option<Course> =
            sqlx::query_as("SELECT * FROM `courses` WHERE `id` = ? FOR SHARE")
                .bind(UuidInDb(course_id))
                .fetch_optional(&mut tx)
                .await
                .map_err(SqlxError)?;
        if course.is_none() {
            errors.course_not_found.push(course_req.id);
            continue;
        }
        let course = course.unwrap();

        if course.status != CourseStatus::Registration {
            errors.not_registrable_status.push(course.id.0);
            continue;
        }

        // MEMO: すでに履修登録済みの科目は無視する
        let count: i64 = sqlx::query_scalar(
            "SELECT COUNT(*) FROM `registrations` WHERE `course_id` = ? AND `user_id` = ?",
        )
        .bind(&course.id)
        .bind(UuidInDb(user_id))
        .fetch_one(&mut tx)
        .await
        .map_err(SqlxError)?;
        if count > 0 {
            continue;
        }

        newly_added.push(course);
    }

    // MEMO: スケジュールの重複バリデーション
    let already_registered: Vec<Course> = sqlx::query_as(concat!(
        "SELECT `courses`.*",
        " FROM `courses`",
        " JOIN `registrations` ON `courses`.`id` = `registrations`.`course_id`",
        " WHERE `courses`.`status` != ? AND `registrations`.`user_id` = ?",
    ))
    .bind(CourseStatus::Closed)
    .bind(UuidInDb(user_id))
    .fetch_all(&mut tx)
    .await
    .map_err(SqlxError)?;

    for course1 in newly_added.iter() {
        for course2 in already_registered.iter().chain(newly_added.iter()) {
            if course1.id.0 != course2.id.0
                && course1.period == course2.period
                && course1.day_of_week == course2.day_of_week
            {
                errors.schedule_conflict.push(course1.id.0);
            }
        }
    }

    if !errors.course_not_found.is_empty()
        || !errors.not_registrable_status.is_empty()
        || !errors.schedule_conflict.is_empty()
    {
        return Ok(HttpResponse::BadRequest().json(errors));
    }

    for course in newly_added {
        sqlx::query("INSERT INTO `registrations` (`course_id`, `user_id`) VALUES (?, ?)")
            .bind(course.id)
            .bind(UuidInDb(user_id))
            .execute(&mut tx)
            .await
            .map_err(SqlxError)?;
    }

    tx.commit().await.map_err(SqlxError)?;

    Ok(HttpResponse::Ok().finish())
}

#[derive(Debug, sqlx::FromRow)]
struct Class {
    id: UuidInDb,
    course_id: UuidInDb,
    part: u8,
    title: String,
    description: String,
    submission_closed: bool,
}

#[derive(Debug, serde::Serialize)]
struct GetGradeResponse {
    summary: Summary,
    #[serde(rename = "courses")]
    course_results: Vec<CourseResult>,
}

#[derive(Debug, Default, serde::Serialize)]
struct Summary {
    credits: i64,
    gpt: f64,
    gpt_t_score: f64, // 偏差値
    gpt_avg: f64,     // 平均値
    gpt_max: f64,     // 最大値
    gpt_min: f64,     // 最小値
}

#[derive(Debug, serde::Serialize)]
struct CourseResult {
    name: String,
    code: String,
    total_score: i64,
    total_score_t_score: f64, // 偏差値
    total_score_avg: f64,     // 平均値
    total_score_max: i64,     // 最大値
    total_score_min: i64,     // 最小値
    class_scores: Vec<ClassScore>,
}

#[derive(Debug, serde::Serialize)]
struct ClassScore {
    class_id: uuid::Uuid,
    title: String,
    part: u8,
    score: i64,      // 0~100点
    submitters: i64, // 提出した生徒数
}

#[derive(Debug, sqlx::FromRow)]
struct SubmissionWithClassName {
    user_id: UuidInDb,
    class_id: UuidInDb,
    file_name: String,
    score: Option<i64>,
    part: u8,
    title: String,
}

async fn get_grades(
    pool: web::Data<sqlx::MySqlPool>,
    session: actix_session::Session,
) -> actix_web::Result<HttpResponse> {
    let (user_id, _) = get_user_id(session)?;

    // 登録済の科目一覧取得
    let registered_courses: Vec<Course> = sqlx::query_as(concat!(
        "SELECT `courses`.*",
        " FROM `registrations`",
        " JOIN `courses` ON `registrations`.`course_id` = `courses`.`id`",
        " WHERE `user_id` = ?",
    ))
    .bind(UuidInDb(user_id))
    .fetch_all(pool.as_ref())
    .await
    .map_err(SqlxError)?;

    // 科目毎の成績計算処理
    let mut course_results = Vec::with_capacity(registered_courses.len());
    let mut summary = Summary::default();
    for course in registered_courses {
        // この科目を受講している学生のTotalScore一覧を取得
        let totals: Vec<i64> = sqlx::query_scalar(concat!(
            "SELECT IFNULL(SUM(`submissions`.`score`), 0) AS `total_score`",
            " FROM `submissions`",
            " JOIN `classes` ON `submissions`.`class_id` = `classes`.`id`",
            " WHERE `classes`.`course_id` = ?",
            " GROUP BY `user_id`",
        ))
        .bind(&course.id)
        .fetch_all(pool.as_ref())
        .await
        .map_err(SqlxError)?;

        // avg max min stdの計算
        let total_score_count = totals.len();
        let mut total_score_avg = 0f64;
        let mut total_score_max = 0i64;
        let mut total_score_min = 500i64; // 1科目5クラスなので最大500点
        let mut total_score_std_dev = 0f64; // 標準偏差

        for total_score in &totals {
            total_score_avg += *total_score as f64 / total_score_count as f64;
            total_score_max = total_score_max.max(*total_score);
            total_score_min = total_score_min.min(*total_score);
        }
        for total_score in totals {
            total_score_std_dev +=
                (total_score as f64 - total_score_avg).powi(2) / total_score_count as f64;
        }
        total_score_std_dev = total_score_std_dev.sqrt();

        // クラス一覧の取得
        let classes: Vec<Class> = sqlx::query_as(concat!(
            "SELECT *",
            " FROM `classes`",
            " WHERE `course_id` = ?",
            " ORDER BY `part` DESC",
        ))
        .bind(&course.id)
        .fetch_all(pool.as_ref())
        .await
        .map_err(SqlxError)?;

        // クラス毎の成績計算処理
        let mut class_scores = Vec::with_capacity(classes.len());
        let mut my_total_score = 0;
        for class in classes {
            let submissions: Vec<SubmissionWithClassName> = sqlx::query_as(concat!(
                "SELECT `submissions`.*, `classes`.`part` AS `part`, `classes`.`title` AS `title`",
                " FROM `submissions`",
                " JOIN `classes` ON `submissions`.`class_id` = `classes`.`id`",
                " WHERE `submissions`.`class_id` = ?",
            ))
            .bind(&class.id)
            .fetch_all(pool.as_ref())
            .await
            .map_err(SqlxError)?;

            let submitters = submissions.len() as i64;
            for submission in submissions {
                if user_id == submission.user_id.0 {
                    let my_score = submission.score.unwrap_or(0);
                    class_scores.push(ClassScore {
                        class_id: class.id.0,
                        part: submission.part,
                        title: submission.title,
                        score: my_score,
                        submitters,
                    });
                    my_total_score += my_score;
                }
            }
        }

        // 対象科目の自分の偏差値の計算
        let total_score_t_score = if total_score_std_dev == 0.0 {
            50.0
        } else {
            (my_total_score as f64 - total_score_avg) / total_score_std_dev * 10.0 + 50.0
        };

        course_results.push(CourseResult {
            name: course.name,
            code: course.code,
            total_score: my_total_score,
            total_score_t_score,
            total_score_avg,
            total_score_max,
            total_score_min,
            class_scores,
        });

        // 自分のGPT計算
        summary.gpt += (my_total_score * course.credit as i64) as f64 / 100.0;
        summary.credits += course.credit as i64;
    }

    // GPTの統計値
    // 全学生ごとのGPT
    let gpts: Vec<f64> = sqlx::query_scalar(concat!(
        "SELECT IFNULL(SUM(`submissions`.`score` * `courses`.`credit` / 100), 0) AS `gpt`",
        " FROM `users`",
        " LEFT JOIN `submissions` ON `users`.`id` = `submissions`.`user_id`",
        " LEFT JOIN `classes` ON `submissions`.`class_id` = `classes`.`id`",
        " LEFT JOIN `courses` ON `classes`.`course_id` = `courses`.`id`",
        " WHERE `users`.`type` = ?",
        " GROUP BY `user_id`",
    ))
    .bind(UserType::Student)
    .fetch_all(pool.as_ref())
    .await
    .map_err(SqlxError)?;

    // avg max min stdの計算
    let gpt_count = gpts.len();
    let mut gpt_avg = 0f64;
    let mut gpt_max = 0f64;
    // MEMO: 1コース500点かつ5秒で20コースを12回転=(240コース)の1/100なので最大1200点
    let mut gpt_min = f64::MAX;
    let mut gpt_std_dev = 0f64;
    for gpt in &gpts {
        gpt_avg += gpt / gpt_count as f64;
        gpt_max = gpt_max.max(*gpt);
        gpt_min = gpt_min.min(*gpt);
    }

    for gpt in gpts {
        gpt_std_dev += (gpt - gpt_avg).powi(2) / gpt_count as f64;
    }
    gpt_std_dev = gpt_std_dev.sqrt();

    // 自分の偏差値の計算
    let gpt_t_score = if gpt_std_dev == 0.0 {
        50.0
    } else {
        (summary.gpt - gpt_avg) / gpt_std_dev * 10.0 + 50.0
    };

    Ok(HttpResponse::Ok().json(GetGradeResponse {
        course_results,
        summary: Summary {
            gpt_t_score,
            gpt_avg,
            gpt_max,
            gpt_min,
            ..summary
        },
    }))
}

#[derive(Debug, serde::Deserialize)]
struct SearchCoursesQuery {
    #[serde(rename = "type")]
    type_: Option<String>,
    credit: Option<i64>,
    teacher: Option<String>,
    period: Option<i64>,
    day_of_week: Option<DayOfWeek>,
    keywords: Option<String>,
    page: Option<String>,
}

// 科目検索
async fn search_courses(
    pool: web::Data<sqlx::MySqlPool>,
    params: web::Query<SearchCoursesQuery>,
    request: actix_web::HttpRequest,
) -> actix_web::Result<HttpResponse> {
    let query = concat!(
        "SELECT `courses`.`id`, `courses`.`code`, `courses`.`type`, `courses`.`name`, `courses`.`description`, `courses`.`credit`, `courses`.`period`, `courses`.`day_of_week`, `courses`.`keywords`, `users`.`name` AS `teacher`",
        " FROM `courses` JOIN `users` ON `courses`.`teacher_id` = `users`.`id`",
        " WHERE 1=1",
        );
    let mut condition = String::new();
    let mut args = sqlx::mysql::MySqlArguments::default();

    // MEMO: 検索条件はtype, credit, teacher, period, day_of_weekの完全一致とname, keywordsの部分一致
    // 無効な検索条件はエラーを返さず無視して良い

    if let Some(ref course_type) = params.type_ {
        condition.push_str(" AND `courses`.`type` = ?");
        args.add(course_type);
    }

    if let Some(credit) = params.credit {
        if credit > 0 {
            condition.push_str(" AND `courses`.`credit` = ?");
            args.add(credit);
        }
    }

    if let Some(ref teacher) = params.teacher {
        condition.push_str(" AND `users`.`name` = ?");
        args.add(teacher);
    }

    if let Some(period) = params.period {
        if period > 0 {
            condition.push_str(" AND `courses`.`period` = ?");
            args.add(period);
        }
    }

    if let Some(ref day_of_week) = params.day_of_week {
        condition.push_str(" AND `courses`.`day_of_week` = ?");
        args.add(day_of_week);
    }

    if let Some(ref keywords) = params.keywords {
        let arr = keywords.split(' ').collect::<Vec<_>>();
        let mut name_condition = String::new();
        for keyword in &arr {
            name_condition.push_str(" AND `courses`.`name` LIKE ?");
            args.add(format!("%{}%", keyword));
        }
        let mut keywords_condition = String::new();
        for keyword in arr {
            keywords_condition.push_str(" AND `courses`.`keywords` LIKE ?");
            args.add(format!("%{}%", keyword));
        }
        condition.push_str(&format!(
            " AND ((1=1{}) OR (1=1{}))",
            name_condition, keywords_condition
        ));
    }

    condition.push_str(" ORDER BY `courses`.`code`");

    // MEMO: ページングの初期実装はページ番号形式
    let page = if let Some(ref page_str) = params.page {
        match page_str.parse() {
            Ok(page) if page > 0 => page,
            _ => return Err(actix_web::error::ErrorBadRequest("Invalid page")),
        }
    } else {
        1
    };
    let limit = 20;
    let offset = limit * (page - 1);

    // limitより多く上限を設定し、実際にlimitより多くレコードが取得できた場合は次のページが存在する
    condition.push_str(" LIMIT ? OFFSET ?");
    args.add(limit + 1);
    args.add(offset);

    // 結果が0件の時は空配列を返却
    let mut res: Vec<GetCourseDetailResponse> =
        sqlx::query_as_with(&format!("{}{}", query, condition), args)
            .fetch_all(pool.as_ref())
            .await
            .map_err(SqlxError)?;

    let mut u = url::Url::parse(&request.uri().to_string())
        .map_err(actix_web::error::ErrorInternalServerError)?;
    let mut links = Vec::new();
    if page > 1 {
        u.query_pairs_mut()
            .clear()
            .append_pair("page", &format!("{}", page - 1));
        links.push(format!("<{}>; rel=\"prev\"", u));
    }
    if res.len() as i64 > limit {
        u.query_pairs_mut()
            .clear()
            .append_pair("page", &format!("{}", page + 1));
        links.push(format!("<{}>; rel=\"next\"", u));
    }

    if res.len() as i64 == limit + 1 {
        res.truncate(res.len() - 1);
    }

    let mut builder = HttpResponse::Ok();
    if !links.is_empty() {
        builder.insert_header((actix_web::http::header::LINK, links.join(",")));
    }
    Ok(builder.json(res))
}

#[derive(Debug, serde::Serialize, sqlx::FromRow)]
struct GetCourseDetailResponse {
    id: UuidInDb,
    code: String,
    #[serde(rename = "type")]
    #[sqlx(rename = "type")]
    type_: String,
    name: String,
    description: String,
    credit: u8,
    period: u8,
    day_of_week: DayOfWeek,
    teacher: String,
    keywords: String,
}

// 科目詳細の取得
async fn get_course_detail(
    pool: web::Data<sqlx::MySqlPool>,
    course_id: web::Path<(String,)>,
) -> actix_web::Result<HttpResponse> {
    let course_id = uuid::Uuid::parse_str(&course_id.0)
        .map_err(|_| actix_web::error::ErrorBadRequest("Invalid courseID."))?;

    let res: Option<GetCourseDetailResponse> = sqlx::query_as(concat!(
        "SELECT `courses`.`id`, `courses`.`code`, `courses`.`type`, `courses`.`name`, `courses`.`description`, `courses`.`credit`, `courses`.`period`, `courses`.`day_of_week`, `courses`.`keywords`, `users`.`name` AS `teacher`",
        " FROM `courses`",
        " JOIN `users` ON `courses`.`teacher_id` = `users`.`id`",
        " WHERE `courses`.`id` = ?",
    ))
    .bind(UuidInDb(course_id))
    .fetch_optional(pool.as_ref())
    .await
    .map_err(SqlxError)?;

    if let Some(res) = res {
        Ok(HttpResponse::Ok().json(res))
    } else {
        Err(actix_web::error::ErrorNotFound("No such course."))
    }
}

#[derive(Debug, serde::Deserialize)]
struct AddCourseRequest {
    code: String,
    #[serde(rename = "type")]
    type_: CourseType,
    name: String,
    description: String,
    credit: i64,
    period: i64,
    day_of_week: DayOfWeek,
    keywords: String,
}

#[derive(Debug, serde::Serialize)]
struct AddCourseResponse {
    id: uuid::Uuid,
}

// 新規科目登録
async fn add_course(
    pool: web::Data<sqlx::MySqlPool>,
    session: actix_session::Session,
    req: web::Json<AddCourseRequest>,
) -> actix_web::Result<HttpResponse> {
    let (user_id, _) = get_user_id(session)?;

    let mut tx = pool.begin().await.map_err(SqlxError)?;
    let course_id = uuid::Uuid::new_v4();
    sqlx::query("INSERT INTO `courses` (`id`, `code`, `type`, `name`, `description`, `credit`, `period`, `day_of_week`, `teacher_id`, `keywords`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
        .bind(UuidInDb(course_id))
        .bind(&req.code)
        .bind(&req.type_)
        .bind(&req.name)
        .bind(&req.description)
        .bind(&req.credit)
        .bind(&req.period)
        .bind(&req.day_of_week)
        .bind(UuidInDb(user_id))
        .bind(&req.keywords)
        .execute(&mut tx)
        .await
        .map_err(SqlxError)?;

    tx.commit().await.map_err(SqlxError)?;

    Ok(HttpResponse::Created().json(AddCourseResponse { id: course_id }))
}

#[derive(Debug, serde::Deserialize)]
struct SetCourseStatusRequest {
    status: CourseStatus,
}

// 科目のステータスを変更
async fn set_course_status(
    pool: web::Data<sqlx::MySqlPool>,
    course_id: web::Path<(String,)>,
    req: web::Json<SetCourseStatusRequest>,
) -> actix_web::Result<HttpResponse> {
    let course_id = uuid::Uuid::parse_str(&course_id.0)
        .map_err(|_| actix_web::error::ErrorBadRequest("Invalid courseID."))?;

    let result = sqlx::query("UPDATE `courses` SET `status` = ? WHERE `id` = ?")
        .bind(&req.status)
        .bind(UuidInDb(course_id))
        .execute(pool.as_ref())
        .await
        .map_err(SqlxError)?;

    if result.rows_affected() == 0 {
        Err(actix_web::error::ErrorNotFound("No such course."))
    } else {
        Ok(HttpResponse::Ok().finish())
    }
}

#[derive(Debug, sqlx::FromRow)]
struct ClassWithSubmitted {
    id: UuidInDb,
    course_id: UuidInDb,
    part: u8,
    title: String,
    description: String,
    submission_closed: bool,
    submitted: bool,
}

#[derive(Debug, serde::Serialize)]
struct GetClassResponse {
    id: uuid::Uuid,
    part: u8,
    title: String,
    description: String,
    submission_closed: bool,
    submitted: bool,
}

// 科目に紐づくクラス一覧の取得
async fn get_classes(
    pool: web::Data<sqlx::MySqlPool>,
    session: actix_session::Session,
    course_id: web::Path<(String,)>,
) -> actix_web::Result<HttpResponse> {
    let (user_id, _) = get_user_id(session)?;

    let course_id = uuid::Uuid::parse_str(&course_id.0)
        .map_err(|_| actix_web::error::ErrorBadRequest("Invalid courseID."))?;

    let count: i64 = sqlx::query_scalar("SELECT COUNT(*) FROM `courses` WHERE `id` = ?")
        .bind(UuidInDb(course_id))
        .fetch_one(pool.as_ref())
        .await
        .map_err(SqlxError)?;
    if count == 0 {
        return Err(actix_web::error::ErrorNotFound("No such course."));
    }

    // MEMO: N+1にしても良い
    let classes: Vec<ClassWithSubmitted> = sqlx::query_as(concat!(
        "SELECT `classes`.*, `submissions`.`user_id` IS NOT NULL AS `submitted`",
        " FROM `classes`",
        " LEFT JOIN `submissions` ON `classes`.`id` = `submissions`.`class_id` AND `submissions`.`user_id` = ?",
        " WHERE `classes`.`course_id` = ?",
        " ORDER BY `classes`.`part`",
    ))
    .bind(UuidInDb(user_id))
    .bind(UuidInDb(course_id))
    .fetch_all(pool.as_ref())
    .await
    .map_err(SqlxError)?;

    // 結果が0件の時は空配列を返却
    let res = classes
        .into_iter()
        .map(|class| GetClassResponse {
            id: class.id.0,
            part: class.part,
            title: class.title,
            description: class.description,
            submission_closed: class.submission_closed,
            submitted: class.submitted,
        })
        .collect::<Vec<_>>();

    Ok(HttpResponse::Ok().json(res))
}

#[derive(Debug, serde::Deserialize)]
struct AssignmentPath {
    course_id: String,
    class_id: String,
}

// 課題ファイルのアップロード
async fn submit_assignment(
    pool: web::Data<sqlx::MySqlPool>,
    session: actix_session::Session,
    path: web::Path<AssignmentPath>,
    mut payload: actix_multipart::Multipart,
) -> actix_web::Result<HttpResponse> {
    let (user_id, _) = get_user_id(session)?;

    let course_id = uuid::Uuid::parse_str(&path.course_id)
        .map_err(|_| actix_web::error::ErrorBadRequest("Invalid courseID."))?;
    let class_id = uuid::Uuid::parse_str(&path.class_id)
        .map_err(|_| actix_web::error::ErrorBadRequest("Invalid classID."))?;

    let mut tx = pool.begin().await.map_err(SqlxError)?;
    let status: Option<CourseStatus> =
        sqlx::query_scalar("SELECT `status` FROM `courses` WHERE `id` = ? FOR SHARE")
            .bind(UuidInDb(course_id))
            .fetch_optional(&mut tx)
            .await
            .map_err(SqlxError)?;
    if let Some(status) = status {
        if status != CourseStatus::InProgress {
            return Err(actix_web::error::ErrorBadRequest(
                "This course is not in progress.",
            ));
        }
    } else {
        return Err(actix_web::error::ErrorBadRequest("No such course."));
    }

    let registration_count: i64 = sqlx::query_scalar(
        "SELECT COUNT(*) FROM `registrations` WHERE `user_id` = ? AND `course_id` = ?",
    )
    .bind(UuidInDb(user_id))
    .bind(UuidInDb(course_id))
    .fetch_one(&mut tx)
    .await
    .map_err(SqlxError)?;
    if registration_count == 0 {
        return Err(actix_web::error::ErrorBadRequest(
            "You have not taken this course.",
        ));
    }

    let submission_closed: Option<bool> =
        sqlx::query_scalar("SELECT `submission_closed` FROM `classes` WHERE `id` = ? FOR SHARE")
            .bind(UuidInDb(class_id))
            .fetch_optional(&mut tx)
            .await
            .map_err(SqlxError)?;
    if let Some(submission_closed) = submission_closed {
        if submission_closed {
            return Err(actix_web::error::ErrorBadRequest(
                "Submission has been closed for this class.",
            ));
        }
    } else {
        return Err(actix_web::error::ErrorBadRequest("No such class."));
    }

    let mut file = None;
    while let Some(field) = payload.next().await {
        let field = field.map_err(|_| actix_web::error::ErrorBadRequest("Invalid file."))?;
        if let Some(content_disposition) = field.content_disposition() {
            if let Some(name) = content_disposition.get_name() {
                if name == "file" {
                    file = Some(field);
                    break;
                }
            }
        }
    }
    if file.is_none() {
        return Err(actix_web::error::ErrorBadRequest("Invalid file."));
    }
    let mut file = file.unwrap();

    let result = sqlx::query(
        "INSERT INTO `submissions` (`user_id`, `class_id`, `file_name`) VALUES (?, ?, ?)",
    )
    .bind(UuidInDb(user_id))
    .bind(UuidInDb(class_id))
    .bind(file.content_disposition().unwrap().get_filename())
    .execute(&mut tx)
    .await;
    if let Err(sqlx::Error::Database(ref db_error)) = result {
        if let Some(mysql_error) = db_error.try_downcast_ref::<sqlx::mysql::MySqlDatabaseError>() {
            if mysql_error.number() == MYSQL_ERR_NUM_DUPLICATE_ENTRY {
                return Err(actix_web::error::ErrorBadRequest(
                    "You have already submitted to this assignment.",
                ));
            }
        }
    }
    result.map_err(SqlxError)?;

    let dst = tokio::fs::File::create(format!("{}{}-{}", ASSIGNMENTS_DIRECTORY, class_id, user_id))
        .await?;
    let mut writer = tokio::io::BufWriter::new(dst);
    while let Some(chunk) = file.next().await {
        let mut chunk = chunk?;
        writer.write_buf(&mut chunk).await?;
    }
    writer.shutdown().await?;

    tx.commit().await.map_err(SqlxError)?;

    Ok(HttpResponse::NoContent().finish())
}

#[derive(Debug, serde::Deserialize)]
struct Score {
    user_code: String,
    score: i64,
}

async fn register_scores(
    pool: web::Data<sqlx::MySqlPool>,
    path: web::Path<AssignmentPath>,
    req: web::Json<Vec<Score>>,
) -> actix_web::Result<HttpResponse> {
    let class_id = uuid::Uuid::parse_str(&path.class_id)
        .map_err(|_| actix_web::error::ErrorBadRequest("Invalid classID."))?;

    let mut tx = pool.begin().await.map_err(SqlxError)?;

    let submission_closed: Option<bool> =
        sqlx::query_scalar("SELECT `submission_closed` FROM `classes` WHERE `id` = ? FOR SHARE")
            .bind(UuidInDb(class_id))
            .fetch_optional(&mut tx)
            .await
            .map_err(SqlxError)?;
    if let Some(submission_closed) = submission_closed {
        if !submission_closed {
            return Err(actix_web::error::ErrorBadRequest(
                "This assignment is not closed yet.",
            ));
        }
    } else {
        return Err(actix_web::error::ErrorBadRequest("No such class."));
    }

    for score in req.into_inner() {
        sqlx::query("UPDATE `submissions` JOIN `users` ON `users`.`id` = `submissions`.`user_id` SET `score` = ? WHERE `users`.`code` = ? AND `class_id` = ?")
            .bind(&score.score)
            .bind(&score.user_code)
            .bind(UuidInDb(class_id))
            .execute(&mut tx)
            .await
            .map_err(SqlxError)?;
    }

    tx.commit().await.map_err(SqlxError)?;

    Ok(HttpResponse::NoContent().finish())
}

#[derive(Debug, sqlx::FromRow)]
struct Submission {
    user_id: UuidInDb,
    user_code: String,
    file_name: String,
}

// 提出済みの課題ファイルをzip形式で一括ダウンロード
async fn download_submitted_assignments(
    pool: web::Data<sqlx::MySqlPool>,
    path: web::Path<AssignmentPath>,
) -> actix_web::Result<actix_files::NamedFile> {
    let course_id = uuid::Uuid::parse_str(&path.course_id)
        .map_err(|_| actix_web::error::ErrorBadRequest("Invalid courseID."))?;
    let class_id = uuid::Uuid::parse_str(&path.class_id)
        .map_err(|_| actix_web::error::ErrorBadRequest("Invalid classID."))?;

    let mut tx = pool.begin().await.map_err(SqlxError)?;
    let submission_closed: Option<bool> =
        sqlx::query_scalar("SELECT `submission_closed` FROM `classes` WHERE `id` = ? FOR UPDATE")
            .bind(UuidInDb(class_id))
            .fetch_optional(&mut tx)
            .await
            .map_err(SqlxError)?;
    if submission_closed.is_none() {
        return Err(actix_web::error::ErrorBadRequest("No such class."));
    }
    let submissions: Vec<Submission> = sqlx::query_as(concat!(
        "SELECT `submissions`.`user_id`, `users`.`code` AS `user_code`",
        " FROM `submissions`",
        " JOIN `users` ON `users`.`id` = `submissions`.`user_id`",
        " WHERE `class_id` = ? FOR SHARE",
    ))
    .bind(UuidInDb(class_id))
    .fetch_all(&mut tx)
    .await
    .map_err(SqlxError)?;

    let zip_file_path = format!("{}{}.zip", ASSIGNMENTS_DIRECTORY, class_id);
    create_submissions_zip(&zip_file_path, &class_id, &submissions).await?;

    sqlx::query("UPDATE `classes` SET `submission_closed` = true WHERE `id` = ?")
        .bind(UuidInDb(class_id))
        .execute(&mut tx)
        .await
        .map_err(SqlxError)?;

    tx.commit().await.map_err(SqlxError)?;

    Ok(actix_files::NamedFile::open(&zip_file_path)?)
}

async fn create_submissions_zip(
    zip_file_path: &str,
    class_id: &uuid::Uuid,
    submissions: &[Submission],
) -> std::io::Result<()> {
    let tmp_dir = format!("{}{}/", ASSIGNMENTS_DIRECTORY, class_id);
    tokio::process::Command::new("mkdir")
        .arg(&tmp_dir)
        .status()
        .await?;

    // ファイル名を指定の形式に変更
    for submission in submissions {
        tokio::process::Command::new("cp")
            .arg(&format!(
                "{}{}-{}",
                ASSIGNMENTS_DIRECTORY, class_id, submission.user_id.0
            ))
            .arg(&format!("{}{}", tmp_dir, submission.user_code))
            .status()
            .await?;
    }

    // -i 'tmpDir/*': 空zipを許す
    tokio::process::Command::new("zip")
        .arg("-j")
        .arg("-r")
        .arg(zip_file_path)
        .arg(&tmp_dir)
        .arg("-i")
        .arg(&format!("{}*", tmp_dir))
        .status()
        .await?;
    Ok(())
}

#[derive(Debug, serde::Deserialize)]
struct AddClassRequest {
    part: u8,
    title: String,
    description: String,
    created_at: i64,
}

#[derive(Debug, serde::Serialize)]
struct AddClassResponse {
    class_id: uuid::Uuid,
    announcement_id: uuid::Uuid,
}

// 新規クラス(&課題)追加
async fn add_class(
    pool: web::Data<sqlx::MySqlPool>,
    course_id: web::Path<(String,)>,
    req: web::Json<AddClassRequest>,
) -> actix_web::Result<HttpResponse> {
    let course_id = uuid::Uuid::parse_str(&course_id.0)
        .map_err(|_| actix_web::error::ErrorBadRequest("Invalid courseID."))?;

    let count: i64 = sqlx::query_scalar("SELECT COUNT(*) FROM `courses` WHERE `id` = ?")
        .bind(UuidInDb(course_id))
        .fetch_one(pool.as_ref())
        .await
        .map_err(SqlxError)?;
    if count == 0 {
        return Err(actix_web::error::ErrorNotFound("No such course."));
    }

    let mut tx = pool.begin().await.map_err(SqlxError)?;

    let class_id = uuid::Uuid::new_v4();
    sqlx::query("INSERT INTO `classes` (`id`, `course_id`, `part`, `title`, `description`) VALUES (?, ?, ?, ?, ?)")
        .bind(UuidInDb(class_id))
        .bind(UuidInDb(course_id))
        .bind(&req.part)
        .bind(&req.title)
        .bind(&req.description)
        .execute(&mut tx)
        .await
        .map_err(SqlxError)?;

    let announcement_id = uuid::Uuid::new_v4();
    let created_at = chrono::NaiveDateTime::from_timestamp(req.created_at, 0);
    sqlx::query("INSERT INTO `announcements` (`id`, `course_id`, `title`, `message`, `created_at`) VALUES (?, ?, ?, ?, ?)")
        .bind(UuidInDb(announcement_id))
        .bind(UuidInDb(course_id))
        .bind(format!("クラス追加: {}", req.title))
        .bind(format!("クラスが新しく追加されました: {}\n{}", req.title, req.description))
        .bind(&created_at)
        .execute(&mut tx)
        .await
        .map_err(SqlxError)?;

    let targets: Vec<User> = sqlx::query_as(concat!(
        "SELECT `users`.*",
        " FROM `users`",
        " JOIN `registrations` ON `users`.`id` = `registrations`.`user_id`",
        " WHERE `registrations`.`course_id` = ?",
    ))
    .bind(UuidInDb(course_id))
    .fetch_all(&mut tx)
    .await
    .map_err(SqlxError)?;

    for user in targets {
        sqlx::query(
            "INSERT INTO `unread_announcements` (`announcement_id`, `user_id`) VALUES (?, ?)",
        )
        .bind(UuidInDb(announcement_id))
        .bind(user.id)
        .execute(&mut tx)
        .await
        .map_err(SqlxError)?;
    }

    tx.commit().await.map_err(SqlxError)?;

    let res = AddClassResponse {
        class_id,
        announcement_id,
    };

    Ok(HttpResponse::Created().json(res))
}

#[derive(Debug, sqlx::FromRow)]
struct Announcement {
    id: UuidInDb,
    course_id: UuidInDb,
    course_name: String,
    title: String,
    unread: bool,
    created_at: chrono::NaiveDateTime,
}

#[derive(Debug, serde::Serialize)]
struct GetAnnouncementsResponse {
    unread_count: i64,
    announcements: Vec<AnnouncementResponse>,
}

#[derive(Debug, serde::Serialize)]
struct AnnouncementResponse {
    id: uuid::Uuid,
    course_id: uuid::Uuid,
    course_name: String,
    title: String,
    unread: bool,
    created_at: i64,
}

#[derive(Debug, serde::Deserialize)]
struct GetAnnouncementsQuery {
    course_id: Option<String>,
    page: Option<String>,
}

// お知らせ一覧取得
async fn get_announcement_list(
    pool: web::Data<sqlx::MySqlPool>,
    session: actix_session::Session,
    params: web::Query<GetAnnouncementsQuery>,
    request: actix_web::HttpRequest,
) -> actix_web::Result<HttpResponse> {
    let (user_id, _) = get_user_id(session)?;

    let mut query = concat!(
        "SELECT `announcements`.`id`, `courses`.`id` AS `course_id`, `courses`.`name` AS `course_name`, `announcements`.`title`, `unread_announcements`.`deleted_at` IS NULL AS `unread`, `announcements`.`created_at`",
        " FROM `announcements`",
        " JOIN `courses` ON `announcements`.`course_id` = `courses`.`id`",
        " JOIN `registrations` ON `courses`.`id` = `registrations`.`course_id`",
        " JOIN `unread_announcements` ON `announcements`.`id` = `unread_announcements`.`announcement_id`",
        " WHERE 1=1",
    ).to_owned();
    let mut args = sqlx::mysql::MySqlArguments::default();

    if let Some(ref course_id) = params.course_id {
        query.push_str(" AND `announcements`.`course_id` = ?");
        args.add(course_id);
    }

    query.push_str(concat!(
        " AND `unread_announcements`.`user_id` = ?",
        " AND `registrations`.`user_id` = ?",
        " ORDER BY `announcements`.`created_at` DESC",
        " LIMIT ? OFFSET ?",
    ));
    args.add(UuidInDb(user_id));
    args.add(UuidInDb(user_id));

    // MEMO: ページングの初期実装はページ番号形式
    let page = if let Some(ref page_str) = params.page {
        match page_str.parse() {
            Ok(page) if page > 0 => page,
            _ => return Err(actix_web::error::ErrorBadRequest("Invalid page.")),
        }
    } else {
        1
    };
    let limit = 20;
    let offset = limit * (page - 1);
    // limitより多く上限を設定し、実際にlimitより多くレコードが取得できた場合は次のページが存在する
    args.add(limit + 1);
    args.add(offset);

    let mut announcements: Vec<Announcement> = sqlx::query_as_with(&query, args)
        .fetch_all(pool.as_ref())
        .await
        .map_err(SqlxError)?;

    let unread_count: i64 = sqlx::query_scalar(
        "SELECT COUNT(*) FROM `unread_announcements` WHERE `user_id` = ? AND `deleted_at` IS NULL",
    )
    .bind(UuidInDb(user_id))
    .fetch_one(pool.as_ref())
    .await
    .map_err(SqlxError)?;

    let mut u = url::Url::parse(&request.uri().to_string())
        .map_err(actix_web::error::ErrorInternalServerError)?;
    let mut links = Vec::new();
    if page > 1 {
        u.query_pairs_mut()
            .clear()
            .append_pair("page", &format!("{}", page - 1));
        links.push(format!("<{}>; rel=\"prev\"", u));
    }
    if announcements.len() as i64 > limit {
        u.query_pairs_mut()
            .clear()
            .append_pair("page", &format!("{}", page + 1));
        links.push(format!("<{}>; rel=\"next\"", u));
    }

    if announcements.len() as i64 == limit + 1 {
        announcements.truncate(announcements.len() - 1);
    }

    // 対象になっているお知らせが0件の時は空配列を返却
    let announcements_res = announcements
        .into_iter()
        .map(|announcement| AnnouncementResponse {
            id: announcement.id.0,
            course_id: announcement.course_id.0,
            course_name: announcement.course_name,
            title: announcement.title,
            unread: announcement.unread,
            created_at: announcement.created_at.timestamp(),
        })
        .collect::<Vec<_>>();

    let mut builder = HttpResponse::Ok();
    if !links.is_empty() {
        builder.insert_header((actix_web::http::header::LINK, links.join(",")));
    }
    Ok(builder.json(GetAnnouncementsResponse {
        unread_count,
        announcements: announcements_res,
    }))
}

#[derive(Debug, sqlx::FromRow)]
struct AnnouncementDetail {
    id: UuidInDb,
    course_id: UuidInDb,
    course_name: String,
    title: String,
    message: String,
    unread: bool,
    created_at: chrono::NaiveDateTime,
}

#[derive(Debug, serde::Serialize)]
struct GetAnnouncementDetailResponse {
    id: uuid::Uuid,
    course_id: uuid::Uuid,
    course_name: String,
    title: String,
    message: String,
    unread: bool,
    created_at: i64,
}

async fn get_announcement_detail(
    pool: web::Data<sqlx::MySqlPool>,
    session: actix_session::Session,
    announcement_id: web::Path<(String,)>,
) -> actix_web::Result<HttpResponse> {
    let (user_id, _) = get_user_id(session)?;

    let announcement_id = uuid::Uuid::parse_str(&announcement_id.0)
        .map_err(|_| actix_web::error::ErrorBadRequest("Invalid announcementID."))?;

    let announcement: Option<AnnouncementDetail> = sqlx::query_as(concat!(
            "SELECT `announcements`.`id`, `courses`.`id` AS `course_id`, `courses`.`name` AS `course_name`, `announcements`.`title`, `announcements`.`message`, `unread_announcements`.`deleted_at` IS NULL AS `unread`, `announcements`.`created_at`",
            " FROM `announcements`",
            " JOIN `courses` ON `courses`.`id` = `announcements`.`course_id`",
            " JOIN `unread_announcements` ON `unread_announcements`.`announcement_id` = `announcements`.`id`",
            " WHERE `announcements`.`id` = ?",
            " AND `unread_announcements`.`user_id` = ?",
    ))
        .bind(UuidInDb(announcement_id))
        .bind(UuidInDb(user_id))
        .fetch_optional(pool.as_ref())
        .await
        .map_err(SqlxError)?;
    if announcement.is_none() {
        return Err(actix_web::error::ErrorNotFound("No such announcement."));
    }
    let announcement = announcement.unwrap();

    let registration_count: i64 = sqlx::query_scalar(
        "SELECT COUNT(*) FROM `registrations` WHERE `course_id` = ? AND `user_id` = ?",
    )
    .bind(&announcement.course_id)
    .bind(UuidInDb(user_id))
    .fetch_one(pool.as_ref())
    .await
    .map_err(SqlxError)?;
    if registration_count == 0 {
        return Err(actix_web::error::ErrorNotFound("No such announcement."));
    }

    sqlx::query("UPDATE `unread_announcements` SET `deleted_at` = NOW() WHERE `announcement_id` = ? AND `user_id` = ?")
        .bind(UuidInDb(announcement_id))
        .bind(UuidInDb(user_id))
        .execute(pool.as_ref())
        .await
        .map_err(SqlxError)?;

    Ok(HttpResponse::Ok().json(GetAnnouncementDetailResponse {
        id: announcement.id.0,
        course_id: announcement.course_id.0,
        course_name: announcement.course_name,
        title: announcement.title,
        message: announcement.message,
        unread: announcement.unread,
        created_at: announcement.created_at.timestamp(),
    }))
}

#[derive(Debug, serde::Deserialize)]
struct AddAnnouncementRequest {
    course_id: uuid::Uuid,
    title: String,
    message: String,
    created_at: i64,
}

#[derive(Debug, serde::Serialize)]
struct AddAnnouncementResponse {
    id: uuid::Uuid,
}

// 新規お知らせ追加
async fn add_announcement(
    pool: web::Data<sqlx::MySqlPool>,
    req: web::Json<AddAnnouncementRequest>,
) -> actix_web::Result<HttpResponse> {
    let count: i64 = sqlx::query_scalar("SELECT COUNT(*) FROM `courses` WHERE `id` = ?")
        .bind(UuidInDb(req.course_id))
        .fetch_one(pool.as_ref())
        .await
        .map_err(SqlxError)?;
    if count == 0 {
        return Err(actix_web::error::ErrorNotFound("No such course."));
    }

    let mut tx = pool.begin().await.map_err(SqlxError)?;

    let announcement_id = uuid::Uuid::new_v4();
    let created_at = chrono::NaiveDateTime::from_timestamp(req.created_at, 0);
    sqlx::query("INSERT INTO `announcements` (`id`, `course_id`, `title`, `message`, `created_at`) VALUES (?, ?, ?, ?, ?)")
        .bind(UuidInDb(announcement_id))
        .bind(UuidInDb(req.course_id))
        .bind(&req.title)
        .bind(&req.message)
        .bind(&created_at)
        .execute(&mut tx)
        .await
        .map_err(SqlxError)?;

    let targets: Vec<User> = sqlx::query_as(concat!(
        "SELECT `users`.* FROM `users`",
        " JOIN `registrations` ON `users`.`id` = `registrations`.`user_id`",
        " WHERE `registrations`.`course_id` = ?",
    ))
    .bind(UuidInDb(req.course_id))
    .fetch_all(&mut tx)
    .await
    .map_err(SqlxError)?;

    for user in targets {
        sqlx::query(
            "INSERT INTO `unread_announcements` (`announcement_id`, `user_id`) VALUES (?, ?)",
        )
        .bind(UuidInDb(announcement_id))
        .bind(user.id)
        .execute(&mut tx)
        .await
        .map_err(SqlxError)?;
    }

    tx.commit().await.map_err(SqlxError)?;

    let res = AddAnnouncementResponse {
        id: announcement_id,
    };

    Ok(HttpResponse::Created().json(res))
}
