package main

import (
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
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
	SQLDirectory  = "../sql/"
	FileDirectory = "./files"
	SessionName   = "session"
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

	//e.POST("/initialize", h.Initialize, h.IsLoggedIn, h.IsAdmin)
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
			coursesAPI.GET("/:courseID/documents", h.GetCourseDocumetList)
			coursesAPI.POST("/:courseID/documents", h.PostDocumentFile, h.IsAdmin)
			coursesAPI.GET("/:courseID/documents/:documentID", h.DownloadDocumentFile)
			coursesAPI.GET("/:courseID/classes/:classID/assignments", h.GetAssignmentList)
			coursesAPI.POST("/:courseID/classes/:classID/assignments", h.PostAssignment, h.IsAdmin)
			coursesAPI.POST("/:courseID/classes/:classID/assignments/:assignmentID", h.SubmitAssignment)
			coursesAPI.GET("/:courseID/classes/:classID/assignments/:assignmentID/export", h.DownloadSubmittedAssignment, h.IsAdmin)
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

func (h *handlers) SetPhase(c echo.Context) error {
	panic("implement me")
}

type InitializeResponse struct {
	Language string `json:"language"`
}

type LoginRequest struct {
	ID       uuid.UUID `json:"name,omitempty"` // TODO: やっぱり学籍番号を用意したほうが良さそうだけど教員の扱いどうしようかな
	Password string    `json:"password,omitempty"`
}

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

func (h *handlers) GetRegisteredCourses(c echo.Context) error {
	panic("implement me")
}

func (h *handlers) RegisterCourses(context echo.Context) error {
	panic("implement me")
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
	panic("implement me")
}

type PostAssignmentRequest struct {
	Name        string
	Description string
	Deadline    time.Time
}

func (h *handlers) PostAssignment(context echo.Context) error {
	var req PostAssignmentRequest
	if err := context.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("bind request: %v", err))
	}

	if req.Name == "" || req.Description == "" || req.Deadline.IsZero() {
		return echo.NewHTTPError(http.StatusBadRequest, "Name, description and deadline must not be empty.")
	}

	classID := context.Param("classID")
	var classes int
	if err := h.DB.Get(&classes, "SELECT COUNT(*) FROM `classes` WHERE `id` = ?", classID); err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusBadRequest, "No such class.")
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("get classes: %v", err))
	}
	if _, err := h.DB.Exec("INSERT INTO `assignments` (`id`, `class_id`, `name`, `description`, `deadline`, `created_at`) VALUES (?, ?, ?, ?, ?, NOW(6))", uuid.New(), classID, req.Name, req.Description, req.Deadline.Truncate(time.Microsecond)); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("insert assignment: %v", err))
	}

	return context.NoContent(http.StatusCreated)
}

func (h *handlers) GetCourseDocumetList(context echo.Context) error {
	panic("implement me")
}

func (h *handlers) PostDocumentFile(context echo.Context) error {
	panic("implement me")
}

func (h *handlers) GetAssignmentList(context echo.Context) error {
	panic("implement me")
}

func (h *handlers) DownloadDocumentFile(context echo.Context) error {
	panic("implement me")
}

// MEMO: PATH・Schemaの再検討
func (h *handlers) SubmitAssignment(context echo.Context) error {
	sess, err := session.Get(SessionName, context)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("get session: %v", err))
	}
	userID := uuid.Parse(sess.Values["userID"].(string))

	assignmentID := context.Param("assignmentID")
	var assignments int
	if err := h.DB.Get(&assignments, "SELECT COUNT(*) FROM `assignments` WHERE `id` = ?", assignmentID); err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusBadRequest, "No such assignment.")
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("get assignments: %v", err))
	}

	file, err := context.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("get file: %v", err))
	}
	src, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("open file: %v", err))
	}
	defer src.Close()

	submissionID := uuid.New()
	dst, err := os.Create(FileDirectory + file.Filename + submissionID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("create file: %v", err))
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("save submitted file: %v", err))
	}

	if _, err := h.DB.Exec("INSERT INTO `submissions` (`id`, `user_id`, `assignment_id`, `name`, `created_at`) VALUES (?, ?, ?, ?, NOW(6))", submissionID, userID, assignmentID, file.Filename); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("insert submission: %v", err))
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

func (h *handlers) PostAttendanceCode(context echo.Context) error {
	sess, err := session.Get(SessionName, context)
	if err != nil {
		return echo.ErrInternalServerError
	}
	userID := uuid.Parse(sess.Values["userID"].(string))
	if uuid.Equal(uuid.NIL, userID) {
		return echo.NewHTTPError(http.StatusInternalServerError, "get userID from session")
	}

	var req PostAttendanceCodeRequest
	if err := context.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("bind request: %v", err))
	}

	// 出席コード確認
	var class Class
	if err := h.DB.Get(&class, "SELECT * FROM `classes` WHERE `attendance_code` = ?", req.Code); err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid code")
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("get class: %v", err))
	}

	// 履修確認
	var registration int
	if err := h.DB.Get(&registration, "SELECT COUNT(*) FROM `registrations` WHERE `course_id` = ? AND `user_id` = ? AND `deleted_at` IS NULL", class.CourseID, userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("check registration: %v", err))
	}
	if registration == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "You are not registered in the course.")
	}

	// 既に出席しているか
	var attendances int
	if err := h.DB.Get(&attendances, "SELECT COUNT(*) FROM `attendances` WHERE `class_id` = ? AND `user_id` = ?", class.ID, userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("check attendance: %v", err))
	}
	if attendances > 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "You have already attended in this class.")
	}

	// 出席コード登録
	if _, err := h.DB.Exec("INSERT INTO `attendances` (`class_id`, `user_id`, `created_at`) VALUES (?, ?, NOW(6))", class.ID, userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("create attendance: %v", err))
	}

	return context.NoContent(http.StatusNoContent)
}
