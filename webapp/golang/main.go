package main

import (
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pborman/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	SQLDirectory         = "../sql/"
	AssignmentsDirectory = "../assignments/"
	DocDirectory         = "../documents/"
	SessionName          = "session"
)

type handlers struct {
	DB *sqlx.DB
}

func main() {
	e := echo.New()
	e.Debug = GetEnv("DEBUG", "") != ""
	e.Server.Addr = fmt.Sprintf(":%v", GetEnv("PORT", "7000"))
	e.HideBanner = true

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(session.Middleware(sessions.NewCookieStore([]byte("trapnomura"))))

	db, _ := GetDB(false)

	h := &handlers{
		DB: db,
	}

	// e.POST("/initialize", h.Initialize, h.IsLoggedIn, h.IsAdmin)
	e.POST("/initialize", h.Initialize)
	e.PUT("/phase", h.SetPhase, h.IsLoggedIn, h.IsAdmin)

	e.POST("/login", h.Login)
	API := e.Group("/api", h.IsLoggedIn)
	{
		usersAPI := API.Group("/users")
		{
			usersAPI.GET("/:userID/courses", h.GetRegisteredCourses)
			usersAPI.PUT("/:userID/courses", h.RegisterCourses)
			usersAPI.GET("/:userID/grades", h.GetGrades)
		}
		syllabusAPI := API.Group("/syllabus")
		{
			syllabusAPI.GET("", h.SearchCourses)
			syllabusAPI.GET("/:courseID", h.GetCourseSyllabus)
		}
		coursesAPI := API.Group("/courses")
		{
			coursesAPI.GET("/:courseID", h.GetCourseDetail)
			coursesAPI.GET("/:courseID/documents", h.GetCourseDocumentList)
			coursesAPI.POST("/:courseID/classes/:classID/documents", h.PostDocumentFile, h.IsAdmin)
			coursesAPI.GET("/:courseID/documents/:documentID", h.DownloadDocumentFile)
			coursesAPI.GET("/:courseID/assignments", h.GetAssignmentList)
			coursesAPI.POST("/:courseID/classes/:classID/assignments", h.PostAssignment, h.IsAdmin)
			coursesAPI.POST("/:courseID/assignments/:assignmentID", h.SubmitAssignment)
			coursesAPI.GET("/:courseID/assignments/:assignmentID/export", h.DownloadSubmittedAssignment, h.IsAdmin)
			coursesAPI.GET("/:courseID/classes/:classID/code", h.GetAttendanceCode, h.IsAdmin)
			coursesAPI.POST("/:courseID/classes/:classID", h.SetClassFlag, h.IsAdmin)
			coursesAPI.GET("/:courseID/classes/:classID/attendances", h.GetAttendances, h.IsAdmin)
			coursesAPI.POST("/:courseID/announcements", h.AddAnnouncements, h.IsAdmin)
			coursesAPI.POST("/:courseID/grades", h.SetUserGrades, h.IsAdmin)
		}
		announcementsAPI := API.Group("/announcements")
		{
			announcementsAPI.GET("", h.GetAnnouncementList)
			announcementsAPI.GET("/:announcementID", h.GetAnnouncementDetail)
		}
		attendanceCodeAPI := API.Group("/attendance_codes")
		{
			attendanceCodeAPI.POST("", h.PostAttendanceCode)
		}
	}

	e.Logger.Error(e.StartServer(e.Server))
}

func (h *handlers) Initialize(c echo.Context) error {
	dbForInit, _ := GetDB(true)

	files := []string{
		"schema.sql",
		"test_data.sql",
	}
	for _, file := range files {
		data, err := ioutil.ReadFile(SQLDirectory + file)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("read sql file: %v", err))
		}
		if _, err := dbForInit.Exec(string(data)); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("exec sql file: %v", err))
		}
	}

	res := InitializeResponse{
		Language: "go",
	}
	return c.JSON(http.StatusOK, res)
}

func (h *handlers) IsLoggedIn(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, err := session.Get(SessionName, c)
		if err != nil {
			return echo.ErrInternalServerError
		}
		if sess.IsNew {
			return echo.NewHTTPError(http.StatusUnauthorized, "You are not logged in.")
		}
		if _, ok := sess.Values["userID"]; !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "You are not logged in.")
		}

		return next(c)
	}
}

