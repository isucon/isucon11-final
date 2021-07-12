package main

import (
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"math"

	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pborman/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	SQLDirectory           = "../sql/"
	AssignmentsDirectory   = "../assignments/"
	AssignmentTmpDirectory = "../assignments/tmp/"
	SessionName            = "session"
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

	e.POST("/login", h.Login)
	// API := e.Group("/api", h.IsLoggedIn)
	API := e.Group("/api")
	{
		usersAPI := API.Group("/users")
		{
			usersAPI.GET("/me/courses", h.GetRegisteredCourses)
			usersAPI.PUT("/me/courses", h.RegisterCourses)
			usersAPI.GET("/me/grades", h.GetGrades)
		}
		syllabusAPI := API.Group("/syllabus")
		{
			syllabusAPI.GET("", h.SearchCourses)
			syllabusAPI.GET("/:courseID", h.GetCourseDetail)
		}
		coursesAPI := API.Group("/courses")
		{
			coursesAPI.POST("", h.AddCourse, h.IsAdmin)
			coursesAPI.PUT("/:courseID/status", h.SetCourseStatus, h.IsAdmin)
			coursesAPI.GET("/:courseID/classes", h.GetClasses)
			coursesAPI.POST("/:courseID/classes", h.AddClass, h.IsAdmin)
			coursesAPI.POST("/:courseID/classes/:classID/assignment", h.SubmitAssignment)
			coursesAPI.GET("/:courseID/classes/:classID/export", h.DownloadSubmittedAssignment, h.IsAdmin)
			coursesAPI.GET("/:courseID/announcements", h.GetCourseAnnouncementList)
			coursesAPI.POST("/:courseID/announcements", h.AddAnnouncement, h.IsAdmin)
		}
		announcementsAPI := API.Group("/announcements")
		{
			announcementsAPI.GET("", h.GetAnnouncementList)
			announcementsAPI.GET("/:announcementID", h.GetAnnouncementDetail)
		}
	}

	e.Logger.Error(e.StartServer(e.Server))
}

