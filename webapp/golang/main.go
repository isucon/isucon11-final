package main

import (
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
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
	SessionName          = "session"
)

type handlers struct {
	DB *sqlx.DB
}

func main() {
	e := echo.New()
	e.Debug = GetEnv("DEBUG", "") == "true"
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
			coursesAPI.POST("/:courseID/classes/:classID/assignments", h.RegisterScores, h.IsAdmin)
			coursesAPI.GET("/:courseID/classes/:classID/assignments/export", h.DownloadSubmittedAssignments, h.IsAdmin)
		}
		announcementsAPI := API.Group("/announcements")
		{
			announcementsAPI.GET("", h.GetAnnouncementList)
			announcementsAPI.POST("", h.AddAnnouncement, h.IsAdmin)
			announcementsAPI.GET("/:announcementID", h.GetAnnouncementDetail)
		}
	}

	e.Logger.Error(e.StartServer(e.Server))
}

type InitializeResponse struct {
	Language string `json:"language"`
}

// Initialize 初期化エンドポイント
func (h *handlers) Initialize(c echo.Context) error {
	dbForInit, _ := GetDB(true)

	files := []string{
		"1_schema.sql",
		"2_init.sql",
	}
	for _, file := range files {
		data, err := ioutil.ReadFile(SQLDirectory + file)
		if err != nil {
			c.Logger().Error(err)
			return c.NoContent(http.StatusInternalServerError)
		}
		if _, err := dbForInit.Exec(string(data)); err != nil {
			c.Logger().Error(err)
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	if err := exec.Command("rm", "-rf", AssignmentsDirectory).Run(); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if err := exec.Command("mkdir", AssignmentsDirectory).Run(); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	res := InitializeResponse{
		Language: "go",
	}
	return c.JSON(http.StatusOK, res)
}

// IsLoggedIn ログイン確認用middleware
func (h *handlers) IsLoggedIn(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, err := session.Get(SessionName, c)
		if err != nil {
			c.Logger().Error(err)
			return c.NoContent(http.StatusInternalServerError)
		}
		if sess.IsNew {
			return echo.NewHTTPError(http.StatusInternalServerError, "You are not logged in.")
		}
		if _, ok := sess.Values["userID"]; !ok {
			return echo.NewHTTPError(http.StatusInternalServerError, "You are not logged in.")
		}

		return next(c)
	}
}

// IsAdmin admin確認用middleware
func (h *handlers) IsAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, err := session.Get(SessionName, c)
		if err != nil {
			c.Logger().Error(err)
			return c.NoContent(http.StatusInternalServerError)
		}
		isAdmin, ok := sess.Values["isAdmin"]
		if !ok {
			c.Logger().Error(err)
			return c.NoContent(http.StatusInternalServerError)
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

type CourseType string

const (
	LiberalArts   CourseType = "liberal-arts"
	MajorSubjects CourseType = "major-subjects"
)

type DayOfWeek string

const (
	Sunday    DayOfWeek = "sunday"
	Monday    DayOfWeek = "monday"
	Tuesday   DayOfWeek = "tuesday"
	Wednesday DayOfWeek = "wednesday"
	Thursday  DayOfWeek = "thursday"
	Friday    DayOfWeek = "friday"
	Saturday  DayOfWeek = "saturday"
)

var daysOfWeek = []DayOfWeek{Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday}

type CourseStatus string

const (
	StatusRegistration CourseStatus = "registration"
	StatusInProgress   CourseStatus = "in-progress"
	StatusClosed       CourseStatus = "closed"
)

type Course struct {
	ID          uuid.UUID    `db:"id"`
	Code        string       `db:"code"`
	Type        CourseType   `db:"type"`
	Name        string       `db:"name"`
	Description string       `db:"description"`
	Credit      uint8        `db:"credit"`
	Period      uint8        `db:"period"`
	DayOfWeek   DayOfWeek    `db:"day_of_week"`
	TeacherID   uuid.UUID    `db:"teacher_id"`
	Keywords    string       `db:"keywords"`
	Status      CourseStatus `db:"status"`
	CreatedAt   time.Time    `db:"created_at"`
}

type LoginRequest struct {
	Code     string `json:"code"`
	Password string `json:"password"`
}

// Login ログイン
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

type GetRegisteredCourseResponseContent struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Teacher   string    `json:"teacher"`
	Period    uint8     `json:"period"`
	DayOfWeek DayOfWeek `json:"day_of_week"`
}

// GetRegisteredCourses 履修中の科目一覧取得
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
	query := "SELECT `courses`.*" +
		" FROM `courses`" +
		" JOIN `registrations` ON `courses`.`id` = `registrations`.`course_id`" +
		" WHERE `courses`.`status` != ? AND `registrations`.`user_id` = ?"
	if err = h.DB.Select(&courses, query, StatusClosed, userID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	// 履修科目が0件の時は空配列を返却
	res := make([]GetRegisteredCourseResponseContent, 0, len(courses))
	for _, course := range courses {
		var teacher User
		if err := h.DB.Get(&teacher, "SELECT * FROM `users` WHERE `id` = ?", course.TeacherID); err != nil {
			c.Logger().Error(err)
			return c.NoContent(http.StatusInternalServerError)
		}

		res = append(res, GetRegisteredCourseResponseContent{
			ID:        course.ID,
			Name:      course.Name,
			Teacher:   teacher.Name,
			Period:    course.Period,
			DayOfWeek: course.DayOfWeek,
		})
	}

	return c.JSON(http.StatusOK, res)
}

type RegisterCourseRequestContent struct {
	ID string `json:"id"`
}

type RegisterCoursesErrorResponse struct {
	CourseNotFound       []string    `json:"course_not_found,omitempty"`
	NotRegistrableStatus []uuid.UUID `json:"not_registrable_status,omitempty"`
	ScheduleConflict     []uuid.UUID `json:"schedule_conflict,omitempty"`
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

	var req []RegisterCourseRequestContent
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid format.")
	}
	sort.Slice(req, func(i, j int) bool {
		return req[i].ID < req[j].ID
	})

	tx, err := h.DB.Beginx()
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var errors RegisterCoursesErrorResponse
	var newlyAdded []Course
	for _, courseReq := range req {
		courseID := uuid.Parse(courseReq.ID)
		if courseID == nil {
			errors.CourseNotFound = append(errors.CourseNotFound, courseReq.ID)
			continue
		}

		var course Course
		if err := tx.Get(&course, "SELECT * FROM `courses` WHERE `id` = ? FOR SHARE", courseID); err == sql.ErrNoRows {
			errors.CourseNotFound = append(errors.CourseNotFound, courseReq.ID)
			continue
		} else if err != nil {
			c.Logger().Error(err)
			_ = tx.Rollback()
			return c.NoContent(http.StatusInternalServerError)
		}

		if course.Status != StatusRegistration {
			errors.NotRegistrableStatus = append(errors.NotRegistrableStatus, course.ID)
			continue
		}

		// MEMO: すでに履修登録済みの科目は無視する
		var count int
		if err := tx.Get(&count, "SELECT COUNT(*) FROM `registrations` WHERE `course_id` = ? AND `user_id` = ?", course.ID, userID); err != nil {
			c.Logger().Error(err)
			_ = tx.Rollback()
			return c.NoContent(http.StatusInternalServerError)
		}
		if count > 0 {
			continue
		}

		newlyAdded = append(newlyAdded, course)
	}

	// MEMO: スケジュールの重複バリデーション
	var alreadyRegistered []Course
	query := "SELECT `courses`.*" +
		" FROM `courses`" +
		" JOIN `registrations` ON `courses`.`id` = `registrations`.`course_id`" +
		" WHERE `courses`.`status` != ? AND `registrations`.`user_id` = ?"
	if err := tx.Select(&alreadyRegistered, query, StatusClosed, userID); err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return c.NoContent(http.StatusInternalServerError)
	}

	alreadyRegistered = append(alreadyRegistered, newlyAdded...)
	for _, course1 := range newlyAdded {
		for _, course2 := range alreadyRegistered {
			if !uuid.Equal(course1.ID, course2.ID) && course1.Period == course2.Period && course1.DayOfWeek == course2.DayOfWeek {
				errors.ScheduleConflict = append(errors.ScheduleConflict, course1.ID)
				break
			}
		}
	}

	if len(errors.CourseNotFound) > 0 || len(errors.NotRegistrableStatus) > 0 || len(errors.ScheduleConflict) > 0 {
		_ = tx.Rollback()
		return c.JSON(http.StatusBadRequest, errors)
	}

	for _, course := range newlyAdded {
		_, err = tx.Exec("INSERT INTO `registrations` (`course_id`, `user_id`, `created_at`) VALUES (?, ?, NOW())", course.ID, userID)
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

type GetGradeResponse struct {
	Summary       Summary        `json:"summary"`
	CourseResults []CourseResult `json:"courses"`
}

type Summary struct {
	Credits   int     `json:"credits"`
	GPT       float64 `json:"gpt"`
	GptTScore float64 `json:"gpt_t_score"` // 偏差値
	GptAvg    float64 `json:"gpt_avg"`     // 平均値
	GptMax    float64 `json:"gpt_max"`     // 最大値
	GptMin    float64 `json:"gpt_min"`     // 最小値
}

type CourseResult struct {
	Name             string       `json:"name"`
	Code             string       `json:"code"`
	TotalScore       int          `json:"total_score"`
	TotalScoreTScore float64      `json:"total_score_t_score"` // 偏差値
	TotalScoreAvg    float64      `json:"total_score_avg"`     // 平均値
	TotalScoreMax    int          `json:"total_score_max"`     // 最大値
	TotalScoreMin    int          `json:"total_score_min"`     // 最小値
	ClassScores      []ClassScore `json:"class_scores"`
}

type ClassScore struct {
	ClassID    uuid.UUID `json:"class_id"`
	Title      string    `json:"title"`
	Part       uint8     `json:"part"`
	Score      int       `json:"score"`      // 0~100点
	Submitters int       `json:"submitters"` // 提出した生徒数
}

type SubmissionWithClassName struct {
	UserID    uuid.UUID     `db:"user_id"`
	ClassID   uuid.UUID     `db:"class_id"`
	Name      string        `db:"file_name"`
	Score     sql.NullInt64 `db:"score"`
	CreatedAt time.Time     `db:"created_at"`
	Part      uint8         `db:"part"`
	Title     string        `db:"title"`
}

func (h *handlers) GetGrades(c echo.Context) error {
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

	var res GetGradeResponse

	// 登録済の科目一覧取得
	var registeredCourses []Course
	query := "SELECT `courses`.*" +
		" FROM `registrations`" +
		" JOIN `courses` ON `registrations`.`course_id` = `courses`.`id`" +
		" WHERE `user_id` = ?"
	if err := h.DB.Select(&registeredCourses, query, userID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	// 科目毎の成績計算処理
	res.CourseResults = make([]CourseResult, 0, len(registeredCourses))
	for _, course := range registeredCourses {
		// この科目を受講している学生のTotalScore一覧を取得
		var totals []int
		query := "SELECT IFNULL(SUM(`submissions`.`score`), 0) AS `total_score`" +
			" FROM `submissions`" +
			" JOIN `classes` ON `submissions`.`class_id` = `classes`.`id`" +
			" WHERE `classes`.`course_id` = ?" +
			" GROUP BY `user_id`"
		if err := h.DB.Select(&totals, query, course.ID); err != nil {
			c.Logger().Error(err)
			return c.NoContent(http.StatusInternalServerError)
		}

		// avg max min stdの計算
		var totalScoreCount = len(totals)
		var totalScoreAvg float64
		var totalScoreMax int
		var totalScoreMin = 500      // 1科目5クラスなので最大500点
		var totalScoreStdDev float64 // 標準偏差

		for _, totalScore := range totals {
			totalScoreAvg += float64(totalScore) / float64(totalScoreCount)
			if totalScoreMax < totalScore {
				totalScoreMax = totalScore
			}
			if totalScoreMin > totalScore {
				totalScoreMin = totalScore
			}
		}
		for _, totalScore := range totals {
			totalScoreStdDev += math.Pow(float64(totalScore)-totalScoreAvg, 2) / float64(totalScoreCount)
		}
		totalScoreStdDev = math.Sqrt(totalScoreStdDev)

		// クラス一覧の取得
		var classes []Class
		query = "SELECT *" +
			" FROM `classes`" +
			" WHERE `course_id` = ?" +
			" ORDER BY `part` DESC"
		if err := h.DB.Select(&classes, query, course.ID); err != nil {
			c.Logger().Error(err)
			return c.NoContent(http.StatusInternalServerError)
		}

		// クラス毎の成績計算処理
		classScores := make([]ClassScore, 0, len(classes))
		var myTotalScore int
		for _, class := range classes {
			var submissions []SubmissionWithClassName
			query := "SELECT `submissions`.*, `classes`.`part` AS `part`, `classes`.`title` AS `title`" +
				" FROM `submissions`" +
				" JOIN `classes` ON `submissions`.`class_id` = `classes`.`id`" +
				" WHERE `submissions`.`class_id` = ?"
			if err := h.DB.Select(&submissions, query, class.ID); err != nil {
				c.Logger().Error(err)
				return c.NoContent(http.StatusInternalServerError)
			}

			for _, submission := range submissions {
				if uuid.Equal(userID, submission.UserID) {
					var myScore int
					if submission.Score.Valid {
						myScore = int(submission.Score.Int64)
					}
					classScores = append(classScores, ClassScore{
						Part:       submission.Part,
						Title:      submission.Title,
						Score:      myScore,
						Submitters: len(submissions),
					})
					myTotalScore += myScore
				}
			}
		}

		// 対象科目の自分の偏差値の計算
		var totalScoreTScore float64
		if totalScoreStdDev == 0 {
			totalScoreTScore = 50
		} else {
			totalScoreTScore = (float64(myTotalScore)-totalScoreAvg)/totalScoreStdDev*10 + 50
		}

		res.CourseResults = append(res.CourseResults, CourseResult{
			Name:             course.Name,
			Code:             course.Code,
			TotalScore:       myTotalScore,
			TotalScoreTScore: totalScoreTScore,
			TotalScoreAvg:    totalScoreAvg,
			TotalScoreMax:    totalScoreMax,
			TotalScoreMin:    totalScoreMin,
			ClassScores:      classScores,
		})

		// 自分のGPT計算
		res.Summary.GPT += float64(myTotalScore*int(course.Credit)) / 100
		res.Summary.Credits += int(course.Credit)
	}

	// GPTの統計値
	// 全学生ごとのGPT
	var gpts []float64
	query = "SELECT IFNULL(SUM(`submissions`.`score` * `courses`.`credit` / 100), 0) AS `gpt`" +
		" FROM `users`" +
		" LEFT JOIN `submissions` ON `users`.`id` = `submissions`.`user_id`" +
		" LEFT JOIN `classes` ON `submissions`.`class_id` = `classes`.`id`" +
		" LEFT JOIN `courses` ON `classes`.`course_id` = `courses`.`id`" +
		" WHERE `users`.`type` = ?" +
		" GROUP BY `user_id`"
	if err := h.DB.Select(&gpts, query, Student); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	// avg max min stdの計算
	var gptCount = len(gpts)
	var gptAvg float64
	var gptMax float64
	// MEMO: 1コース500点かつ5秒で20コースを12回転=(240コース)の1/100なので最大1200点
	var gptMin = math.MaxFloat64
	var gptStdDev float64
	for _, gpt := range gpts {
		gptAvg += gpt / float64(gptCount)

		if gptMax < gpt {
			gptMax = gpt
		}
		if gptMin > gpt {
			gptMin = gpt
		}
	}

	for _, gpt := range gpts {
		gptStdDev += math.Pow(gpt-gptAvg, 2) / float64(gptCount)
	}
	gptStdDev = math.Sqrt(gptStdDev)

	// 自分の偏差値の計算
	var gptTScore float64
	if gptStdDev == 0 {
		gptTScore = 50
	} else {
		gptTScore = (res.Summary.GPT-gptAvg)/gptStdDev*10 + 50
	}

	res.Summary = Summary{
		GptTScore: gptTScore,
		GptAvg:    gptAvg,
		GptMax:    gptMax,
		GptMin:    gptMin,
	}

	return c.JSON(http.StatusOK, res)
}

// SearchCourses 科目検索
func (h *handlers) SearchCourses(c echo.Context) error {
	query := "SELECT `courses`.`id`, `courses`.`code`, `courses`.`type`, `courses`.`name`, `courses`.`description`, `courses`.`credit`, `courses`.`period`, `courses`.`day_of_week`, `courses`.`keywords`, `users`.`name` AS `teacher`" +
		" FROM `courses` JOIN `users` ON `courses`.`teacher_id` = `users`.`id`" +
		" WHERE 1=1"
	var condition string
	var args []interface{}

	// MEMO: 検索条件はtype, credit, teacher, period, day_of_weekの完全一致とname, keywordsの部分一致
	// 無効な検索条件はエラーを返さず無視して良い

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
	var page int
	if c.QueryParam("page") == "" {
		page = 1
	} else {
		var err error
		page, err = strconv.Atoi(c.QueryParam("page"))
		if err != nil || page <= 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid page")
		}
	}
	limit := 20
	offset := limit * (page - 1)

	// limitより多く上限を設定し、実際にlimitより多くレコードが取得できた場合は次のページが存在する
	condition += " LIMIT ? OFFSET ?"
	args = append(args, limit+1, offset)

	// 結果が0件の時は空配列を返却
	res := make([]GetCourseDetailResponse, 0)
	if err := h.DB.Select(&res, query+condition, args...); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var links []string
	url, err := url.Parse(c.Request().URL.String())
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	url.Scheme = c.Scheme()
	url.Host = c.Request().Host
	q := url.Query()
	if page > 1 {
		q.Set("page", strconv.Itoa(page-1))
		url.RawQuery = q.Encode()
		links = append(links, fmt.Sprintf("<%v>; rel=\"prev\"", url))
	}
	if len(res) > limit {
		q.Set("page", strconv.Itoa(page+1))
		url.RawQuery = q.Encode()
		links = append(links, fmt.Sprintf("<%v>; rel=\"next\"", url))
	}
	if len(links) > 0 {
		c.Response().Header().Set("Link", strings.Join(links, ","))
	}

	if len(res) == limit+1 {
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

// GetCourseDetail 科目詳細の取得
func (h *handlers) GetCourseDetail(c echo.Context) error {
	courseID := c.Param("courseID")
	if courseID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid CourseID.")
	}

	var res GetCourseDetailResponse
	query := "SELECT `courses`.`id`, `courses`.`code`, `courses`.`type`, `courses`.`name`, `courses`.`description`, `courses`.`credit`, `courses`.`period`, `courses`.`day_of_week`, `courses`.`keywords`, `users`.`name` AS `teacher`" +
		" FROM `courses`" +
		" JOIN `users` ON `courses`.`teacher_id` = `users`.`id`" +
		" WHERE `courses`.`id` = ?"
	if err := h.DB.Get(&res, query, courseID); err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusNotFound, "No such course.")
	} else if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, res)
}