func (h *handlers) IsAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, err := session.Get(SessionName, c)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("get session: %v", err))
		}
		isAdmin, ok := sess.Values["isAdmin"]
		if !ok {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("get session value: %v", err))
		}
		if !isAdmin.(bool) {
			return echo.NewHTTPError(http.StatusForbidden, "You are not admin user.")
		}

		return next(c)
	}
}

type SetPhaseRequest struct {
	Phase    PhaseType `json:"phase"`
	Year     uint32    `json:"year"`
	Semester Semester  `json:"semester"`
}

func (h *handlers) SetPhase(c echo.Context) error {
	var req SetPhaseRequest
	if err := c.Bind(&req); err != nil {
		log.Println(err)
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if req.Phase != Registration && req.Phase != TermTime && req.Phase != ExamPeriod {
		return echo.NewHTTPError(http.StatusBadRequest, "bad phase")
	}
	if req.Semester != FirstSemester && req.Semester != SecondSemester {
		return echo.NewHTTPError(http.StatusBadRequest, "bad semester")
	}

	if _, err := h.DB.Exec("TRUNCATE TABLE `phase`"); err != nil {
		log.Println(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if _, err := h.DB.Exec("INSERT INTO `phase` (`phase`, `year`, `semester`) VALUES (?, ?, ?)", req.Phase, req.Year, req.Semester); err != nil {
		log.Println(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

type InitializeResponse struct {
	Language string `json:"language"`
}

type LoginRequest struct {
	ID       uuid.UUID `json:"name,omitempty"` // TODO: やっぱり学籍番号を用意したほうが良さそうだけど教員の扱いどうしようかな
	Password string    `json:"password,omitempty"`
}

type RegisterCoursesRequestContent struct {
	ID string `json:"id"`
}

type RegisterCoursesRequest []RegisterCoursesRequestContent
type GetAttendanceCodeResponse struct {
	Code string `json:"code"`
}

type GetAttendancesAttendance struct {
	UserID     uuid.UUID `json:"user_id"`
	AttendedAt int64     `json:"attended_at"`
}

type GetAttendancesResponse []GetAttendancesAttendance

type PostAttendanceCodeRequest struct {
	Code string `json:"code"`
}

type GetDocumentResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type GetDocumentsResponse []GetDocumentResponse

type PhaseType string

const (
	Registration PhaseType = "registration"
	TermTime     PhaseType = "term-time"
	ExamPeriod   PhaseType = "exam-period"
)

type Semester string

const (
	FirstSemester  Semester = "first"
	SecondSemester Semester = "second"
)

type Phase struct {
	Phase    PhaseType `json:"phase"`
	Year     uint32    `json:"year"`
	Semester Semester  `json:"semester"`
}

type UserType string

const (
	_       UserType = "student" /* FIXME: use Student */
	Faculty UserType = "faculty"
)

type User struct {
	ID             uuid.UUID `db:"id"`
	Name           string    `db:"name"`
	MailAddress    string    `db:"mail_address"`
	HashedPassword []byte    `db:"hashed_password"`
	Type           UserType  `db:"type"`
}

type Course struct {
	ID          uuid.UUID `db:"id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	Credit      uint8     `db:"credit"`
	Classroom   string    `db:"classroom"`
	Capacity    uint32    `db:"capacity"`
}

type Schedule struct {
	ID        uuid.UUID `db:"id"`
	Period    uint8     `db:"period"`
	DayOfWeek string    `db:"day_of_week"`
	Semester  Semester  `db:"semester"`
	Year      uint32    `db:"year"`
}

type Class struct {
	ID             uuid.UUID `db:"id"`
	CourseID       uuid.UUID `db:"course_id"`
	Title          string    `db:"title"`
	Description    string    `db:"description"`
	AttendanceCode string    `db:"attendance_code"`
}

type Attendance struct {
	ClassID   uuid.UUID `db:"class_id"`
	UserID    uuid.UUID `db:"user_id"`
	CreatedAt time.Time `db:"created_at"`
}

type DocumentsMeta struct {
	ID        uuid.UUID `db:"id"`
	ClassID   uuid.UUID `db:"class_id"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
}

func (h *handlers) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("bind request: %v", err))
	}

	sess, err := session.Get(SessionName, c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("get session: %v", err))
	}
	if s, ok := sess.Values["userID"].(string); ok {
		userID := uuid.Parse(s)
		if uuid.Equal(userID, req.ID) {
			return echo.NewHTTPError(http.StatusBadRequest, "You are already logged in.")
		}
	}

	var user User
	err = h.DB.Get(&user, "SELECT * FROM users WHERE id = ?", req.ID)
	if err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusUnauthorized, "ID or Password is wrong.")
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("get users: %v", err))
	}

	if bcrypt.CompareHashAndPassword(user.HashedPassword, []byte(req.Password)) != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "ID or Password is wrong.")
	}

	sess.Values["userID"] = user.ID.String()
	sess.Values["userName"] = user.Name
	sess.Values["isAdmin"] = user.Type == Faculty
	sess.Options = &sessions.Options{
		Path:   "/",
		MaxAge: 3600,
	}

	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("save session: %v", err))
	}

	return c.NoContent(http.StatusOK)
}