type InitializeResponse struct {
	Language string `json:"language"`
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

type UserType string

const (
	Student UserType = "student"
	Teacher UserType = "teacher"
)

type User struct {
	ID             uuid.UUID `db:"id"`
	Code           string    `db:"code"`
	Name           string    `db:"name"`
	HashedPassword []byte    `db:"hashed_password"`
	Type           UserType  `db:"type"`
}

type Course struct {
	ID          uuid.UUID    `db:"id"`
	Code        string       `db:"code"`
	Type        string       `db:"type"`
	Name        string       `db:"name"`
	Description string       `db:"description"`
	Credit      uint8        `db:"credit"`
	Period      uint8        `db:"period"`
	DayOfWeek   string       `db:"day_of_week"`
	TeacherID   uuid.UUID    `db:"teacher_id"`
	Keywords    string       `db:"keywords"`
	Status      CourseStatus `db:"status"`
	CreatedAt   time.Time    `db:"created_at"`
}

type CourseStatus string

const (
	StatusRegistration CourseStatus = "registration"
	StatusInProgress   CourseStatus = "in-progress"
	StatusClosed       CourseStatus = "closed"
)

type Class struct {
	ID                 uuid.UUID    `db:"id"`
	CourseID           uuid.UUID    `db:"course_id"`
	Part               uint8        `db:"part"`
	Title              string       `db:"title"`
	Description        string       `db:"description"`
	CreatedAt          time.Time    `db:"created_at"`
	SubmissionClosedAt sql.NullTime `db:"submission_closed_at"`
}

type Assignment struct {
	ID          uuid.UUID `db:"id"`
	ClassID     uuid.UUID `db:"class_id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
}

type Announcement struct {
	ID         uuid.UUID `db:"id"`
	CourseID   uuid.UUID `db:"course_id"`
	CourseName string    `db:"name"`
	Title      string    `db:"title"`
	Message    string    `db:"message"`
	Read       bool      `db:"read"`
	CreatedAt  time.Time `db:"created_at"`
}

type SubmissionWithUserName struct {
	ID           uuid.UUID `db:"id"`
	UserID       uuid.UUID `db:"user_id"`
	UserName     string    `db:"user_name"`
	AssignmentID uuid.UUID `db:"assignment_id"`
	Name         string    `db:"name"`
	CreatedAt    time.Time `db:"created_at"`
}

type LoginRequest struct {
	Code     string `json:"code"`
	Password string `json:"password"`
}

func (h *handlers) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("bind request: %v", err))
	}

	var user User
	err := h.DB.Get(&user, "SELECT * FROM `users` WHERE `code` = ?", req.Code)
	if err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusUnauthorized, "Code or Password is wrong.")
	} else if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if bcrypt.CompareHashAndPassword(user.HashedPassword, []byte(req.Password)) != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Code or Password is wrong.")
	}

	sess, err := session.Get(SessionName, c)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if s, ok := sess.Values["userID"].(string); ok {
		userID := uuid.Parse(s)
		if uuid.Equal(userID, user.ID) {
			return echo.NewHTTPError(http.StatusBadRequest, "You are already logged in.")
		}
	}

	sess.Values["userID"] = user.ID.String()
	sess.Values["userName"] = user.Name
	sess.Values["isAdmin"] = user.Type == Teacher
	sess.Options = &sessions.Options{
		Path:   "/",
		MaxAge: 3600,
	}

	if err := sess.Save(c.Request(), c.Response()); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}

type GetRegisteredCourseResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Teacher   string    `json:"teacher"`
	Period    uint8     `json:"period"`
	DayOfWeek string    `json:"day_of_week"`
}

func (h *handlers) GetRegisteredCourses(c echo.Context) error {
	sess, err := session.Get(SessionName, c)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	userID := uuid.Parse(sess.Values["userID"].(string))
	if userID == nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var courses []Course
	if err = h.DB.Select(&courses, "SELECT `courses`.* "+
		"FROM `courses` "+
		"JOIN `registrations` ON `courses`.`id` = `registrations`.`course_id` "+
		"WHERE `courses`.`status` != ? AND `registrations`.`user_id` = ?", StatusClosed, userID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	res := make([]GetRegisteredCourseResponse, 0, len(courses))
	for _, course := range courses {
		var teacher User
		if err := h.DB.Get(&teacher, "SELECT * FROM `users` WHERE `id` = ?", course.TeacherID); err != nil {
			c.Logger().Error(err)
			return c.NoContent(http.StatusInternalServerError)
		}

		res = append(res, GetRegisteredCourseResponse{
			ID:        course.ID,
			Name:      course.Name,
			Teacher:   teacher.Name,
			Period:    course.Period,
			DayOfWeek: course.DayOfWeek,
		})
	}

	return c.JSON(http.StatusOK, res)
}

type RegisterCourseRequest struct {
	ID string `json:"id"`
}

type RegisterCoursesRequest []RegisterCourseRequest

type RegisterCoursesErrorResponse struct {
	NotFoundCourse        []string    `json:"not_found_course,omitempty"`
	StatusNotRegistration []uuid.UUID `json:"status_not_registration,omitempty"`
	TimeslotDuplicated    []uuid.UUID `json:"timeslot_duplicated,omitempty"`
}

func (h *handlers) RegisterCourses(c echo.Context) error {
	sess, err := session.Get(SessionName, c)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	userID := uuid.Parse(sess.Values["userID"].(string))
	if userID == nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var req RegisterCoursesRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("bind request: %v", err))
	}

	tx, err := h.DB.Beginx()
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var errors RegisterCoursesErrorResponse
	var coursesToRegister []Course
	for _, courseReq := range req {
		courseID := uuid.Parse(courseReq.ID)
		if courseID == nil {
			errors.NotFoundCourse = append(errors.NotFoundCourse, courseReq.ID)
			continue
		}

		var course Course
		if err := tx.Get(&course, "SELECT * FROM `courses` WHERE `id` = ? FOR SHARE", courseID); err == sql.ErrNoRows {
			errors.NotFoundCourse = append(errors.NotFoundCourse, courseReq.ID)
			continue
		} else if err != nil {
			c.Logger().Error(err)
			_ = tx.Rollback()
			return c.NoContent(http.StatusInternalServerError)
		}

		if course.Status != StatusRegistration {
			errors.StatusNotRegistration = append(errors.StatusNotRegistration, course.ID)
			continue
		}

		// MEMO: すでに履修登録済みの科目は無視する
		var registerCount int
		if err := tx.Get(&registerCount, "SELECT COUNT(*) FROM `registrations` WHERE `course_id` = ? AND `user_id` = ?", course.ID, userID); err != nil {
			c.Logger().Error(err)
			_ = tx.Rollback()
			return c.NoContent(http.StatusInternalServerError)
		}
		if registerCount > 0 {
			continue
		}

		coursesToRegister = append(coursesToRegister, course)
	}

	// MEMO: スケジュールの重複バリデーション
	var registeredCourses []Course
	if err := tx.Select(&registeredCourses, "SELECT `courses`.* "+
		"FROM `courses` "+
		"JOIN `registrations` ON `courses`.`id` = `registrations`.`course_id` "+
		"WHERE `courses`.`status` != ? AND `registrations`.`user_id` = ?", StatusClosed, userID); err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return c.NoContent(http.StatusInternalServerError)
	}

	registeredCourses = append(registeredCourses, coursesToRegister...)

	for _, course1 := range coursesToRegister {
		for _, course2 := range registeredCourses {
			if !uuid.Equal(course1.ID, course2.ID) && course1.Period == course2.Period && course1.DayOfWeek == course2.DayOfWeek {
				errors.TimeslotDuplicated = append(errors.TimeslotDuplicated, course1.ID)
				break
			}
		}
	}

	if len(errors.NotFoundCourse) > 0 || len(errors.StatusNotRegistration) > 0 || len(errors.TimeslotDuplicated) > 0 {
		_ = tx.Rollback()
		return c.JSON(http.StatusBadRequest, errors)
	}

	for _, course := range coursesToRegister {
		_, err = tx.Exec("INSERT INTO `registrations` (`course_id`, `user_id`, `created_at`) VALUES (?, ?, NOW(6))", course.ID, userID)
		if err != nil {
			c.Logger().Error(err)
			_ = tx.Rollback()
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	if err = tx.Commit(); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}

type GetGptResponse struct {
	Summary       Summary        `json:"summary"`
	CourseResults []CourseResult `json:"courses"`
}

type Summary struct {
	GPT     uint32  `json:"gpt"`
	Credits int     `json:"credits"`
	GptDev  uint32  `json:"gpt_dev"`
	GptAvg  float64 `json:"gpt_avg"`
	GptMax  uint32  `json:"gpt_max"`
	GptMin  uint32  `json:"gpt_min"`
}

type CourseResult struct {
	Name          string       `json:"name"`
	Code          string       `json:"code"`
	TotalScore    uint32       `json:"total_score"`
	TotalScoreDev uint32       `json:"gpt_dev"`
	TotalScoreAvg float64      `json:"gpt_avg"`
	TotalScoreMax uint32       `json:"gpt_max"`
	TotalScoreMin uint32       `json:"gpt_min"`
	ClassScores   []ClassScore `json:"class_scores"`
}

type ClassScore struct {
	ClassID    uuid.UUID `json:"class_id"`
	Title      string    `json:"title"`
	Part       uint8     `json:"part"`
	Score      uint32    `json:"score"`
	Submitters uint8     `json:"submitters"`
}

type SubmissionWithClassName struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	ClassID   uuid.UUID `db:"class_id"`
	Name      string    `db:"name"`
	Score     uint32    `db:"score"`
	CreatedAt time.Time `db:"created_at"`
	Part      uint8     `db:"part"`
	Title     string    `db:"title"`
}

type GPT struct {
	GPT uint32 `db:"gpt"`
}

type TotalScore struct {
	TotalScore float64 `db:"total_score"`
}

func (h *handlers) GetGrades(context echo.Context) error {
	sess, err := session.Get(SessionName, context)
	if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}
	userID := uuid.Parse(sess.Values["userID"].(string))
	if userID == nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	res := GetGptResponse{}

	//登録済のコース一覧取得

	var registeredCourses []Course
	if err := h.DB.Select(&registeredCourses,
		"SELECT `courses`.* "+
			"FROM `registrations` "+
			"JOIN `courses` ON `registrations`.`couse`.`id` = `courses`.`id` "+
			"WHERE `user_id` = ?", userID); err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	//Courses以下の取得
	//登録済コースごとループ
	//人気コースが分裂しても各種idはuniqueだから問題ないよね？
	//以下uniqueな自分の受講したコースに紐ついたクラス/課題をSelectしていく
	for _, course := range registeredCourses {
		//学生ごとのTotalScore取得
		var TotalScores []TotalScore
		if err := h.DB.Select(&TotalScores,
			"SELECT SUM(`submissions`.`score`) AS `total_score` "+
				"FROM `submissions` "+
				"JOIN `classses` ON `submiisons`.`class_id` = `classes`.`id` "+
				"WHERE `classes`.`course_id`= ? "+
				"GROUP BY `user_id`", course.ID); err != nil {
			log.Println(err)
			return context.NoContent(http.StatusInternalServerError)
		}

		//avg max min stdの計算
		total_score_count := len(TotalScores)
		total_score_avg := 0.0
		total_score_max := 0
		total_score_min := 999 //1コース5クラスなので最大500点
		total_score_std := 0.0
		for _, totalScore := range TotalScores {
			total_score_avg += totalScore.TotalScore / float64(total_score_count)
			if total_score_max > int(totalScore.TotalScore) {
				total_score_max = int(totalScore.TotalScore)
			}
			if total_score_min < int(totalScore.TotalScore) {
				total_score_min = int(totalScore.TotalScore)
			}
		}

		for _, totalScore := range TotalScores {
			total_score_std += math.Pow(totalScore.TotalScore-total_score_avg, 2) / float64(total_score_count)
		}
		total_score_std = math.Sqrt(total_score_std)

		//class情報の取得
		var classes []Class
		if err := h.DB.Select(&classes,
			"SELECT * "+
				"FROM `classes` "+
				"WHERE `classes`.`course_id` = ?", course.ID); err != nil {
			log.Println(err)
			return context.NoContent(http.StatusInternalServerError)
		}

		//submission情報の取得
		var myclassScores []ClassScore
		mytotalScore := 0
		for _, class := range classes {
			var submissions []SubmissionWithClassName
			if err := h.DB.Select(&submissions,
				"SELECT `submissions`.*, `classes`.`part` AS `part`, `classes`.`title` AS `title` "+
					"FROM `submissions` "+
					"WHERE `submission`.`class_id` = ? "+
					"JOIN `classes` ON `submissions`.`class_id` = `classes`.`id`", class.ID); err != nil {
				log.Println(err)
				return context.NoContent(http.StatusInternalServerError)

				//課題スコアを代入しつつ自分のコース成績計算
				//提出者数(submitters)とコース成績(mytotalScore)を計算するので上のクエリで一括取得できないのが面倒
				//->submitter消したい
				for _, submission := range submissions {
					if uuid.Equal(userID, submission.UserID) {
						myclassScores = append(myclassScores, ClassScore{
							Part:       submission.Part,
							Title:      submission.Title,
							Score:      uint32(submission.Score),
							Submitters: uint8(len(submission.ID)),
						})
						mytotalScore += int(submission.Score)
					}
				}

			}

		}

		//自分のコース偏差値の計算
		total_score_dev := (float64(mytotalScore)-total_score_avg)/total_score_std*10 + 50
		//コース情報の作成
		res.CourseResults = append(res.CourseResults, CourseResult{
			Name:          course.Name,
			Code:          course.Code,
			TotalScore:    uint32(mytotalScore),
			TotalScoreDev: uint32(total_score_dev),
			TotalScoreAvg: total_score_avg,
			TotalScoreMax: uint32(total_score_max),
			TotalScoreMin: uint32(total_score_min),
			ClassScores:   myclassScores,
		})

		//自分のGPT計算
		res.Summary.GPT += uint32(mytotalScore) * uint32(course.Credit) / 100
		res.Summary.Credits += int(course.Credit)

	}

	//GPTの統計値
	//全学生ごとのGPT
	var GPTList []GPT
	if err := h.DB.Select(&GPTList,
		"SELECT SUM(`submissions`.`score` * `courses`.`credit`/100) AS `gpt` " +
			"FROM `submissions` " +
			"JOIN `classes` ON 'submissions'.'class_id' = `class`.`id` " +
			"JOIN `courses` ON `classes'.'course_id' = `courses`.`id` " +
			"GROUP BY `user_id`"); err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	//avg max min stdの計算
	GPT_count := len(GPTList)
	GPT_avg := 0.0
	GPT_max := 0
	GPT_std := 0.0
	GPT_min := 9999 //1コース500点かつ5秒で20コースを12回転=(240コース)の1/100なので最大1200点
	for _, gpt := range GPTList {
		res.Summary.GptAvg += float64(gpt.GPT) / float64(GPT_count)

		if res.Summary.GptMax > uint32(gpt.GPT) {
			res.Summary.GptMax = uint32(gpt.GPT)
		}
		if res.Summary.GptMax < uint32(gpt.GPT) {
			res.Summary.GptMax = uint32(gpt.GPT)
		}
	}

	for _, gpt := range GPTList {
		GPT_std += math.Pow(float64(gpt.GPT)-res.Summary.GptAvg, 2) / float64(GPT_count)
	}
	GPT_std = math.Sqrt(GPT_std)

	//自分の偏差値の計算
	GPT_dev := uint32((float64(res.Summary.GPT)-res.Summary.GptAvg)/GPT_std*10 + 50)

	res.Summary = Summary{
		GptDev: uint32(GPT_dev),
		GptAvg: float64(GPT_avg),
		GptMax: uint32(GPT_max),
		GptMin: uint32(GPT_min),
	}

	return context.JSON(http.StatusOK, res)
}
func (h *handlers) SearchCourses(c echo.Context) error {
	query := "SELECT `courses`.`id`, `courses`.`code`, `courses`.`type`, `courses`.`name`, `courses`.`description`, `courses`.`credit`, `courses`.`period`, `courses`.`day_of_week`, `courses`.`keywords`, `users`.`name` AS `teacher`" +
		" FROM `courses` JOIN `users` ON `courses`.`teacher_id` = `users`.`id`" +
		" WHERE 1=1"
	var condition string
	var args []interface{}

	// MEMO: 検索条件はtype, credit, teacher, period, day_of_weekの完全一致とname, keywordsの部分一致

	if courseType := c.QueryParam("type"); courseType != "" {
		condition += " AND `courses`.`type` = ?"
		args = append(args, courseType)
	}

	if credit, err := strconv.Atoi(c.QueryParam("credit")); err == nil && credit > 0 {
		condition += " AND `courses`.`credit` = ?"
		args = append(args, credit)
	}

	if teacher := c.QueryParam("teacher"); teacher != "" {
		condition += " AND `users`.`name` = ?"
		args = append(args, teacher)
	}

	if period, err := strconv.Atoi(c.QueryParam("period")); err == nil && period > 0 {
		condition += " AND `courses`.`period` = ?"
		args = append(args, period)
	}

	if dayOfWeek := c.QueryParam("day_of_week"); dayOfWeek != "" {
		condition += " AND `courses`.`day_of_week` = ?"
		args = append(args, dayOfWeek)
	}

	// MEMO: 組み立てられたSQLはうまく動くけどリクエストには返ってこないからPlaceholderへの挿入がうまく言ってない気がする
	if keywords := c.QueryParam("keywords"); keywords != "" {
		arr := strings.Split(keywords, " ")
		var nameCondition string
		for _, keyword := range arr {
			nameCondition += " AND `courses`.`name` LIKE ?"
			args = append(args, "%"+keyword+"%")
		}
		var keywordsCondition string
		for _, keyword := range arr {
			keywordsCondition += " AND `courses`.`keywords` LIKE ?"
			args = append(args, "%"+keyword+"%")
		}
		condition += fmt.Sprintf(" AND ((1=1%s) OR (1=1%s))", nameCondition, keywordsCondition)
	}

	condition += " ORDER BY `courses`.`code` DESC"

	// MEMO: ページングの初期実装はページ番号形式
	var limit = 20
	var page int
	if page, err := strconv.Atoi(c.QueryParam("page")); err == nil && page > 0 {
		offset := limit * (page - 1)
		condition += " LIMIT ? OFFSET ?"
		args = append(args, limit+1, offset)
	}

	var res []GetCourseDetailResponse
	if err := h.DB.Select(&res, query+condition, args...); err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusNotFound, "No course found")
	} else if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var links []string
	path := fmt.Sprintf("%v://%v%v", c.Scheme(), c.Request().Host, c.Path())
	if page > 1 {
		links = append(links, fmt.Sprintf("<%v?page=%v>; rel=\"prev\"", path, page-1))
	}
	if len(res) > limit {
		links = append(links, fmt.Sprintf("<%v?page=%v>; rel=\"next\"", path, page+1))
	}
	if len(links) > 0 {
		c.Response().Header().Set("Link", strings.Join(links, ","))
	}

	if len(res) > limit {
		res = res[:len(res)-1]
	}

	return c.JSON(http.StatusOK, res)
}