type AddCourseRequest struct {
	Code        string     `json:"code"`
	Type        CourseType `json:"type"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Credit      int        `json:"credit"`
	Period      int        `json:"period"`
	DayOfWeek   DayOfWeek  `json:"day_of_week"`
	Keywords    string     `json:"keywords"`
}

type AddCourseResponse struct {
	ID uuid.UUID `json:"id"`
}

// AddCourse 新規科目登録
func (h *handlers) AddCourse(c echo.Context) error {
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

	var req AddCourseRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid format.")
	}

	if req.Type != LiberalArts && req.Type != MajorSubjects {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid course type.")
	}
	if !contains(daysOfWeek, req.DayOfWeek) {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid day of week.")
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

	if err := tx.Commit(); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusCreated, AddCourseResponse{ID: courseID})
}

type SetCourseStatusRequest struct {
	Status CourseStatus `json:"status"`
}

// SetCourseStatus 科目のステータスを変更
func (h *handlers) SetCourseStatus(c echo.Context) error {
	courseID := uuid.Parse(c.Param("courseID"))
	if courseID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid courseID.")
	}

	var req SetCourseStatusRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid format.")
	}

	result, err := h.DB.Exec("UPDATE `courses` SET `status` = ? WHERE `id` = ?", req.Status, courseID)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if num, err := result.RowsAffected(); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	} else if num == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "No such course.")
	}

	return c.NoContent(http.StatusOK)
}

type Class struct {
	ID                 uuid.UUID    `db:"id"`
	CourseID           uuid.UUID    `db:"course_id"`
	Part               uint8        `db:"part"`
	Title              string       `db:"title"`
	Description        string       `db:"description"`
	CreatedAt          time.Time    `db:"created_at"`
	SubmissionClosedAt sql.NullTime `db:"submission_closed_at"`
}

type GetClassResponse struct {
	ID                 uuid.UUID `json:"id"`
	Part               uint8     `json:"part"`
	Title              string    `json:"title"`
	Description        string    `json:"description"`
	SubmissionClosedAt int64     `json:"submission_closed_at,omitempty"`
}

// GetClasses 科目に紐づくクラス一覧の取得
func (h *handlers) GetClasses(c echo.Context) error {
	courseID := c.Param("courseID")
	if courseID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid CourseID.")
	}

	var count int
	if err := h.DB.Get(&count, "SELECT COUNT(*) FROM `courses` WHERE `id` = ?", courseID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if count == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "No such course.")
	}

	var classes []Class
	if err := h.DB.Select(&classes, "SELECT * FROM `classes` WHERE `course_id` = ? ORDER BY `part`", courseID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	// 結果が0件の時は空配列を返却
	res := make([]GetClassResponse, 0, len(classes))
	for _, class := range classes {
		getClassRes := GetClassResponse{
			ID:          class.ID,
			Part:        class.Part,
			Title:       class.Title,
			Description: class.Description,
		}
		if class.SubmissionClosedAt.Valid {
			getClassRes.SubmissionClosedAt = class.SubmissionClosedAt.Time.Unix()
		}

		res = append(res, getClassRes)
	}

	return c.JSON(http.StatusOK, res)
}

// SubmitAssignment 課題ファイルのアップロード
func (h *handlers) SubmitAssignment(c echo.Context) error {
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

	courseID := uuid.Parse(c.Param("courseID"))
	if courseID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid courseID.")
	}

	var status CourseStatus
	if err := h.DB.Get(&status, "SELECT `status` FROM `courses` WHERE `id` = ?", courseID); err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusBadRequest, "No such course.")
	} else if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if status != StatusInProgress {
		return echo.NewHTTPError(http.StatusBadRequest, "This course is not in progress.")
	}

	classID := uuid.Parse(c.Param("classID"))
	if classID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid classID.")
	}

	var count int
	if err := h.DB.Get(&count, "SELECT COUNT(*) FROM `classes` WHERE `id` = ?", classID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if count == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "No such class.")
	}

	file, header, err := c.Request().FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid file.")
	}
	defer file.Close()

	dst, err := os.Create(AssignmentsDirectory + classID.String() + "-" + userID.String())
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	defer dst.Close()

	if _, err = io.Copy(dst, file); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if _, err := h.DB.Exec("INSERT INTO `submissions` (`user_id`, `class_id`, `file_name`, `created_at`) VALUES (?, ?, ?, NOW())", userID, classID, header.Filename); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

type Score struct {
	UserCode string `json:"user_code"`
	Score    int    `json:"score"`
}

func (h *handlers) RegisterScores(c echo.Context) error {
	classID := uuid.Parse(c.Param("classID"))
	if classID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid classID.")
	}

	tx, err := h.DB.Beginx()
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var closedAt sql.NullTime
	if err := tx.Get(&closedAt, "SELECT `submission_closed_at` FROM `classes` WHERE `id` = ? FOR SHARE", classID); err == sql.ErrNoRows {
		_ = tx.Rollback()
		return echo.NewHTTPError(http.StatusBadRequest, "No such class.")
	} else if err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return c.NoContent(http.StatusInternalServerError)
	}

	if !closedAt.Valid {
		_ = tx.Rollback()
		return echo.NewHTTPError(http.StatusBadRequest, "This assignment is not closed yet.")
	}

	var req []Score
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid format.")
	}

	for _, score := range req {
		if _, err := tx.Exec("UPDATE `submissions` JOIN `users` ON `users`.`id` = `submissions`.`user_id` SET `score` = ? WHERE `users`.`code` = ? AND `class_id` = ?", score.Score, score.UserCode, classID); err != nil {
			c.Logger().Error(err)
			_ = tx.Rollback()
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	if err := tx.Commit(); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

type Submission struct {
	UserID   uuid.UUID `db:"user_id"`
	UserCode string    `db:"user_code"`
	FileName string    `db:"file_name"`
}

// DownloadSubmittedAssignments 提出済みの課題ファイルをzip形式で一括ダウンロード
func (h *handlers) DownloadSubmittedAssignments(c echo.Context) error {
	courseID := uuid.Parse(c.Param("courseID"))
	if courseID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid courseID.")
	}

	classID := uuid.Parse(c.Param("classID"))
	if classID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid classID.")
	}

	tx, err := h.DB.Beginx()
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var closedAt sql.NullTime
	if err := tx.Get(&closedAt, "SELECT `submission_closed_at` FROM `classes` WHERE `id` = ? FOR UPDATE", classID); err == sql.ErrNoRows {
		_ = tx.Rollback()
		return echo.NewHTTPError(http.StatusBadRequest, "No such class.")
	} else if err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return c.NoContent(http.StatusInternalServerError)
	}
	var submissions []Submission
	query := "SELECT `submissions`.`user_id`, `users`.`code` AS `user_code`" +
		" FROM `submissions`" +
		" JOIN `users` ON `users`.`id` = `submissions`.`user_id`" +
		" WHERE `class_id` = ? FOR SHARE"
	if err := tx.Select(&submissions, query, classID); err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return c.NoContent(http.StatusInternalServerError)
	}

	zipFilePath := AssignmentsDirectory + classID.String() + ".zip"
	if err := createSubmissionsZip(zipFilePath, classID, submissions); err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return c.NoContent(http.StatusInternalServerError)
	}

	if _, err := tx.Exec("UPDATE `classes` SET `submission_closed_at` = NOW() WHERE `id` = ?", classID); err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return c.NoContent(http.StatusInternalServerError)
	}

	if err := tx.Commit(); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.File(zipFilePath)
}

func createSubmissionsZip(zipFilePath string, classID uuid.UUID, submissions []Submission) error {
	tmpDir := AssignmentsDirectory + classID.String() + "/"
	if err := exec.Command("mkdir", tmpDir).Run(); err != nil {
		return err
	}

	// ファイル名を指定の形式に変更
	for _, submission := range submissions {
		if err := exec.Command(
			"cp",
			AssignmentsDirectory+classID.String()+"-"+submission.UserID.String(),
			tmpDir+submission.UserCode,
		).Run(); err != nil {
			return err
		}
	}

	var zipArgs []string
	zipArgs = append(zipArgs, "-j", zipFilePath)

	for _, submission := range submissions {
		zipArgs = append(zipArgs, tmpDir+submission.UserCode)
	}

	return exec.Command("zip", zipArgs...).Run()
}

type AddClassRequest struct {
	Part        uint8  `json:"part"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CreatedAt   int64  `json:"created_at"`
}