func (h *handlers) GetRegisteredCourses(context echo.Context) error {
	sess, err := session.Get(SessionName, context)
	if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}
	userID := uuid.Parse(sess.Values["userID"].(string))
	if uuid.Equal(uuid.NIL, userID) {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}
	userIDParam := uuid.Parse(context.Param("userID"))
	if uuid.Equal(uuid.NIL, userIDParam) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid userID")
	}
	if !uuid.Equal(userID, userIDParam) {
		return echo.NewHTTPError(http.StatusForbidden, "invalid userID")
	}

	courses := make([]Course, 0)
	err = h.DB.Select(&courses, "SELECT `id`, `name`, `description`, `credit`, `classroom`, `capacity`\n"+
		"	FROM `courses` JOIN `registrations` ON `courses`.`id` = `registrations`.`course_id`\n"+
		"	WHERE `user_id` = ?", userID)
	if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	return context.JSON(http.StatusOK, courses)
}

func (h *handlers) RegisterCourses(context echo.Context) error {
	sess, err := session.Get(SessionName, context)
	if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}
	userID := uuid.Parse(sess.Values["userID"].(string))
	if uuid.Equal(uuid.NIL, userID) {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}
	userIDParam := uuid.Parse(context.Param("userID"))
	if uuid.Equal(uuid.NIL, userIDParam) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid userID")
	}
	if !uuid.Equal(userID, userIDParam) {
		return echo.NewHTTPError(http.StatusForbidden, "invalid userID")
	}

	var req RegisterCoursesRequest
	if err := context.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("bind request: %v", err))
	}

	tx, err := h.DB.Beginx()
	if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	// MEMO: SELECT ... FOR UPDATE は今のDB構造だとデッドロックする
	_, err = tx.Exec("LOCK TABLES `registrations` WRITE, `courses` READ, `course_requirements` READ, `grades` READ, `schedules` READ, `course_schedules` READ")
	if err != nil {
		_ = tx.Rollback()
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}
	defer func() {
		_, _ = h.DB.Exec("UNLOCK TABLES")
	}()

	var courseList []Course
	for _, content := range req {
		var course Course
		// MEMO: TODO: 年度、学期の扱い
		err := tx.Get(&course, "SELECT * FROM `courses` WHERE `id` = ?", content.ID)
		if err == sql.ErrNoRows {
			_ = tx.Rollback()
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Not found course. id: %v", content.ID))
		} else if err != nil {
			_ = tx.Rollback()
			log.Println(err)
			return context.NoContent(http.StatusInternalServerError)
		}
		courseList = append(courseList, course)
	}

	// MEMO: LOGIC: 前提講義/受講者数制限バリデーション
	for _, course := range courseList {
		var requiredCourseIDList []string
		err = tx.Select(&requiredCourseIDList, "SELECT `required_course_id` FROM `course_requirements` WHERE `course_id` = ?", course.ID)
		if err != nil {
			_ = tx.Rollback()
			log.Println(err)
			return context.NoContent(http.StatusInternalServerError)
		}
		for _, requiredCourseID := range requiredCourseIDList {
			var gradeCount uint32
			err = tx.Get(&gradeCount, "SELECT COUNT(*) FROM `grades` WHERE `user_id` = ? AND `course_id` = ?", userID, requiredCourseID)
			if err != nil {
				_ = tx.Rollback()
				log.Println(err)
				return context.NoContent(http.StatusInternalServerError)
			}
			if gradeCount == 0 {
				_ = tx.Rollback()
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("You have not taken required course. required course id: %v", requiredCourseID))
			}
		}

		var registerCount uint32
		err = tx.Get(&registerCount, "SELECT COUNT(*) FROM `registrations` WHERE `course_id` = ?", course.ID)
		if err != nil {
			_ = tx.Rollback()
			log.Println(err)
			return context.NoContent(http.StatusInternalServerError)
		}
		if registerCount >= course.Capacity {
			_ = tx.Rollback()
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Course capacity exceeded. course id: %v", course.ID))
		}
	}

	// MEMO: LOGIC: スケジュールの重複バリデーション
	// MEMO: さすがに二重ループはやりすぎな気もする
	getSchedule := func(courseID uuid.UUID) (Schedule, error) {
		var schedule Schedule
		err := tx.Get(&schedule, "SELECT `id`, `period`, `day_of_week`, `semester`, `year`\n"+
			"	FROM `schedules` JOIN `course_schedules` ON `schedules`.`id` = `course_schedules`.`schedule_id`\n"+
			"	WHERE `course_id` = ?", courseID)
		if err != nil {
			return schedule, err
		}
		return schedule, nil
	}

	for i := 0; i < len(courseList); i++ {
		for j := i + 1; j < len(courseList); j++ {
			schedule1, err := getSchedule(courseList[i].ID)
			if err != nil {
				_ = tx.Rollback()
				log.Println(err)
				return context.NoContent(http.StatusInternalServerError)
			}
			schedule2, err := getSchedule(courseList[j].ID)
			if err != nil {
				_ = tx.Rollback()
				log.Println(err)
				return context.NoContent(http.StatusInternalServerError)
			}
			if schedule1.Period == schedule2.Period && schedule1.DayOfWeek == schedule2.DayOfWeek {
				_ = tx.Rollback()
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("You cannot take courses held on same schedule. course id: %v and %v", courseList[i].ID, courseList[j].ID))
			}
		}
	}

	// MEMO: LOGIC: 履修登録
	for _, course := range courseList {
		var count uint32
		err := tx.Get(&count, "SELECT COUNT(*) FROM `registrations` WHERE `course_id` = ? AND `user_id` = ?", course.ID, userID)
		if err != nil {
			_ = tx.Rollback()
			log.Println(err)
			return context.NoContent(http.StatusInternalServerError)
		}
		if count > 0 {
			continue
		}

		_, err = tx.Exec("INSERT INTO `registrations` (`course_id`, `user_id`, `created_at`) VALUES (?, ?, NOW(6))", course.ID, userID)
		if err != nil {
			_ = tx.Rollback()
			log.Println(err)
			return context.NoContent(http.StatusInternalServerError)
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	return context.NoContent(http.StatusOK)
}

func (h *handlers) GetGrades(context echo.Context) error {
	panic("implement me")
}

func (h *handlers) SearchCourses(context echo.Context) error {
	panic("implement me")
}

func (h *handlers) GetCourseSyllabus(context echo.Context) error {
	panic("implement me")
}

func (h *handlers) GetCourseDetail(context echo.Context) error {
	courseID := uuid.Parse(context.Param("courseID"))
	if courseID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid courseID")
	}

	var course Course
	if err := h.DB.Get(&course, "SELECT * from `courses` WHERE `id` = ?", courseID); err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusNotFound, "No such course")
	} else if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	return context.JSON(http.StatusOK, course)
}

