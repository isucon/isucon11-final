package main

import (
	"database/sql"
	"fmt"
	"github.com/pborman/uuid"
	"golang.org/x/crypto/bcrypt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	SQLDirectory = "../sql/"
	SessionName  = "session"
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
			coursesAPI.GET("/:courseID/assignments", h.GetAssignmentList)
			coursesAPI.POST("/:courseID/assignments", h.AddAssignment, h.IsAdmin)
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
		attendanceCodeAPI := API.Group("/code")
		{
			attendanceCodeAPI.POST("", h.CheckAttendanceCode)
		}
	}

	e.Logger.Error(e.StartServer(e.Server))
}

func (h *handlers) Initialize(c echo.Context) error {
	dbForInit, _ := GetDB(true)

	files := []string{
		"schema.sql",
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

type UserType string

const (
	_ UserType = "student" /* FIXME: use Student */
	Faculty UserType = "faculty"
)

type User struct {
	ID             uuid.UUID `db:"id"`
	Name           string    `db:"name"`
	MailAddress    string    `db:"mail_address"`
	HashedPassword []byte    `db:"hashed_password"`
	Type           UserType  `db:"type"`
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

func (h *handlers) AddAssignment(context echo.Context) error {
	panic("implement me")
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

func (h *handlers) SubmitAssignment(context echo.Context) error {
	panic("implement me")
}

func (h *handlers) DownloadSubmittedAssignment(context echo.Context) error {
	panic("implement me")
}

func (h *handlers) GetAttendanceCode(context echo.Context) error {
	panic("implement me")
}

func (h *handlers) SetClassFlag(context echo.Context) error {
	panic("implement me")
}

func (h *handlers) GetAttendances(context echo.Context) error {
	panic("implement me")
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

func (h *handlers) CheckAttendanceCode(context echo.Context) error {
	panic("implement me")
}