type GetCourseDetailResponse struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Code        string    `json:"code" db:"code"`
	Type        string    `json:"type" db:"type"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Credit      uint8     `json:"credit" db:"credit"`
	Period      uint8     `json:"period" db:"period"`
	DayOfWeek   string    `json:"day_of_week" db:"day_of_week"`
	Teacher     string    `json:"teacher" db:"teacher"`
	Keywords    string    `json:"keywords" db:"keywords"`
}

func (h *handlers) GetCourseDetail(c echo.Context) error {
	courseID := c.Param("courseID")
	if courseID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "CourseID is required")
	}

	var res GetCourseDetailResponse
	query := "SELECT `courses`.`id`, `courses`.`code`, `courses`.`type`, `courses`.`name`, `courses`.`description`, `courses`.`credit`, `courses`.`period`, `courses`.`day_of_week`, `courses`.`keywords`, `users`.`name` AS `teacher`" +
		"FROM `courses`" +
		"JOIN `users` ON `courses`.`teacher_id` = `users`.`id`" +
		"WHERE `courses`.`id` = ?"
	if err := h.DB.Get(&res, query, courseID); err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusNotFound, "No such course")
	} else if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, res)
}

type AddCourseRequest struct {
	Code        string `json:"code"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Credit      int    `json:"credit"`
	Period      int    `json:"period"`
	DayOfWeek   string `json:"day_of_week"`
	Keywords    string `json:"keywords"`
}