type PostAssignmentRequest struct {
	Name        string
	Description string
	Deadline    time.Time
}

func (h *handlers) PostAssignment(context echo.Context) error {
	var req PostAssignmentRequest
	if err := context.Bind(&req); err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	if req.Name == "" || req.Description == "" || req.Deadline.IsZero() {
		return echo.NewHTTPError(http.StatusBadRequest, "Name, description and deadline must not be empty.")
	}

	classID := context.Param("classID")
	var classes int
	if err := h.DB.Get(&classes, "SELECT COUNT(*) FROM `classes` WHERE `id` = ?", classID); err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusBadRequest, "No such class.")
	} else if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}
	if _, err := h.DB.Exec("INSERT INTO `assignments` (`id`, `class_id`, `name`, `description`, `deadline`, `created_at`) VALUES (?, ?, ?, ?, ?, NOW(6))", uuid.New(), classID, req.Name, req.Description, req.Deadline.Truncate(time.Microsecond)); err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	return context.NoContent(http.StatusCreated)
}

func (h *handlers) GetCourseDocumentList(context echo.Context) error {
	courseID := uuid.Parse(context.Param("courseID"))
	if uuid.Equal(uuid.NIL, courseID) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid courseID")
	}
	classID := uuid.Parse(context.Param("classID"))
	if uuid.Equal(uuid.NIL, classID) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid classID")
	}

	documentsMeta := make([]DocumentsMeta, 0)
	err := h.DB.Select(&documentsMeta, "SELECT `documents`.* FROM `documents` JOIN `classes` ON `classes`.`id` = `documents`.`class_id` WHERE `classes`.`course_id` = ?", courseID)
	if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	res := make(GetDocumentsResponse, 0, len(documentsMeta))
	for _, meta := range documentsMeta {
		res = append(res, GetDocumentResponse{
			ID:   meta.ID,
			Name: meta.Name,
		})
	}

	return context.JSON(http.StatusOK, res)

}

