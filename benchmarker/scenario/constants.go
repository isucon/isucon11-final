package scenario

import "time"

// Load
const (
	// initialStudentsCount 初期学生数
	initialStudentsCount = 50
	// initialCourseCount 初期科目数 30以上である必要がある
	initialCourseCount = 30
	// registerCourseLimitPerStudent は学生あたりの同時履修可能科目数の制限
	registerCourseLimitPerStudent = 20
	// StudentCapacityPerCourse は科目あたりの履修定員
	StudentCapacityPerCourse = 50
	// searchCountPerRegistration は履修登録前に実行するシラバス取得の回数
	searchCountPerRegistration = 3
	// ClassCountPerCourse は科目あたりのクラス数 -> same const exist in model/course.go
	ClassCountPerCourse = 5
	// minimumCheckAnnouncementInterval は課題登録で発火されるお知らせ一覧取得の最小間隔
	minimumCheckAnnouncementInterval = 100 * time.Millisecond
	// AnnouncePagingStudentInterval はお知らせページングシナリオを開始する人数間隔
	AnnouncePagingStudentInterval = 5
	// announcePagingInterval はページングを繰り返すシナリオのページング間隔
	announcePagingInterval = 100 * time.Millisecond
	// waitCourseFullTimeout は最初の履修成功時刻から科目が満員に達するまでの待ち時間
	waitCourseFullTimeout = 2 * time.Second
	// waitReadClassAnnouncementTimeout は学生がクラス課題のお知らせを確認するのを待つ最大時間
	waitReadClassAnnouncementTimeout = 5 * time.Second
	// waitGradeTimeout は成績確認がタイムアウトした際に再度確認しに行くまでの待ち
	waitGradeTimeout = 10 * time.Second
	// loadRequestTime はLoadシナリオ内でリクエストを送り続ける時間(Load自体のTimeoutより早めに終わらせる)
	loadRequestTime = 60 * time.Second
)

// Verify
// TODO: 決め打ちではなく外から指定できるようにする
const (
	searchCourseVerifyRate = 0.2
	assignmentsVerifyRate  = 0.2
)

// Validation
const (
	validateAnnouncementsRate        = 1.0
	validateGPAErrorTolerance        = 0.01
	validateTotalScoreErrorTolerance = 0.01
)
