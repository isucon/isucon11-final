package scenario

import "time"

// Load
const (
	// initialStudentsCount 初期学生数
	initialStudentsCount = 50
	// initialCourseCount 初期科目数
	initialCourseCount = 20
	// registerCourseLimitPerStudent は学生あたりの同時履修可能科目数の制限
	registerCourseLimitPerStudent = 20
	// StudentCapacityPerCourse は科目あたりの履修定員 -> same const exist in model/course.go
	//StudentCapacityPerCourse = 50
	// searchCountPerRegistration は履修登録前に実行するシラバス取得の回数
	searchCountPerRegistration = 3
	// ClassCountPerCourse は科目あたりのクラス数 -> same const exist in model/course.go
	ClassCountPerCourse = 5
	// invalidSubmitFrequency は誤ったFileTypeのファイルを提出する確率
	invalidSubmitFrequency = 0.1
	// waitReadClassAnnouncementTimeout は学生がクラス課題のお知らせを確認するのを待つ最大時間
	waitReadClassAnnouncementTimeout = 5 * time.Second
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
	validateAnnouncementsRate = 1.0
)