func (h *handlers) PostDocumentFile(context echo.Context) error {
	courseID := uuid.Parse(context.Param("courseID"))
	if uuid.Equal(uuid.NIL, courseID) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid courseID")
	}
	classID := uuid.Parse(context.Param("classID"))
	if uuid.Equal(uuid.NIL, classID) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid classID")
	}

	form, err := context.MultipartForm()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "read request err")
	}
	files := form.File["files"]

	// 作ったファイルの名前を格納しておく
	dsts := make([]string, 0, len(files))

	tx, err := h.DB.Begin()
	if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	// 作成したファイルを削除する
	deleteFiles := func(dsts []string) {
		for _, file := range dsts {
			os.Remove(file)
		}
	}

	for _, file := range files {
		src, err := file.Open()
		if err != nil {
			log.Println(err)
			_ = tx.Rollback()
			deleteFiles(dsts)
			return context.NoContent(http.StatusInternalServerError)
		}

		fileMeta := DocumentsMeta{
			ID:      uuid.NewRandom(),
			ClassID: classID,
			Name:    file.Filename,
		}

		filePath := fmt.Sprintf("%s%s", DocDirectory, fileMeta.ID)

		dst, err := os.Create(filePath)
		if err != nil {
			log.Println(err)
			_ = tx.Rollback()
			deleteFiles(dsts)
			return context.NoContent(http.StatusInternalServerError)
		}

		dsts = append(dsts, filePath)
		_, err = tx.Exec("INSERT INTO `documents` (`id`, `class_id`, `name`, `created_at`) VALUES (?, ?, ?, NOW(6))",
			fileMeta.ID,
			fileMeta.ClassID,
			fileMeta.Name,
		)
		if err != nil {
			log.Println(err)
			_ = tx.Rollback()
			deleteFiles(dsts)
			return context.NoContent(http.StatusInternalServerError)
		}

		if _, err = io.Copy(dst, src); err != nil {
			log.Println(err)
			_ = tx.Rollback()
			deleteFiles(dsts)
			return context.NoContent(http.StatusInternalServerError)
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	return context.NoContent(http.StatusCreated)
}

func (h *handlers) GetAssignmentList(context echo.Context) error {
	panic("implement me")
}

func (h *handlers) DownloadDocumentFile(context echo.Context) error {
	courseID := uuid.Parse(context.Param("courseID"))
	if uuid.Equal(uuid.NIL, courseID) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid courseID")
	}
	documentID := uuid.Parse(context.Param("documentID"))
	if uuid.Equal(uuid.NIL, documentID) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid classID")
	}

	var documentMeta DocumentsMeta
	err := h.DB.Get(&documentMeta, "SELECT `documents`.* FROM `documents` JOIN `classes` ON `classes`.`id` = `documents`.`class_id` "+
		"WHERE `documents`.`id` = ? AND `classes`.`course_id` = ?", documentID, courseID)
	if err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusNotFound, "file not found")
	} else if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	filePath := fmt.Sprintf("%s%s", DocDirectory, documentMeta.ID)
	return context.File(filePath)
}

func (h *handlers) SubmitAssignment(context echo.Context) error {
	sess, err := session.Get(SessionName, context)
	if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}
	userID := uuid.Parse(sess.Values["userID"].(string))

	assignmentID := context.Param("assignmentID")
	var assignments int
	if err := h.DB.Get(&assignments, "SELECT COUNT(*) FROM `assignments` WHERE `id` = ?", assignmentID); err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusBadRequest, "No such assignment.")
	} else if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	file, err := context.FormFile("file")
	if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}
	src, err := file.Open()
	if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}
	defer src.Close()

	submissionID := uuid.New()
	dst, err := os.Create(AssignmentsDirectory + submissionID)
	if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	if _, err := h.DB.Exec("INSERT INTO `submissions` (`id`, `user_id`, `assignment_id`, `name`, `created_at`) VALUES (?, ?, ?, ?, NOW(6))", submissionID, userID, assignmentID, file.Filename); err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	return context.NoContent(http.StatusNoContent)
}