type AddClassResponse struct {
	ClassID        uuid.UUID `json:"class_id"`
	AnnouncementID uuid.UUID `json:"announcement_id"`
}

// AddClass 新規クラス(&課題)追加
func (h *handlers) AddClass(c echo.Context) error {
	courseID := c.Param("courseID")
	if courseID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid CourseID.")
	}

	var count int
	if err := h.DB.Get(&count, "SELECT COUNT(*) FROM `courses` WHERE `id` = ?", courseID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if count == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "No such course.")
	}

	var req AddClassRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid format.")
	}

	tx, err := h.DB.Beginx()
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	classID := uuid.NewRandom()
	createdAt := time.Unix(req.CreatedAt, 0)
	if _, err := tx.Exec("INSERT INTO `classes` (`id`, `course_id`, `part`, `title`, `description`, `created_at`) VALUES (?, ?, ?, ?, ?, ?)",
		classID, courseID, req.Part, req.Title, req.Description, createdAt); err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return c.NoContent(http.StatusInternalServerError)
	}

	announcementID := uuid.NewRandom()
	if _, err = tx.Exec("INSERT INTO `announcements` (`id`, `course_id`, `title`, `message`, `created_at`) VALUES (?, ?, ?, ?, ?)",
		announcementID, courseID, fmt.Sprintf("クラス追加: %s", req.Title), fmt.Sprintf("クラスが新しく追加されました: %s\n%s", req.Title, req.Description), createdAt); err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return c.NoContent(http.StatusInternalServerError)
	}

	var targets []User
	query := "SELECT `users`.*" +
		" FROM `users`" +
		" JOIN `registrations` ON `users`.`id` = `registrations`.`user_id`" +
		" WHERE `registrations`.`course_id` = ?"
	if err := tx.Select(&targets, query, courseID); err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return c.NoContent(http.StatusInternalServerError)
	}

	for _, user := range targets {
		if _, err := tx.Exec("INSERT INTO `unread_announcements` (`announcement_id`, `user_id`, `created_at`) VALUES (?, ?, ?)", announcementID, user.ID, createdAt); err != nil {
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
		ClassID:        classID,
		AnnouncementID: announcementID,
	}

	return c.JSON(http.StatusCreated, res)
}

