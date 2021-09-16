use actix_web::web;
use actix_web::HttpResponse;
use futures::StreamExt as _;
use futures::TryStreamExt as _;
use num_traits::ToPrimitive as _;
use sqlx::Arguments as _;
use sqlx::Executor as _;
use tokio::io::AsyncWriteExt as _;

const SQL_DIRECTORY: &str = "../sql/";
const ASSIGNMENTS_DIRECTORY: &str = "../assignments/";
const INIT_DATA_DIRECTORY: &str = "../data/";
const SESSION_NAME: &str = "isucholar_rust";
const MYSQL_ERR_NUM_DUPLICATE_ENTRY: u16 = 1062;

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    env_logger::Builder::from_env(env_logger::Env::default().default_filter_or("info,sqlx=warn"))
        .init();

    let mysql_config = sqlx::mysql::MySqlConnectOptions::new()
        .host(
            &std::env::var("MYSQL_HOSTNAME")
                .ok()
                .unwrap_or_else(|| "127.0.0.1".to_owned()),
        )
        .port(
            std::env::var("MYSQL_PORT")
                .ok()
                .and_then(|port_str| port_str.parse().ok())
                .unwrap_or(3306),
        )
        .username(
            &std::env::var("MYSQL_USER")
                .ok()
                .unwrap_or_else(|| "isucon".to_owned()),
        )
        .password(
            &std::env::var("MYSQL_PASS")
                .ok()
                .unwrap_or_else(|| "isucon".to_owned()),
        )
        .database(
            &std::env::var("MYSQL_DATABASE")
                .ok()
                .unwrap_or_else(|| "isucholar".to_owned()),
        )
        .ssl_mode(sqlx::mysql::MySqlSslMode::Disabled);
    let pool = sqlx::mysql::MySqlPoolOptions::new()
        .max_connections(10)
        .after_connect(|conn| {
            Box::pin(async move {
                conn.execute("set time_zone = '+00:00'").await?;
                Ok(())
            })
        })
        .connect_with(mysql_config)
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

        let courses_api = web::scope("/courses")
            .route("", web::get().to(search_courses))
            .service(
                web::resource("")
                    .guard(actix_web::guard::Post())
                    .wrap(isucholar::middleware::IsAdmin)
                    .to(add_course),
            )
            .route("/{course_id}", web::get().to(get_course_detail))
            .service(
                web::resource("/{course_id}/status")
                    .guard(actix_web::guard::Put())
                    .wrap(isucholar::middleware::IsAdmin)
                    .to(set_course_status),
            )
            .route("/{course_id}/classes", web::get().to(get_classes))
            .service(
                web::resource("/{course_id}/classes")
                    .guard(actix_web::guard::Post())
                    .wrap(isucholar::middleware::IsAdmin)
                    .to(add_class),
            )
            .route(
                "/{course_id}/classes/{class_id}/assignments",
                web::post().to(submit_assignment),
            )
            .service(
                web::resource("/{course_id}/classes/{class_id}/assignments/scores")
                    .guard(actix_web::guard::Put())
                    .wrap(isucholar::middleware::IsAdmin)
                    .to(register_scores),
            )
            .service(
                web::resource("/{course_id}/classes/{class_id}/assignments/export")
                    .guard(actix_web::guard::Get())
                    .wrap(isucholar::middleware::IsAdmin)
                    .to(download_submitted_assignments),
            );

        let announcements_api = web::scope("/announcements")
            .route("", web::get().to(get_announcement_list))
            .service(
                web::resource("")
                    .guard(actix_web::guard::Post())
                    .wrap(isucholar::middleware::IsAdmin)
                    .to(add_announcement),
            )
            .route("/{announcement_id}", web::get().to(get_announcement_detail));

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
            .route("/logout", web::post().to(logout))
            .service(
                web::scope("/api")
                    .wrap(isucholar::middleware::IsLoggedIn)
                    .service(users_api)
                    .service(courses_api)
                    .service(announcements_api),
            )
    });
    if let Some(l) = listenfd::ListenFd::from_env().take_tcp_listener(0)? {
        server.listen(l)?
    } else {
        server.bind((
            "0.0.0.0",
            std::env::var("PORT")
                .ok()
                .and_then(|port_str| port_str.parse().ok())
                .unwrap_or(7000),
        ))?
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
struct InitializeResponse {
    language: &'static str,
}

// POST /initialize 初期化エンドポイント
async fn initialize(pool: web::Data<sqlx::MySqlPool>) -> actix_web::Result<HttpResponse> {
    let files = ["1_schema.sql", "2_init.sql", "3_sample.sql"];
    for file in files {
        let data = tokio::fs::read_to_string(format!("{}{}", SQL_DIRECTORY, file)).await?;
        let mut stream = pool.execute_many(data.as_str());
        while let Some(result) = stream.next().await {
            result.map_err(SqlxError)?;
        }
    }

    if !tokio::process::Command::new("rm")
        .stdin(std::process::Stdio::null())
        .stdout(std::process::Stdio::null())
        .stderr(std::process::Stdio::null())
        .arg("-rf")
        .arg(ASSIGNMENTS_DIRECTORY)
        .status()
        .await?
        .success()
    {
        return Err(actix_web::error::ErrorInternalServerError(""));
    }
    if !tokio::process::Command::new("cp")
        .stdin(std::process::Stdio::null())
        .stdout(std::process::Stdio::null())
        .stderr(std::process::Stdio::null())
        .arg("-r")
        .arg(INIT_DATA_DIRECTORY)
        .arg(ASSIGNMENTS_DIRECTORY)
        .status()
        .await?
        .success()
    {
        return Err(actix_web::error::ErrorInternalServerError(""));
    }

    Ok(HttpResponse::Ok().json(InitializeResponse { language: "rust" }))
}

fn get_user_info(session: actix_session::Session) -> actix_web::Result<(String, String, bool)> {
    let user_id = session.get("userID")?;
    if user_id.is_none() {
        return Err(actix_web::error::ErrorInternalServerError(
            "failed to get userID from session",
        ));
    }
    let user_name = session.get("userName")?;
    if user_name.is_none() {
        return Err(actix_web::error::ErrorInternalServerError(
            "failed to get userName from session",
        ));
    }
    let is_admin = session.get("isAdmin")?;
    if is_admin.is_none() {
        return Err(actix_web::error::ErrorInternalServerError(
            "failed to get isAdmin from session",
        ));
    }
    Ok((user_id.unwrap(), user_name.unwrap(), is_admin.unwrap()))
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
    id: String,
    code: String,
    name: String,
    hashed_password: Vec<u8>,
    #[sqlx(rename = "type")]
    type_: UserType,
}

#[derive(Debug, PartialEq, Eq, serde::Deserialize)]
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
    Monday,
    Tuesday,
    Wednesday,
    Thursday,
    Friday,
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
            "monday" => Ok(Self::Monday),
            "tuesday" => Ok(Self::Tuesday),
            "wednesday" => Ok(Self::Wednesday),
            "thursday" => Ok(Self::Thursday),
            "friday" => Ok(Self::Friday),
            v => Err(format!("Unknown enum variant: {}", v).into()),
        }
    }
}
impl<'q> sqlx::Encode<'q, sqlx::MySql> for DayOfWeek {
    fn encode_by_ref(&self, buf: &mut Vec<u8>) -> sqlx::encode::IsNull {
        match *self {
            Self::Monday => "monday",
            Self::Tuesday => "tuesday",
            Self::Wednesday => "wednesday",
            Self::Thursday => "thursday",
            Self::Friday => "friday",
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
    id: String,
    code: String,
    #[sqlx(rename = "type")]
    type_: CourseType,
    name: String,
    description: String,
    credit: u8,
    period: u8,
    day_of_week: DayOfWeek,
    teacher_id: String,
    keywords: String,
    status: CourseStatus,
}

// ---------- Public API ----------

#[derive(Debug, serde::Deserialize)]
struct LoginRequest {
    code: String,
    password: String,
}

// POST /login ログイン
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
        &String::from_utf8(user.hashed_password).unwrap(),
    )
    .map_err(actix_web::error::ErrorInternalServerError)?
    {
        return Err(actix_web::error::ErrorUnauthorized(
            "Code or Password is wrong.",
        ));
    }