func (h *handlers) DownloadSubmittedAssignment(context echo.Context) error {
	panic("implement me")
}

func (h *handlers) GetAttendanceCode(context echo.Context) error {
	courseID := uuid.Parse(context.Param("courseID"))
	if uuid.Equal(uuid.NIL, courseID) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid courseID")
	}
	classID := uuid.Parse(context.Param("classID"))
	if uuid.Equal(uuid.NIL, classID) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid classID")
	}

	var res GetAttendanceCodeResponse
	if err := h.DB.Get(&res.Code, "SELECT `attendance_code` FROM `classes` WHERE `course_id` = ? AND `id` = ?", courseID, classID); err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusNotFound, "course or class not found")
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("get attendance code: %v", err))
	}

	return context.JSON(http.StatusOK, res)
}

func (h *handlers) SetClassFlag(context echo.Context) error {
	panic("implement me")
}

func (h *handlers) GetAttendances(context echo.Context) error {
	courseID := uuid.Parse(context.Param("courseID"))
	if uuid.Equal(uuid.NIL, courseID) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid courseID")
	}
	classID := uuid.Parse(context.Param("classID"))
	if uuid.Equal(uuid.NIL, classID) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid classID")
	}

	var attendances []Attendance
	if err := h.DB.Select(&attendances, "SELECT * FROM `attendances` WHERE `class_id` = ?", classID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("get attendances: %v", err))
	}

	res := make(GetAttendancesResponse, len(attendances))
	for i, attendance := range attendances {
		res[i] = GetAttendancesAttendance{
			UserID:     attendance.UserID,
			AttendedAt: attendance.CreatedAt.UnixNano() / int64(time.Millisecond),
		}
	}

	return context.JSON(http.StatusOK, res)
}

func (h *handlers) AddAnnouncements(context echo.Context) error {
	panic("implement me")
}

func (h *handlers) SetUserGrades(context echo.Context) error {
	panic("implement me")
}

func (h *handlers) GetAnnouncementList(context echo.Context) error {
	panic("implement me")
}

func (h *handlers) GetAnnouncementDetail(context echo.Context) error {
	panic("implement me")
}

func (h *handlers) PostAttendanceCode(c echo.Context) error {
	sess, err := session.Get(SessionName, c)
	if err != nil {
		log.Println(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	userID := uuid.Parse(sess.Values["userID"].(string))
	if uuid.Equal(uuid.NIL, userID) {
		log.Println(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var req PostAttendanceCodeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("bind request: %v", err))
	}

	// 出席コード確認
	var class Class
	if err := h.DB.Get(&class, "SELECT * FROM `classes` WHERE `attendance_code` = ?", req.Code); err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid code")
	} else if err != nil {
		log.Println(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	// 学期確認
	// MEMO: 複数phaseに渡る講義を想定していない
	var schedule Schedule
	query := "SELECT `schedules`.*" +
		"FROM `schedules`" +
		"JOIN `course_schedules` ON `schedules`.`id` = `course_schedules`.`schedule_id`" +
		"JOIN `courses` ON `course_schedules`.`course_id` = `courses`.`id`" +
		"WHERE `courses`.`id` = ?" +
		"LIMIT 1"
	if err := h.DB.Get(&schedule, query, class.CourseID); err != nil {
		log.Println(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	var phase Phase
	if err := h.DB.Get(&phase, "SELECT * FROM `phase`"); err != nil {
		log.Println(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if schedule.Year != phase.Year || schedule.Semester != phase.Semester {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid code")
	}

	// 履修確認
	var registration int
	if err := h.DB.Get(&registration, "SELECT COUNT(*) FROM `registrations` WHERE `course_id` = ? AND `user_id` = ? AND `deleted_at` IS NULL", class.CourseID, userID); err != nil {
		log.Println(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if registration == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "You are not registered in the course.")
	}

	// 既に出席しているか
	var attendances int
	if err := h.DB.Get(&attendances, "SELECT COUNT(*) FROM `attendances` WHERE `class_id` = ? AND `user_id` = ?", class.ID, userID); err != nil {
		log.Println(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if attendances > 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "You have already attended in this class.")
	}

	// 出席コード登録
	if _, err := h.DB.Exec("INSERT INTO `attendances` (`class_id`, `user_id`, `created_at`) VALUES (?, ?, NOW(6))", class.ID, userID); err != nil {
		log.Println(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}