type Announcement struct {
	ID         uuid.UUID `db:"id"`
	CourseID   uuid.UUID `db:"course_id"`
	CourseName string    `db:"course_name"`
	Title      string    `db:"title"`
	Unread     bool      `db:"unread"`
	CreatedAt  time.Time `db:"created_at"`
}

type GetAnnouncementsResponse struct {
	UnreadCount   int                    `json:"unread_count"`
	Announcements []AnnouncementResponse `json:"announcements"`
}

type AnnouncementResponse struct {
	ID         uuid.UUID `json:"id"`
	CourseID   uuid.UUID `json:"course_id"`
	CourseName string    `json:"course_name"`
	Title      string    `json:"title"`
	Unread     bool      `json:"unread"`
	CreatedAt  int64     `json:"created_at"`
}

// GetAnnouncementList お知らせ一覧取得
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

	var announcements []Announcement
	var args []interface{}
	query := "SELECT `announcements`.`id`, `courses`.`id` AS `course_id`, `courses`.`name` AS `course_name`, `announcements`.`title`, `unread_announcements`.`deleted_at` IS NULL AS `unread`, `announcements`.`created_at`" +
		" FROM `announcements`" +
		" JOIN `courses` ON `announcements`.`course_id` = `courses`.`id`" +
		" JOIN `registrations` ON `courses`.`id` = `registrations`.`course_id`" +
		" JOIN `unread_announcements` ON `announcements`.`id` = `unread_announcements`.`announcement_id`" +
		" WHERE 1=1"

	if courseID := c.QueryParam("course_id"); courseID != "" {
		query += " AND `announcements`.`course_id` = ?"
		args = append(args, courseID)
	}

	query += " AND `unread_announcements`.`user_id` = ?" +
		" AND `registrations`.`user_id` = ?" +
		" ORDER BY `announcements`.`created_at` DESC" +
		" LIMIT ? OFFSET ?"
	args = append(args, userID, userID)

	// MEMO: ページングの初期実装はページ番号形式
	var page int
	if c.QueryParam("page") == "" {
		page = 1
	} else {
		page, err = strconv.Atoi(c.QueryParam("page"))
		if err != nil || page <= 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid page.")
		}
	}
	limit := 20
	offset := limit * (page - 1)
	// limitより多く上限を設定し、実際にlimitより多くレコードが取得できた場合は次のページが存在する
	args = append(args, limit+1, offset)

	if err := h.DB.Select(&announcements, query, args...); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var unreadCount int
	if err := h.DB.Get(&unreadCount, "SELECT COUNT(*) FROM `unread_announcements` WHERE `user_id` = ? AND `deleted_at` IS NULL", userID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var links []string
	url, err := url.Parse(c.Request().URL.String())
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	url.Scheme = c.Scheme()
	url.Host = c.Request().Host
	q := url.Query()
	if page > 1 {
		q.Set("page", strconv.Itoa(page-1))
		url.RawQuery = q.Encode()
		links = append(links, fmt.Sprintf("<%v>; rel=\"prev\"", url))
	}
	if len(announcements) > limit {
		q.Set("page", strconv.Itoa(page+1))
		url.RawQuery = q.Encode()
		links = append(links, fmt.Sprintf("<%v>; rel=\"next\"", url))
	}
	if len(links) > 0 {
		c.Response().Header().Set("Link", strings.Join(links, ","))
	}

	if len(announcements) == limit+1 {
		announcements = announcements[:len(announcements)-1]
	}

	// 対象になっているお知らせが0件の時は空配列を返却
	announcementsRes := make([]AnnouncementResponse, 0, len(announcements))
	for _, announcement := range announcements {
		announcementsRes = append(announcementsRes, AnnouncementResponse{
			ID:         announcement.ID,
			CourseID:   announcement.CourseID,
			CourseName: announcement.CourseName,
			Title:      announcement.Title,
			Unread:     announcement.Unread,
			CreatedAt:  announcement.CreatedAt.Unix(),
		})
	}

	return c.JSON(http.StatusOK, GetAnnouncementsResponse{
		UnreadCount:   unreadCount,
		Announcements: announcementsRes,
	})
}