    if let Some(user_id) = session.get::<String>("userID")? {
        if user_id == user.id {
            return Err(actix_web::error::ErrorBadRequest(
                "You are already logged in.",
            ));
        }
    }

    session.insert("userID", user.id)?;
    session.insert("userName", user.name)?;
    session.insert("isAdmin", user.type_ == UserType::Teacher)?;
    Ok(HttpResponse::Ok().finish())
}

// POST /logout ログアウト
async fn logout(session: actix_session::Session) -> actix_web::Result<HttpResponse> {
    session.purge();
    Ok(HttpResponse::Ok().finish())
}

// ---------- Users API ----------

#[derive(Debug, serde::Serialize)]
struct GetMeResponse {
    code: String,
    name: String,
    is_admin: bool,
}

// GET /api/users/me 自身の情報を取得
async fn get_me(
    pool: web::Data<sqlx::MySqlPool>,
    session: actix_session::Session,
) -> actix_web::Result<HttpResponse> {
    let (user_id, user_name, is_admin) = get_user_info(session)?;

    let user_code = sqlx::query_scalar("SELECT `code` FROM `users` WHERE `id` = ?")
        .bind(&user_id)
        .fetch_one(pool.as_ref())
        .await
        .map_err(SqlxError)?;

    Ok(HttpResponse::Ok().json(GetMeResponse {
        code: user_code,
        name: user_name,
        is_admin,
    }))
}

#[derive(Debug, serde::Serialize)]
struct GetRegisteredCourseResponseContent {
    id: String,
    name: String,
    teacher: String,
    period: u8,
    day_of_week: DayOfWeek,
}

// GET /api/users/me/courses 履修中の科目一覧取得
async fn get_registered_courses(
    pool: web::Data<sqlx::MySqlPool>,
    session: actix_session::Session,
) -> actix_web::Result<HttpResponse> {
    let (user_id, _, _) = get_user_info(session)?;

    let mut tx = pool.begin().await.map_err(SqlxError)?;

    let courses: Vec<Course> = sqlx::query_as(concat!(
        "SELECT `courses`.*",
        " FROM `courses`",
        " JOIN `registrations` ON `courses`.`id` = `registrations`.`course_id`",
        " WHERE `courses`.`status` != ? AND `registrations`.`user_id` = ?",
    ))
    .bind(CourseStatus::Closed)
    .bind(&user_id)
    .fetch_all(&mut tx)
    .await
    .map_err(SqlxError)?;

    // 履修科目が0件の時は空配列を返却
    let mut res = Vec::with_capacity(courses.len());
    for course in courses {
        let teacher: User = isucholar::db::fetch_one_as(
            sqlx::query_as("SELECT * FROM `users` WHERE `id` = ?").bind(&course.teacher_id),
            &mut tx,
        )
        .await
        .map_err(SqlxError)?;

        res.push(GetRegisteredCourseResponseContent {
            id: course.id,
            name: course.name,
            teacher: teacher.name,
            period: course.period,
            day_of_week: course.day_of_week,
        });
    }

    tx.commit().await.map_err(SqlxError)?;

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
    not_registrable_status: Vec<String>,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    schedule_conflict: Vec<String>,
}