type AddCourseResponse struct {
	ID uuid.UUID `json:"id"`
}

func (h *handlers) AddCourse(c echo.Context) error {
	sess, err := session.Get(SessionName, c)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	userID := uuid.Parse(sess.Values["userID"].(string))
	if userID == nil {
		return c.NoContent(http.StatusInternalServerError)
	}

	var req AddCourseRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("bind request: %s", err))
	}

	tx, err := h.DB.Beginx()
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	courseID := uuid.NewRandom()
	_, err = tx.Exec("INSERT INTO `courses` (`id`, `code`, `type`, `name`, `description`, `credit`, `period`, `day_of_week`, `teacher_id`, `keywords`, `created_at`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())",
		courseID, req.Code, req.Type, req.Name, req.Description, req.Credit, req.Period, req.DayOfWeek, userID, req.Keywords)
	if err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return c.NoContent(http.StatusInternalServerError)
	}

	announcementID := uuid.NewRandom()
	_, err = tx.Exec("INSERT INTO `announcements` (`id`, `course_id`, `title`, `message`, `created_at`) VALUES (?, ?, ?, ?, NOW())",
		announcementID, courseID, fmt.Sprintf("コース追加: %s", req.Name), fmt.Sprintf("コースが新しく追加されました: %s\n%s", req.Name, req.Description))
	if err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return c.NoContent(http.StatusInternalServerError)
	}

	var users []*User
	if err := tx.Select(&users, "SELECT * FROM `users` WHERE `type` = ?", Student); err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return c.NoContent(http.StatusInternalServerError)
	}

	// MEMO: N+1だけど最初から無くても良いかもしれない
	for _, user := range users {
		_, err := tx.Exec("INSERT INTO `unread_announcements` (`announcement_id`, `user_id`, `created_at`) VALUES (?, ?, NOW())",
			announcementID, user.ID)
		if err != nil {
			c.Logger().Error(err)
			_ = tx.Rollback()
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	if err := tx.Commit(); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusCreated, AddCourseResponse{ID: courseID})
}