type AnnouncementDetail struct {
	ID         uuid.UUID `db:"id"`
	CourseID   uuid.UUID `db:"course_id"`
	CourseName string    `db:"course_name"`
	Title      string    `db:"title"`
	Message    string    `db:"message"`
	Unread     bool      `db:"unread"`
	CreatedAt  time.Time `db:"created_at"`
}

type GetAnnouncementDetailResponse struct {
	ID         uuid.UUID `json:"id"`
	CourseID   uuid.UUID `json:"course_id"`
	CourseName string    `json:"course_name"`
	Title      string    `json:"title"`
	Message    string    `json:"message"`
	Unread     bool      `json:"unread"`
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

	var announcement AnnouncementDetail
	query := "SELECT `announcements`.`id`, `courses`.`id` AS `course_id`, `courses`.`name` AS `course_name`, `announcements`.`title`, `announcements`.`message`, `unread_announcements`.`deleted_at` IS NULL AS `unread`, `announcements`.`created_at`" +
		" FROM `announcements`" +
		" JOIN `courses` ON `courses`.`id` = `announcements`.`course_id`" +
		" JOIN `unread_announcements` ON `unread_announcements`.`announcement_id` = `announcements`.`id`" +
		" WHERE `announcements`.`id` = ?" +
		" AND `unread_announcements`.`user_id` = ?"
	if err := h.DB.Get(&announcement, query, announcementID, userID); err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusNotFound, "No such announcement.")
	} else if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var registrationCount int
	if err := h.DB.Get(&registrationCount, "SELECT COUNT(*) FROM `registrations` WHERE `course_id` = ? AND `user_id` = ?", announcement.CourseID, userID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if registrationCount == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "No such announcement.")
	}

	if _, err := h.DB.Exec("UPDATE `unread_announcements` SET `deleted_at` = NOW() WHERE `announcement_id` = ? AND `user_id` = ?", announcementID, userID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, GetAnnouncementDetailResponse{
		ID:         announcement.ID,
		CourseID:   announcement.CourseID,
		CourseName: announcement.CourseName,
		Title:      announcement.Title,
		Message:    announcement.Message,
		Unread:     announcement.Unread,
		CreatedAt:  announcement.CreatedAt.Unix(),
	})
}