// PUT /api/users/me/courses 履修登録
async fn register_courses(
    pool: web::Data<sqlx::MySqlPool>,
    session: actix_session::Session,
    req: web::Json<Vec<RegisterCourseRequestContent>>,
) -> actix_web::Result<HttpResponse> {
    let (user_id, _, _) = get_user_info(session)?;

    let mut req = req.into_inner();
    req.sort_by(|x, y| x.id.cmp(&y.id));

    let mut tx = pool.begin().await.map_err(SqlxError)?;

    let mut errors = RegisterCoursesErrorResponse::default();
    let mut newly_added = Vec::new();
    for course_req in req {
        let course: Option<Course> = isucholar::db::fetch_optional_as(
            sqlx::query_as("SELECT * FROM `courses` WHERE `id` = ? FOR SHARE").bind(&course_req.id),
            &mut tx,
        )
        .await
        .map_err(SqlxError)?;
        if course.is_none() {
            errors.course_not_found.push(course_req.id);
            continue;
        }
        let course = course.unwrap();

        if course.status != CourseStatus::Registration {
            errors.not_registrable_status.push(course.id);
            continue;
        }

        // すでに履修登録済みの科目は無視する
        let count: i64 = isucholar::db::fetch_one_scalar(
            sqlx::query_scalar(
                "SELECT COUNT(*) FROM `registrations` WHERE `course_id` = ? AND `user_id` = ?",
            )
            .bind(&course.id)
            .bind(&user_id),
            &mut tx,
        )
        .await
        .map_err(SqlxError)?;
        if count > 0 {
            continue;
        }

        newly_added.push(course);
    }

    let already_registered: Vec<Course> = sqlx::query_as(concat!(
        "SELECT `courses`.*",
        " FROM `courses`",
        " JOIN `registrations` ON `courses`.`id` = `registrations`.`course_id`",
        " WHERE `courses`.`status` != ? AND `registrations`.`user_id` = ?",
    ))
    .bind(CourseStatus::Closed)
    .bind(&user_id)
    .fetch_all(&mut tx)
    .await
    .map_err(SqlxError)?;

    for course1 in &newly_added {
        for course2 in already_registered.iter().chain(newly_added.iter()) {
            if course1.id != course2.id
                && course1.period == course2.period
                && course1.day_of_week == course2.day_of_week
            {
                errors.schedule_conflict.push(course1.id.to_owned());
                break;
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
        sqlx::query("INSERT INTO `registrations` (`course_id`, `user_id`) VALUES (?, ?) ON DUPLICATE KEY UPDATE `course_id` = VALUES(`course_id`), `user_id` = VALUES(`user_id`)")
            .bind(course.id)
            .bind(&user_id)
            .execute(&mut tx)
            .await
            .map_err(SqlxError)?;
    }

    tx.commit().await.map_err(SqlxError)?;

    Ok(HttpResponse::Ok().finish())
}

#[derive(Debug, sqlx::FromRow)]
struct Class {
    id: String,
    course_id: String,
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
    gpa: f64,
    gpa_t_score: f64, // 偏差値
    gpa_avg: f64,     // 平均値
    gpa_max: f64,     // 最大値
    gpa_min: f64,     // 最小値
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
    class_id: String,
    title: String,
    part: u8,
    score: Option<i64>, // 0~100点
    submitters: i64,    // 提出した学生数
}

// GET /api/users/me/grades 成績取得
async fn get_grades(
    pool: web::Data<sqlx::MySqlPool>,
    session: actix_session::Session,
) -> actix_web::Result<HttpResponse> {
    let (user_id, _, _) = get_user_info(session)?;

    // 履修している科目一覧取得
    let registered_courses: Vec<Course> = sqlx::query_as(concat!(
        "SELECT `courses`.*",
        " FROM `registrations`",
        " JOIN `courses` ON `registrations`.`course_id` = `courses`.`id`",
        " WHERE `user_id` = ?",
    ))
    .bind(&user_id)
    .fetch_all(pool.as_ref())
    .await
    .map_err(SqlxError)?;

    // 科目毎の成績計算処理
    let mut course_results = Vec::with_capacity(registered_courses.len());
    let mut my_gpa = 0f64;
    let mut my_credits = 0;
    for course in registered_courses {
        // 講義一覧の取得
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

        // 講義毎の成績計算処理
        let mut class_scores = Vec::with_capacity(classes.len());
        let mut my_total_score = 0;
        for class in classes {
            let submissions_count: i64 =
                sqlx::query_scalar("SELECT COUNT(*) FROM `submissions` WHERE `class_id` = ?")
                    .bind(&class.id)
                    .fetch_one(pool.as_ref())
                    .await
                    .map_err(SqlxError)?;

            let my_score: Option<Option<u8>> = sqlx::query_scalar(concat!(
                "SELECT `submissions`.`score` FROM `submissions`",
                " WHERE `user_id` = ? AND `class_id` = ?"
            ))
            .bind(&user_id)
            .bind(&class.id)
            .fetch_optional(pool.as_ref())
            .await
            .map_err(SqlxError)?;
            if let Some(Some(my_score)) = my_score {
                let my_score = my_score as i64;
                my_total_score += my_score;
                class_scores.push(ClassScore {
                    class_id: class.id,
                    part: class.part,
                    title: class.title,
                    score: Some(my_score),
                    submitters: submissions_count,
                });
            } else {
                class_scores.push(ClassScore {
                    class_id: class.id,
                    part: class.part,
                    title: class.title,
                    score: None,
                    submitters: submissions_count,
                });
            }
        }

        // この科目を履修している学生のtotal_score一覧を取得
        let mut rows = sqlx::query_scalar(concat!(
            "SELECT IFNULL(SUM(`submissions`.`score`), 0) AS `total_score`",
            " FROM `users`",
            " JOIN `registrations` ON `users`.`id` = `registrations`.`user_id`",
            " JOIN `courses` ON `registrations`.`course_id` = `courses`.`id`",
            " LEFT JOIN `classes` ON `courses`.`id` = `classes`.`course_id`",
            " LEFT JOIN `submissions` ON `users`.`id` = `submissions`.`user_id` AND `submissions`.`class_id` = `classes`.`id`",
            " WHERE `courses`.`id` = ?",
            " GROUP BY `users`.`id`",
        ))
        .bind(&course.id)
        .fetch(pool.as_ref());
        let mut totals = Vec::new();
        while let Some(row) = rows.next().await {
            let total_score: sqlx::types::Decimal = row.map_err(SqlxError)?;
            totals.push(total_score.to_i64().unwrap());
        }

        course_results.push(CourseResult {
            name: course.name,
            code: course.code,
            total_score: my_total_score,
            total_score_t_score: isucholar::util::t_score_int(my_total_score, &totals),
            total_score_avg: isucholar::util::average_int(&totals, 0.0),
            total_score_max: isucholar::util::max_int(&totals, 0),
            total_score_min: isucholar::util::min_int(&totals, 0),
            class_scores,
        });

        // 自分のGPA計算
        if course.status == CourseStatus::Closed {
            my_gpa += (my_total_score * course.credit as i64) as f64;
            my_credits += course.credit as i64;
        }
    }
    if my_credits > 0 {
        my_gpa = my_gpa / 100.0 / my_credits as f64;
    }

    // GPAの統計値
    // 一つでも修了した科目がある学生のGPA一覧
    let gpas = {
        let mut rows = sqlx::query_scalar(concat!(
            "SELECT IFNULL(SUM(`submissions`.`score` * `courses`.`credit`), 0) / 100 / `credits`.`credits` AS `gpa`",
            " FROM `users`",
            " JOIN (",
            "     SELECT `users`.`id` AS `user_id`, SUM(`courses`.`credit`) AS `credits`",
            "     FROM `users`",
            "     JOIN `registrations` ON `users`.`id` = `registrations`.`user_id`",
            "     JOIN `courses` ON `registrations`.`course_id` = `courses`.`id` AND `courses`.`status` = ?",
            "     GROUP BY `users`.`id`",
            " ) AS `credits` ON `credits`.`user_id` = `users`.`id`",
            " JOIN `registrations` ON `users`.`id` = `registrations`.`user_id`",
            " JOIN `courses` ON `registrations`.`course_id` = `courses`.`id` AND `courses`.`status` = ?",
            " LEFT JOIN `classes` ON `courses`.`id` = `classes`.`course_id`",
            " LEFT JOIN `submissions` ON `users`.`id` = `submissions`.`user_id` AND `submissions`.`class_id` = `classes`.`id`",
            " WHERE `users`.`type` = ?",
            " GROUP BY `users`.`id`",
        ))
        .bind(CourseStatus::Closed)
        .bind(CourseStatus::Closed)
        .bind(UserType::Student)
        .fetch(pool.as_ref());
        let mut gpas = Vec::new();
        while let Some(row) = rows.next().await {
            let gpa: sqlx::types::Decimal = row.map_err(SqlxError)?;
            gpas.push(gpa.to_f64().unwrap());
        }
        gpas
    };

    Ok(HttpResponse::Ok().json(GetGradeResponse {
        course_results,
        summary: Summary {
            credits: my_credits,
            gpa: my_gpa,
            gpa_t_score: isucholar::util::t_score_f64(my_gpa, &gpas),
            gpa_avg: isucholar::util::average_f64(&gpas, 0.0),
            gpa_max: isucholar::util::max_f64(&gpas, 0.0),
            gpa_min: isucholar::util::min_f64(&gpas, 0.0),
        },
    }))
}

// ---------- Courses API ----------

#[derive(Debug, serde::Deserialize, serde::Serialize)]
struct SearchCoursesQuery {
    #[serde(rename = "type")]
    type_: Option<String>,
    credit: Option<i64>,
    teacher: Option<String>,
    period: Option<i64>,
    day_of_week: Option<DayOfWeek>,
    keywords: Option<String>,
    status: Option<String>,
    page: Option<String>,
}

// GET /api/courses 科目検索
async fn search_courses(
    pool: web::Data<sqlx::MySqlPool>,
    params: web::Query<SearchCoursesQuery>,
    request: actix_web::HttpRequest,
) -> actix_web::Result<HttpResponse> {
    let query = concat!(
        "SELECT `courses`.*, `users`.`name` AS `teacher`",
        " FROM `courses` JOIN `users` ON `courses`.`teacher_id` = `users`.`id`",
        " WHERE 1=1",
    );
    let mut condition = String::new();
    let mut args = sqlx::mysql::MySqlArguments::default();

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

    if let Some(ref status) = params.status {
        condition.push_str(" AND `courses`.`status` = ?");
        args.add(status);
    }

    condition.push_str(" ORDER BY `courses`.`code`");

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
    condition.push_str(" LIMIT ? OFFSET ?");
    args.add(limit + 1);
    args.add(offset);

    // 結果が0件の時は空配列を返却
    let mut res: Vec<GetCourseDetailResponse> =
        sqlx::query_as_with(&format!("{}{}", query, condition), args)
            .fetch_all(pool.as_ref())
            .await
            .map_err(SqlxError)?;

    let uri = request.uri();
    let mut params = params.into_inner();
    let mut links = Vec::new();
    if page > 1 {
        params.page = Some(format!("{}", page - 1));
        links.push(format!(
            "<{}?{}>; rel=\"prev\"",
            uri.path(),
            serde_urlencoded::to_string(&params)?
        ));
    }
    if res.len() as i64 > limit {
        params.page = Some(format!("{}", page + 1));
        links.push(format!(
            "<{}?{}>; rel=\"next\"",
            uri.path(),
            serde_urlencoded::to_string(&params)?
        ));
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
    id: String,
}

// POST /api/courses 新規科目登録
async fn add_course(
    pool: web::Data<sqlx::MySqlPool>,
    session: actix_session::Session,
    req: web::Json<AddCourseRequest>,
) -> actix_web::Result<HttpResponse> {
    let (user_id, _, _) = get_user_info(session)?;

    let course_id = isucholar::util::new_ulid().await;
    let result = sqlx::query("INSERT INTO `courses` (`id`, `code`, `type`, `name`, `description`, `credit`, `period`, `day_of_week`, `teacher_id`, `keywords`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
        .bind(&course_id)
        .bind(&req.code)
        .bind(&req.type_)
        .bind(&req.name)
        .bind(&req.description)
        .bind(&req.credit)
        .bind(&req.period)
        .bind(&req.day_of_week)
        .bind(&user_id)
        .bind(&req.keywords)
        .execute(pool.as_ref())
        .await;
    if let Err(sqlx::Error::Database(ref db_error)) = result {
        if let Some(mysql_error) = db_error.try_downcast_ref::<sqlx::mysql::MySqlDatabaseError>() {
            if mysql_error.number() == MYSQL_ERR_NUM_DUPLICATE_ENTRY {
                let course: Course = sqlx::query_as("SELECT * FROM `courses` WHERE `code` = ?")
                    .bind(&req.code)
                    .fetch_one(pool.as_ref())
                    .await
                    .map_err(SqlxError)?;
                if req.type_ != course.type_
                    || req.name != course.name
                    || req.description != course.description
                    || req.credit != course.credit as i64
                    || req.period != course.period as i64
                    || req.day_of_week != course.day_of_week
                    || req.keywords != course.keywords
                {
                    return Err(actix_web::error::ErrorConflict(
                        "A course with the same code already exists.",
                    ));
                } else {
                    return Ok(HttpResponse::Created().json(AddCourseResponse { id: course.id }));
                }
            }
        }
    }
    result.map_err(SqlxError)?;

    Ok(HttpResponse::Created().json(AddCourseResponse { id: course_id }))
}

#[derive(Debug, serde::Serialize, sqlx::FromRow)]
struct GetCourseDetailResponse {
    id: String,
    code: String,
    #[serde(rename = "type")]
    #[sqlx(rename = "type")]
    type_: String,
    name: String,
    description: String,
    credit: u8,
    period: u8,
    day_of_week: DayOfWeek,
    #[serde(skip)]
    teacher_id: String,
    keywords: String,
    status: CourseStatus,
    teacher: String,
}

// GET /api/courses/{course_id} 科目詳細の取得
async fn get_course_detail(
    pool: web::Data<sqlx::MySqlPool>,
    course_id: web::Path<(String,)>,
) -> actix_web::Result<HttpResponse> {
    let course_id = &course_id.0;

    let res: Option<GetCourseDetailResponse> = sqlx::query_as(concat!(
        "SELECT `courses`.*, `users`.`name` AS `teacher`",
        " FROM `courses`",
        " JOIN `users` ON `courses`.`teacher_id` = `users`.`id`",
        " WHERE `courses`.`id` = ?",
    ))
    .bind(course_id)
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
struct SetCourseStatusRequest {
    status: CourseStatus,
}

// PUT /api/courses/{course_id}/status 科目のステータスを変更
async fn set_course_status(
    pool: web::Data<sqlx::MySqlPool>,
    course_id: web::Path<(String,)>,
    req: web::Json<SetCourseStatusRequest>,
) -> actix_web::Result<HttpResponse> {
    let course_id = &course_id.0;

    let mut tx = pool.begin().await.map_err(SqlxError)?;

    let count: i64 = isucholar::db::fetch_one_scalar(
        sqlx::query_scalar("SELECT COUNT(*) FROM `courses` WHERE `id` = ? FOR UPDATE")
            .bind(course_id),
        &mut tx,
    )
    .await
    .map_err(SqlxError)?;
    if count == 0 {
        return Err(actix_web::error::ErrorNotFound("No such course."));
    }

    sqlx::query("UPDATE `courses` SET `status` = ? WHERE `id` = ?")
        .bind(&req.status)
        .bind(course_id)
        .execute(&mut tx)
        .await
        .map_err(SqlxError)?;

    tx.commit().await.map_err(SqlxError)?;

    Ok(HttpResponse::Ok().finish())
}

#[derive(Debug, sqlx::FromRow)]
struct ClassWithSubmitted {
    id: String,
    course_id: String,
    part: u8,
    title: String,
    description: String,
    submission_closed: bool,
    submitted: bool,
}

#[derive(Debug, serde::Serialize)]
struct GetClassResponse {
    id: String,
    part: u8,
    title: String,
    description: String,
    submission_closed: bool,
    submitted: bool,
}

// GET /api/courses/{course_id}/classes 科目に紐づく講義一覧の取得
async fn get_classes(
    pool: web::Data<sqlx::MySqlPool>,
    session: actix_session::Session,
    course_id: web::Path<(String,)>,
) -> actix_web::Result<HttpResponse> {
    let (user_id, _, _) = get_user_info(session)?;

    let course_id = &course_id.0;

    let mut tx = pool.begin().await.map_err(SqlxError)?;

    let count: i64 = isucholar::db::fetch_one_scalar(
        sqlx::query_scalar("SELECT COUNT(*) FROM `courses` WHERE `id` = ?").bind(course_id),
        &mut tx,
    )
    .await
    .map_err(SqlxError)?;
    if count == 0 {
        return Err(actix_web::error::ErrorNotFound("No such course."));
    }

    let classes: Vec<ClassWithSubmitted> = sqlx::query_as(concat!(
        "SELECT `classes`.*, `submissions`.`user_id` IS NOT NULL AS `submitted`",
        " FROM `classes`",
        " LEFT JOIN `submissions` ON `classes`.`id` = `submissions`.`class_id` AND `submissions`.`user_id` = ?",
        " WHERE `classes`.`course_id` = ?",
        " ORDER BY `classes`.`part`",
    ))
    .bind(&user_id)
    .bind(course_id)
    .fetch_all(&mut tx)
    .await
    .map_err(SqlxError)?;

    tx.commit().await.map_err(SqlxError)?;

    // 結果が0件の時は空配列を返却
    let res = classes
        .into_iter()
        .map(|class| GetClassResponse {
            id: class.id,
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
struct AddClassRequest {
    part: u8,
    title: String,
    description: String,
}

#[derive(Debug, serde::Serialize)]
struct AddClassResponse {
    class_id: String,
}

// POST /api/courses/{course_id}/classes 新規講義(&課題)追加
async fn add_class(
    pool: web::Data<sqlx::MySqlPool>,
    course_id: web::Path<(String,)>,
    req: web::Json<AddClassRequest>,
) -> actix_web::Result<HttpResponse> {
    let course_id = &course_id.0;

    let mut tx = pool.begin().await.map_err(SqlxError)?;

    let course: Option<Course> = isucholar::db::fetch_optional_as(
        sqlx::query_as("SELECT * FROM `courses` WHERE `id` = ? FOR SHARE").bind(course_id),
        &mut tx,
    )
    .await
    .map_err(SqlxError)?;
    if course.is_none() {
        return Err(actix_web::error::ErrorNotFound("No such course."));
    }
    let course = course.unwrap();
    if course.status != CourseStatus::InProgress {
        return Err(actix_web::error::ErrorBadRequest(
            "This course is not in-progress.",
        ));
    }

    let class_id = isucholar::util::new_ulid().await;
    let result = sqlx::query("INSERT INTO `classes` (`id`, `course_id`, `part`, `title`, `description`) VALUES (?, ?, ?, ?, ?)")
        .bind(&class_id)
        .bind(course_id)
        .bind(&req.part)
        .bind(&req.title)
        .bind(&req.description)
        .execute(&mut tx)
        .await;
    if let Err(e) = result {
        let _ = tx.rollback().await;
        if let sqlx::error::Error::Database(ref db_error) = e {
            if let Some(mysql_error) =
                db_error.try_downcast_ref::<sqlx::mysql::MySqlDatabaseError>()
            {
                if mysql_error.number() == MYSQL_ERR_NUM_DUPLICATE_ENTRY {
                    let class: Class = sqlx::query_as(
                        "SELECT * FROM `classes` WHERE `course_id` = ? AND `part` = ?",
                    )
                    .bind(course_id)
                    .bind(&req.part)
                    .fetch_one(pool.as_ref())
                    .await
                    .map_err(SqlxError)?;
                    if req.title != class.title || req.description != class.description {
                        return Err(actix_web::error::ErrorConflict(
                            "A class with the same part already exists.",
                        ));
                    } else {
                        return Ok(
                            HttpResponse::Created().json(AddClassResponse { class_id: class.id })
                        );
                    }
                }
            }
        }
        return Err(SqlxError(e).into());
    }

    tx.commit().await.map_err(SqlxError)?;

    Ok(HttpResponse::Created().json(AddClassResponse { class_id }))
}

#[derive(Debug, serde::Deserialize)]
struct AssignmentPath {
    course_id: String,
    class_id: String,
}

// POST /api/courses/{course_id}/classes/{class_id}/assignments 課題の提出
async fn submit_assignment(
    pool: web::Data<sqlx::MySqlPool>,
    session: actix_session::Session,
    path: web::Path<AssignmentPath>,
    mut payload: actix_multipart::Multipart,
) -> actix_web::Result<HttpResponse> {
    let (user_id, _, _) = get_user_info(session)?;

    let course_id = &path.course_id;
    let class_id = &path.class_id;

    let mut tx = pool.begin().await.map_err(SqlxError)?;
    let status: Option<CourseStatus> = isucholar::db::fetch_optional_scalar(
        sqlx::query_scalar("SELECT `status` FROM `courses` WHERE `id` = ? FOR SHARE")
            .bind(course_id),
        &mut tx,
    )
    .await
    .map_err(SqlxError)?;
    if let Some(status) = status {
        if status != CourseStatus::InProgress {
            return Err(actix_web::error::ErrorBadRequest(
                "This course is not in progress.",
            ));
        }
    } else {
        return Err(actix_web::error::ErrorNotFound("No such course."));
    }

    let registration_count: i64 = isucholar::db::fetch_one_scalar(
        sqlx::query_scalar(
            "SELECT COUNT(*) FROM `registrations` WHERE `user_id` = ? AND `course_id` = ?",
        )
        .bind(&user_id)
        .bind(course_id),
        &mut tx,
    )
    .await
    .map_err(SqlxError)?;
    if registration_count == 0 {
        return Err(actix_web::error::ErrorBadRequest(
            "You have not taken this course.",
        ));
    }

    let submission_closed: Option<bool> = isucholar::db::fetch_optional_scalar(
        sqlx::query_scalar("SELECT `submission_closed` FROM `classes` WHERE `id` = ? FOR SHARE")
            .bind(class_id),
        &mut tx,
    )
    .await
    .map_err(SqlxError)?;
    if let Some(submission_closed) = submission_closed {
        if submission_closed {
            return Err(actix_web::error::ErrorBadRequest(
                "Submission has been closed for this class.",
            ));
        }
    } else {
        return Err(actix_web::error::ErrorNotFound("No such class."));
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
    let file = file.unwrap();

    sqlx::query(
        "INSERT INTO `submissions` (`user_id`, `class_id`, `file_name`) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE `file_name` = VALUES(`file_name`)",
    )
    .bind(&user_id)
    .bind(class_id)
    .bind(file.content_disposition().unwrap().get_filename())
    .execute(&mut tx)
    .await
    .map_err(SqlxError)?;

    let mut data = file
        .map_ok(|b| web::BytesMut::from(&b[..]))
        .try_concat()
        .await?;
    let dst = format!("{}{}-{}.pdf", ASSIGNMENTS_DIRECTORY, class_id, user_id);
    let mut file = tokio::fs::OpenOptions::new()
        .write(true)
        .create(true)
        .truncate(true)
        .mode(0o666)
        .open(&dst)
        .await?;
    file.write_all_buf(&mut data).await?;

    tx.commit().await.map_err(SqlxError)?;

    Ok(HttpResponse::NoContent().finish())
}

#[derive(Debug, serde::Deserialize)]
struct Score {
    user_code: String,
    score: i64,
}

// PUT /api/courses/{course_id}/classes/{class_id}/assignments/scores 採点結果登録
async fn register_scores(
    pool: web::Data<sqlx::MySqlPool>,
    path: web::Path<AssignmentPath>,
    req: web::Json<Vec<Score>>,
) -> actix_web::Result<HttpResponse> {
    let class_id = &path.class_id;

    let mut tx = pool.begin().await.map_err(SqlxError)?;

    let submission_closed: Option<bool> = isucholar::db::fetch_optional_scalar(
        sqlx::query_scalar("SELECT `submission_closed` FROM `classes` WHERE `id` = ? FOR SHARE")
            .bind(class_id),
        &mut tx,
    )
    .await
    .map_err(SqlxError)?;
    if let Some(submission_closed) = submission_closed {
        if !submission_closed {
            return Err(actix_web::error::ErrorBadRequest(
                "This assignment is not closed yet.",
            ));
        }
    } else {
        return Err(actix_web::error::ErrorNotFound("No such class."));
    }

    for score in req.into_inner() {
        sqlx::query("UPDATE `submissions` JOIN `users` ON `users`.`id` = `submissions`.`user_id` SET `score` = ? WHERE `users`.`code` = ? AND `class_id` = ?")
            .bind(&score.score)
            .bind(&score.user_code)
            .bind(class_id)
            .execute(&mut tx)
            .await
            .map_err(SqlxError)?;
    }

    tx.commit().await.map_err(SqlxError)?;

    Ok(HttpResponse::NoContent().finish())
}

#[derive(Debug, sqlx::FromRow)]
struct Submission {
    user_id: String,
    user_code: String,
    file_name: String,
}

// GET /api/courses/{course_id}/classes/{class_id}/assignments/export 提出済みの課題ファイルをzip形式で一括ダウンロード
async fn download_submitted_assignments(
    pool: web::Data<sqlx::MySqlPool>,
    path: web::Path<AssignmentPath>,
) -> actix_web::Result<actix_files::NamedFile> {
    let class_id = &path.class_id;

    let mut tx = pool.begin().await.map_err(SqlxError)?;

    let class_count: i64 = isucholar::db::fetch_one_scalar(
        sqlx::query_scalar("SELECT COUNT(*) FROM `classes` WHERE `id` = ? FOR UPDATE")
            .bind(class_id),
        &mut tx,
    )
    .await
    .map_err(SqlxError)?;
    if class_count == 0 {
        return Err(actix_web::error::ErrorNotFound("No such class."));
    }
    let submissions: Vec<Submission> = sqlx::query_as(concat!(
        "SELECT `submissions`.`user_id`, `submissions`.`file_name`, `users`.`code` AS `user_code`",
        " FROM `submissions`",
        " JOIN `users` ON `users`.`id` = `submissions`.`user_id`",
        " WHERE `class_id` = ?",
    ))
    .bind(class_id)
    .fetch_all(&mut tx)
    .await
    .map_err(SqlxError)?;

    let zip_file_path = format!("{}{}.zip", ASSIGNMENTS_DIRECTORY, class_id);
    create_submissions_zip(&zip_file_path, class_id, &submissions).await?;

    sqlx::query("UPDATE `classes` SET `submission_closed` = true WHERE `id` = ?")
        .bind(class_id)
        .execute(&mut tx)
        .await
        .map_err(SqlxError)?;

    tx.commit().await.map_err(SqlxError)?;

    Ok(actix_files::NamedFile::open(&zip_file_path)?)
}

async fn create_submissions_zip(
    zip_file_path: &str,
    class_id: &str,
    submissions: &[Submission],
) -> std::io::Result<()> {
    let tmp_dir = format!("{}{}/", ASSIGNMENTS_DIRECTORY, class_id);
    tokio::process::Command::new("rm")
        .stdin(std::process::Stdio::null())
        .stdout(std::process::Stdio::null())
        .stderr(std::process::Stdio::null())
        .arg("-rf")
        .arg(&tmp_dir)
        .status()
        .await?;
    tokio::process::Command::new("mkdir")
        .stdin(std::process::Stdio::null())
        .stdout(std::process::Stdio::null())
        .stderr(std::process::Stdio::null())
        .arg(&tmp_dir)
        .status()
        .await?;

    // ファイル名を指定の形式に変更
    for submission in submissions {
        tokio::process::Command::new("cp")
            .stdin(std::process::Stdio::null())
            .stdout(std::process::Stdio::null())
            .stderr(std::process::Stdio::null())
            .arg(&format!(
                "{}{}-{}.pdf",
                ASSIGNMENTS_DIRECTORY, class_id, submission.user_id
            ))
            .arg(&format!(
                "{}{}-{}",
                tmp_dir, submission.user_code, submission.file_name
            ))
            .status()
            .await?;
    }

    // -i 'tmp_dir/*': 空zipを許す
    tokio::process::Command::new("zip")
        .stdin(std::process::Stdio::null())
        .stdout(std::process::Stdio::null())
        .stderr(std::process::Stdio::null())
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

// ---------- Announcement API ----------

#[derive(Debug, sqlx::FromRow, serde::Serialize)]
struct AnnouncementWithoutDetail {
    id: String,
    course_id: String,
    course_name: String,
    title: String,
    unread: bool,
}

#[derive(Debug, serde::Serialize)]
struct GetAnnouncementsResponse {
    unread_count: i64,
    announcements: Vec<AnnouncementWithoutDetail>,
}

#[derive(Debug, serde::Deserialize, serde::Serialize)]
struct GetAnnouncementsQuery {
    course_id: Option<String>,
    page: Option<String>,
}

// GET /api/announcements お知らせ一覧取得
async fn get_announcement_list(
    pool: web::Data<sqlx::MySqlPool>,
    session: actix_session::Session,
    params: web::Query<GetAnnouncementsQuery>,
    request: actix_web::HttpRequest,
) -> actix_web::Result<HttpResponse> {
    let (user_id, _, _) = get_user_info(session)?;

    let mut tx = pool.begin().await.map_err(SqlxError)?;

    let mut query = concat!(
        "SELECT `announcements`.`id`, `courses`.`id` AS `course_id`, `courses`.`name` AS `course_name`, `announcements`.`title`, NOT `unread_announcements`.`is_deleted` AS `unread`",
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
        " ORDER BY `announcements`.`id` DESC",
        " LIMIT ? OFFSET ?",
    ));
    args.add(&user_id);
    args.add(&user_id);

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

    let mut announcements: Vec<AnnouncementWithoutDetail> = sqlx::query_as_with(&query, args)
        .fetch_all(&mut tx)
        .await
        .map_err(SqlxError)?;

    let unread_count: i64 = isucholar::db::fetch_one_scalar(
        sqlx::query_scalar(
            "SELECT COUNT(*) FROM `unread_announcements` WHERE `user_id` = ? AND NOT `is_deleted`",
        )
        .bind(&user_id),
        &mut tx,
    )
    .await
    .map_err(SqlxError)?;

    tx.commit().await.map_err(SqlxError)?;

    let uri = request.uri();
    let mut params = params.into_inner();
    let mut links = Vec::new();
    if page > 1 {
        params.page = Some(format!("{}", page - 1));
        links.push(format!(
            "<{}?{}>; rel=\"prev\"",
            uri.path(),
            serde_urlencoded::to_string(&params)?
        ));
    }
    if announcements.len() as i64 > limit {
        params.page = Some(format!("{}", page + 1));
        links.push(format!(
            "<{}?{}>; rel=\"next\"",
            uri.path(),
            serde_urlencoded::to_string(&params)?
        ));
    }

    if announcements.len() as i64 == limit + 1 {
        announcements.truncate(announcements.len() - 1);
    }

    // 対象になっているお知らせが0件の時は空配列を返却

    let mut builder = HttpResponse::Ok();
    if !links.is_empty() {
        builder.insert_header((actix_web::http::header::LINK, links.join(",")));
    }
    Ok(builder.json(GetAnnouncementsResponse {
        unread_count,
        announcements,
    }))
}

#[derive(Debug, sqlx::FromRow)]
struct Announcement {
    id: String,
    course_id: String,
    title: String,
    message: String,
}

#[derive(Debug, serde::Deserialize)]
struct AddAnnouncementRequest {
    id: String,
    course_id: String,
    title: String,
    message: String,
}

// POST /api/announcements 新規お知らせ追加
async fn add_announcement(
    pool: web::Data<sqlx::MySqlPool>,
    req: web::Json<AddAnnouncementRequest>,
) -> actix_web::Result<HttpResponse> {
    let mut tx = pool.begin().await.map_err(SqlxError)?;

    let count: i64 = isucholar::db::fetch_one_scalar(
        sqlx::query_scalar("SELECT COUNT(*) FROM `courses` WHERE `id` = ?").bind(&req.course_id),
        &mut tx,
    )
    .await
    .map_err(SqlxError)?;
    if count == 0 {
        return Err(actix_web::error::ErrorNotFound("No such course."));
    }

    let result = sqlx::query(
        "INSERT INTO `announcements` (`id`, `course_id`, `title`, `message`) VALUES (?, ?, ?, ?)",
    )
    .bind(&req.id)
    .bind(&req.course_id)
    .bind(&req.title)
    .bind(&req.message)
    .execute(&mut tx)
    .await;
    if let Err(e) = result {
        let _ = tx.rollback().await;
        if let sqlx::error::Error::Database(ref db_error) = e {
            if let Some(mysql_error) =
                db_error.try_downcast_ref::<sqlx::mysql::MySqlDatabaseError>()
            {
                if mysql_error.number() == MYSQL_ERR_NUM_DUPLICATE_ENTRY {
                    let announcement: Announcement =
                        sqlx::query_as("SELECT * FROM `announcements` WHERE `id` = ?")
                            .bind(&req.id)
                            .fetch_one(pool.as_ref())
                            .await
                            .map_err(SqlxError)?;
                    if announcement.course_id != req.course_id
                        || announcement.title != req.title
                        || announcement.message != req.message
                    {
                        return Err(actix_web::error::ErrorConflict(
                            "An announcement with the same id already exists.",
                        ));
                    } else {
                        return Ok(HttpResponse::Created().finish());
                    }
                }
            }
        }
        return Err(SqlxError(e).into());
    }

    let targets: Vec<User> = sqlx::query_as(concat!(
        "SELECT `users`.* FROM `users`",
        " JOIN `registrations` ON `users`.`id` = `registrations`.`user_id`",
        " WHERE `registrations`.`course_id` = ?",
    ))
    .bind(&req.course_id)
    .fetch_all(&mut tx)
    .await
    .map_err(SqlxError)?;

    for user in targets {
        sqlx::query(
            "INSERT INTO `unread_announcements` (`announcement_id`, `user_id`) VALUES (?, ?)",
        )
        .bind(&req.id)
        .bind(user.id)
        .execute(&mut tx)
        .await
        .map_err(SqlxError)?;
    }

    tx.commit().await.map_err(SqlxError)?;

    Ok(HttpResponse::Created().finish())
}

#[derive(Debug, sqlx::FromRow, serde::Serialize)]
struct AnnouncementDetail {
    id: String,
    course_id: String,
    course_name: String,
    title: String,
    message: String,
    unread: bool,
}

// GET /api/announcements/{announcement_id} お知らせ詳細取得
async fn get_announcement_detail(
    pool: web::Data<sqlx::MySqlPool>,
    session: actix_session::Session,
    announcement_id: web::Path<(String,)>,
) -> actix_web::Result<HttpResponse> {
    let (user_id, _, _) = get_user_info(session)?;

    let announcement_id = &announcement_id.0;

    let mut tx = pool.begin().await.map_err(SqlxError)?;

    let announcement: Option<AnnouncementDetail> = isucholar::db::fetch_optional_as(
        sqlx::query_as(concat!(
                "SELECT `announcements`.`id`, `courses`.`id` AS `course_id`, `courses`.`name` AS `course_name`, `announcements`.`title`, `announcements`.`message`, NOT `unread_announcements`.`is_deleted` AS `unread`",
                " FROM `announcements`",
                " JOIN `courses` ON `courses`.`id` = `announcements`.`course_id`",
                " JOIN `unread_announcements` ON `unread_announcements`.`announcement_id` = `announcements`.`id`",
                " WHERE `announcements`.`id` = ?",
                " AND `unread_announcements`.`user_id` = ?",
        )).bind(announcement_id).bind(&user_id),
        &mut tx
    )
    .await
    .map_err(SqlxError)?;
    if announcement.is_none() {
        return Err(actix_web::error::ErrorNotFound("No such announcement."));
    }
    let announcement = announcement.unwrap();

    let registration_count: i64 = isucholar::db::fetch_one_scalar(
        sqlx::query_scalar(
            "SELECT COUNT(*) FROM `registrations` WHERE `course_id` = ? AND `user_id` = ?",
        )
        .bind(&announcement.course_id)
        .bind(&user_id),
        &mut tx,
    )
    .await
    .map_err(SqlxError)?;
    if registration_count == 0 {
        return Err(actix_web::error::ErrorNotFound("No such announcement."));
    }

    sqlx::query("UPDATE `unread_announcements` SET `is_deleted` = true WHERE `announcement_id` = ? AND `user_id` = ?")
        .bind(announcement_id)
        .bind(&user_id)
        .execute(&mut tx)
        .await
        .map_err(SqlxError)?;

    tx.commit().await.map_err(SqlxError)?;

    Ok(HttpResponse::Ok().json(announcement))
}