type SetCourseStatusRequest struct {
	Status string `json:"status"`
}

func (h *handlers) SetCourseStatus(c echo.Context) error {
	courseID := uuid.Parse(c.Param("courseID"))
	if courseID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid courseID")
	}

	var req SetCourseStatusRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("bind request: %s", err))
	}

	if _, err := h.DB.Exec("UPDATE `courses` SET `status` = ? WHERE `id` = ?", req.Status, courseID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}

type GetClassResponse struct {
	ID                 uuid.UUID `json:"id"`
	Part               uint8     `json:"part"`
	Title              string    `json:"title"`
	Description        string    `json:"description"`
	SubmissionClosedAt int64     `json:"submission_closed_at,omitempty"`
}

type GetClassesResponse []GetClassResponse

func (h *handlers) GetClasses(c echo.Context) error {
	courseID := c.Param("courseID")
	var count int
	if err := h.DB.Get(&count, "SELECT COUNT(*) FROM `courses` WHERE `id` = ?", courseID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if count == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "no such course")
	}

	var classes []Class
	if err := h.DB.Select(&classes, "SELECT * FROM `classes` WHERE `course_id` = ? ORDER BY `part`", courseID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	res := make(GetClassesResponse, 0, len(classes))
	for _, class := range classes {
		getClassRes := GetClassResponse{
			ID:          class.ID,
			Part:        class.Part,
			Title:       class.Title,
			Description: class.Description,
		}
		if class.SubmissionClosedAt.Valid {
			getClassRes.SubmissionClosedAt = class.SubmissionClosedAt.Time.UnixNano() / int64(time.Millisecond)
		}

		res = append(res, getClassRes)
	}

	return c.JSON(http.StatusOK, res)
}