type AddAnnouncementRequest struct {
	CourseID  uuid.UUID `json:"course_id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	CreatedAt int64     `json:"created_at"`
}

type AddAnnouncementResponse struct {
	ID uuid.UUID `json:"id"`
}

// AddAnnouncement 新規お知らせ追加
func (h *handlers) AddAnnouncement(c echo.Context) error {
	var req AddAnnouncementRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid format.")
	}

	var count int
	if err := h.DB.Get(&count, "SELECT COUNT(*) FROM `courses` WHERE `id` = ?", req.CourseID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if count == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "No such course.")
	}

	tx, err := h.DB.Beginx()
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	announcementID := uuid.NewRandom()
	createdAt := time.Unix(req.CreatedAt, 0)
	if _, err := tx.Exec("INSERT INTO `announcements` (`id`, `course_id`, `title`, `message`, `created_at`) VALUES (?, ?, ?, ?, ?)",
		announcementID, req.CourseID, req.Title, req.Message, createdAt); err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return c.NoContent(http.StatusInternalServerError)
	}

	var targets []User
	query := "SELECT `users`.* FROM `users`" +
		" JOIN `registrations` ON `users`.`id` = `registrations`.`user_id`" +
		" WHERE `registrations`.`course_id` = ?"
	if err := tx.Select(&targets, query, req.CourseID); err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return c.NoContent(http.StatusInternalServerError)
	}

	for _, user := range targets {
		if _, err := tx.Exec("INSERT INTO `unread_announcements` (`announcement_id`, `user_id`, `created_at`) VALUES (?, ?, ?)", announcementID, user.ID, createdAt); err != nil {
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