func (h *handlers) SubmitAssignment(c echo.Context) error {
	sess, err := session.Get(SessionName, c)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	userID := uuid.Parse(sess.Values["userID"].(string))
	if userID == nil {
		return c.NoContent(http.StatusInternalServerError)
	}

	courseID := uuid.Parse(c.Param("courseID"))
	if courseID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid courseID")
	}

	var course Course
	if err := h.DB.Get(&course, "SELECT * FROM `courses` WHERE `id` = ?", courseID); err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusBadRequest, "No such course.")
	} else if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if course.Status != StatusInProgress {
		return echo.NewHTTPError(http.StatusBadRequest, "The course is not in progress.")
	}

	classID := c.Param("classID")
	var class Class
	if err := h.DB.Get(&class, "SELECT * FROM `classes` WHERE `id` = ?", classID); err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusBadRequest, "No such class.")
	} else if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	src, err := file.Open()
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	defer src.Close()

	submissionID := uuid.New()
	dst, err := os.Create(AssignmentsDirectory + submissionID)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if _, err := h.DB.Exec("INSERT INTO `submissions` (`id`, `user_id`, `class_id`, `name`, `created_at`) VALUES (?, ?, ?, ?, NOW())", submissionID, userID, classID, file.Filename); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *handlers) DownloadSubmittedAssignment(c echo.Context) error {
	courseID := uuid.Parse(c.Param("courseID"))
	if courseID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid courseID")
	}

	classID := uuid.Parse(c.Param("classID"))
	if classID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid classID")
	}

	tx, err := h.DB.Beginx()
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	// MEMO: zipファイルを作るためFOR UPDATEでclass、FOR SHAREでsubmissionをロック
	var assignment Assignment
	if err := tx.Get(&assignment, "SELECT * FROM `classes` WHERE `id` = ? FOR UPDATE", classID); err == sql.ErrNoRows {
		_ = tx.Rollback()
		return echo.NewHTTPError(http.StatusBadRequest, "No such assignment.")
	} else if err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	var submissions []*SubmissionWithUserName
	if err := tx.Select(&submissions,
		"SELECT `submissions`.*, `users`.`name` AS `user_name` FROM `submissions` JOIN `users` ON `users`.`id` = `submissions`.`user_id` WHERE `class_id` = ? ORDER BY `user_id` FOR SHARE", classID); err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	// MEMO: TODO: export時でなく提出時にzipファイルを作ることでボトルネックを作りたいが、「そうはならんやろ」という気持ち
	zipFilePath := AssignmentTmpDirectory + classID.String() + ".zip"
	if err := createSubmissionsZip(zipFilePath, submissions); err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	if err := tx.Commit(); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.File(zipFilePath)
}

type AddClassRequest struct {
	Part        uint8  `json:"part"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type AddClassResponse struct {
	ID uuid.UUID `json:"id"`
}

func (h *handlers) AddClass(c echo.Context) error {
	courseID := c.Param("courseID")
	var count int
	if err := h.DB.Get(&count, "SELECT COUNT(*) FROM `courses` WHERE `id` = ?", courseID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if count == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "no such course")
	}

	var req AddClassRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("bind request: %v", err))
	}

	tx, err := h.DB.Beginx()
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	classID := uuid.NewRandom()
	if _, err := tx.Exec("INSERT INTO `classes` (`id`, `course_id`, `part`, `title`, `description`, `created_at`) VALUES (?, ?, ?, ?, ?, NOW(6))",
		classID, courseID, req.Part, req.Title, req.Description); err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return c.NoContent(http.StatusInternalServerError)
	}

	announcementID := uuid.NewRandom()
	_, err = tx.Exec("INSERT INTO `announcements` (`id`, `course_id`, `title`, `message`, `created_at`) VALUES (?, ?, ?, ?, NOW(6))",
		announcementID, courseID, fmt.Sprintf("クラス追加: %s", req.Title), fmt.Sprintf("クラスが新しく追加されました: %s\n%s", req.Title, req.Description))
	if err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return c.NoContent(http.StatusInternalServerError)
	}

	// MEMO: 履修登録しているユーザにお知らせを追加
	var registeredUsers []User
	if err := tx.Select(&registeredUsers, "SELECT `users`.* FROM `users` "+
		"JOIN `registrations` ON `users`.`id` = `registrations`.`user_id` "+
		"WHERE `registrations`.`course_id` = ?", courseID); err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return c.NoContent(http.StatusInternalServerError)
	}

	for _, user := range registeredUsers {
		if _, err := tx.Exec("INSERT INTO `unread_announcements` (`announcement_id`, `user_id`, `created_at`) VALUES (?, ?, NOW(6))",
			announcementID, user.ID); err != nil {
			c.Logger().Error(err)
			_ = tx.Rollback()
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	if err := tx.Commit(); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	res := AddClassResponse{
		ID: classID,
	}

	return c.JSON(http.StatusCreated, res)
}

type GetCourseAnnouncementResponse struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Read      bool      `json:"read"`
	CreatedAt int64     `json:"created_at"`
}

type GetCourseAnnouncementsResponse []GetCourseAnnouncementResponse

func (h *handlers) GetCourseAnnouncementList(c echo.Context) error {
	sess, err := session.Get(SessionName, c)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	userID := uuid.Parse(sess.Values["userID"].(string))
	if userID == nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	courseID := c.Param("courseID")
	var count int
	if err := h.DB.Get(&count, "SELECT COUNT(*) FROM `courses` WHERE `id` = ?", courseID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if count == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "No such course.")
	}

	// MEMO: ページングの初期実装はページ番号形式
	var page int
	if c.QueryParam("page") == "" {
		page = 1
	} else {
		page, err = strconv.Atoi(c.QueryParam("page"))
		if err != nil || page <= 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid page.")
		}
	}
	limit := 20
	offset := limit * (page - 1)

	announcements := make([]Announcement, 0)
	if err := h.DB.Select(&announcements, "SELECT `announcements`.`id`, `courses`.`id` AS `course_id`, `courses`.`name`, `announcements`.`title`, `announcements`.`message`, `unread_announcements`.`deleted_at` IS NOT NULL AS `read`, `announcements`.`created_at` "+
		"FROM `announcements` "+
		"JOIN `courses` ON `announcements`.`course_id` = `courses`.`id` "+
		"JOIN `unread_announcements` ON `announcements`.`id` = `unread_announcements`.`announcement_id` "+
		"WHERE `announcements`.`course_id` = ? AND `unread_announcements`.`user_id` = ? "+
		"ORDER BY `announcements`.`created_at` DESC "+
		"LIMIT ? OFFSET ?", courseID, userID, limit+1, offset); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	lenRes := len(announcements)
	if len(announcements) == limit+1 {
		lenRes = limit
	}
	res := make(GetCourseAnnouncementsResponse, 0, lenRes)
	for _, announcement := range announcements[:lenRes] {
		res = append(res, GetCourseAnnouncementResponse{
			ID:        announcement.ID,
			Title:     announcement.Title,
			Read:      announcement.Read,
			CreatedAt: announcement.CreatedAt.UnixNano() / int64(time.Millisecond),
		})
	}

	if lenRes > 0 {
		var links []string
		url := fmt.Sprintf("%v://%v%v", c.Scheme(), c.Request().Host, c.Request().URL.Path)
		if page > 1 {
			links = append(links, fmt.Sprintf("<%v?page=%v>; rel=\"prev\"", url, page-1))
		}
		if len(announcements) == limit+1 {
			links = append(links, fmt.Sprintf("<%v?page=%v>; rel=\"next\"", url, page+1))
		}
		if len(links) > 0 {
			c.Response().Header().Set("Link", strings.Join(links, ","))
		}
	}

	return c.JSON(http.StatusOK, res)
}

type AddAnnouncementRequest struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

type AddAnnouncementResponse struct {
	ID uuid.UUID `json:"id"`
}

func (h *handlers) AddAnnouncement(c echo.Context) error {
	courseID := c.Param("courseID")
	var count int
	if err := h.DB.Get(&count, "SELECT COUNT(*) FROM `courses` WHERE `id` = ?", courseID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if count == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "No such course.")
	}

	var req AddAnnouncementRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("bind request: %v", err))
	}

	tx, err := h.DB.Beginx()
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	announcementID := uuid.NewRandom()
	if _, err := tx.Exec("INSERT INTO `announcements` (`id`, `course_id`, `title`, `message`, `created_at`) VALUES (?, ?, ?, ?, NOW(6))",
		announcementID, courseID, req.Title, req.Message); err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return c.NoContent(http.StatusInternalServerError)
	}

	// MEMO: 履修登録しているユーザにお知らせを追加
	var registeredUsers []User
	if err := tx.Select(&registeredUsers, "SELECT `users`.* FROM `users` "+
		"JOIN `registrations` ON `users`.`id` = `registrations`.`user_id` "+
		"WHERE `registrations`.`course_id` = ?", courseID); err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return c.NoContent(http.StatusInternalServerError)
	}

	for _, user := range registeredUsers {
		if _, err := tx.Exec("INSERT INTO `unread_announcements` (`announcement_id`, `user_id`, `created_at`) VALUES (?, ?, NOW(6))",
			announcementID, user.ID); err != nil {
			c.Logger().Error(err)
			_ = tx.Rollback()
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	if err := tx.Commit(); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	res := AddAnnouncementResponse{
		ID: announcementID,
	}

	return c.JSON(http.StatusCreated, res)
}

type GetAnnouncementResponse struct {
	ID         uuid.UUID `json:"id"`
	CourseID   uuid.UUID `json:"course_id"`
	CourseName string    `json:"course_name"`
	Title      string    `json:"title"`
	Read       bool      `json:"read"`
	CreatedAt  int64     `json:"created_at"`
}

type GetAnnouncementsResponse []GetAnnouncementResponse

func (h *handlers) GetAnnouncementList(c echo.Context) error {
	sess, err := session.Get(SessionName, c)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	userID := uuid.Parse(sess.Values["userID"].(string))
	if userID == nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	// MEMO: ページングの初期実装はページ番号形式
	var page int
	if c.QueryParam("page") == "" {
		page = 1
	} else {
		page, err = strconv.Atoi(c.QueryParam("page"))
		if err != nil || page <= 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid page.")
		}
	}
	limit := 20
	offset := limit * (page - 1)

	announcements := make([]Announcement, 0)
	if err := h.DB.Select(&announcements, "SELECT `announcements`.`id`, `courses`.`id` AS `course_id`, `courses`.`name`, `announcements`.`title`, `announcements`.`message`, `unread_announcements`.`deleted_at` IS NOT NULL AS `read`, `announcements`.`created_at` "+
		"FROM `announcements` "+
		"JOIN `courses` ON `announcements`.`course_id` = `courses`.`id` "+
		"JOIN `unread_announcements` ON `announcements`.`id` = `unread_announcements`.`announcement_id` "+
		"WHERE `unread_announcements`.`user_id` = ? "+
		"ORDER BY `announcements`.`created_at` DESC "+
		"LIMIT ? OFFSET ?", userID, limit+1, offset); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	lenRes := len(announcements)
	if len(announcements) == limit+1 {
		lenRes = limit
	}
	res := make(GetAnnouncementsResponse, 0, lenRes)
	for _, announcement := range announcements[:lenRes] {
		res = append(res, GetAnnouncementResponse{
			ID:         announcement.ID,
			CourseID:   announcement.CourseID,
			CourseName: announcement.CourseName,
			Title:      announcement.Title,
			Read:       announcement.Read,
			CreatedAt:  announcement.CreatedAt.UnixNano() / int64(time.Millisecond),
		})
	}

	if lenRes > 0 {
		var links []string
		url := fmt.Sprintf("%v://%v%v", c.Scheme(), c.Request().Host, c.Request().URL.Path)
		if page > 1 {
			links = append(links, fmt.Sprintf("<%v?page=%v>; rel=\"prev\"", url, page-1))
		}
		if len(announcements) == limit+1 {
			links = append(links, fmt.Sprintf("<%v?page=%v>; rel=\"next\"", url, page+1))
		}
		if len(links) > 0 {
			c.Response().Header().Set("Link", strings.Join(links, ","))
		}
	}

	return c.JSON(http.StatusOK, res)
}

type GetAnnouncementDetailResponse struct {
	ID         uuid.UUID `json:"id"`
	CourseID   uuid.UUID `json:"course_id"`
	CourseName string    `json:"course_name"`
	Title      string    `json:"title"`
	Message    string    `json:"message"`
	CreatedAt  int64     `json:"created_at"`
}

func (h *handlers) GetAnnouncementDetail(c echo.Context) error {
	sess, err := session.Get(SessionName, c)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	userID := uuid.Parse(sess.Values["userID"].(string))
	if userID == nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	announcementID := c.Param("announcementID")
	var announcement Announcement
	if err := h.DB.Get(&announcement, "SELECT `announcements`.`id`, `courses`.`id` AS `course_id`, `courses`.`name`, `announcements`.`title`, `announcements`.`message`, `announcements`.`created_at` "+
		"FROM `announcements` "+
		"JOIN `courses` ON `announcements`.`course_id` = `courses`.`id` "+
		"JOIN `registrations` ON `announcements`.`course_id` = `registrations`.`course_id` "+
		"WHERE `announcements`.`id` = ? AND `registrations`.`user_id` = ?", announcementID, userID); err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusNotFound, "no such announcement")
	} else if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if _, err := h.DB.Exec("UPDATE `unread_announcements` SET `deleted_at` = NOW(6) WHERE `announcement_id` = ? AND `user_id` = ? AND `deleted_at` IS NULL",
		announcementID, userID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	res := GetAnnouncementDetailResponse{
		ID:         announcement.ID,
		CourseID:   announcement.CourseID,
		CourseName: announcement.CourseName,
		Title:      announcement.Title,
		Message:    announcement.Message,
		CreatedAt:  announcement.CreatedAt.UnixNano() / int64(time.Millisecond),
	}
	return c.JSON(http.StatusOK, res)
}

func createSubmissionsZip(zipFilePath string, submissions []*SubmissionWithUserName) error {
	// Zipに含めるファイルの名称変更のためコピー
	// MEMO: N回 cp はやりすぎかも
	for _, submission := range submissions {
		cpCmd := exec.Command(
			"cp",
			AssignmentsDirectory+submission.ID.String(),
			AssignmentTmpDirectory+submission.UserName+"-"+submission.ID.String()+"-"+submission.Name,
		)
		if err := cpCmd.Start(); err != nil {
			return err
		}
		if err := cpCmd.Wait(); err != nil {
			return err
		}
	}

	zipArgs := make([]string, 0, len(submissions)+2)
	zipArgs = append(zipArgs, "-j", zipFilePath)
	for _, submission := range submissions {
		zipArgs = append(zipArgs, AssignmentTmpDirectory+submission.UserName+"-"+submission.ID.String()+"-"+submission.Name)
	}
	cmd := exec.Command("zip", zipArgs...)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Wait()
}
