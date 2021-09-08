package generate

import (
	_ "embed"

	"github.com/isucon/isucon11-final/benchmarker/model"
)

var (
	initCourses []*model.Course
)

func init() {
	// 動作確認用のアカウント
	// ベンチではこのアカウントを操作することはないためUserAccountのみを管理する
	testTeacher := &model.Teacher{
		UserAccount: &model.UserAccount{
			Code:        "T99999",
			Name:        "isucon-teacher",
			RawPassword: "isucon",
		},
	}

	// course(in-progress/registration)の作成
	// grade計算はclosedのコースのみ含まれないので学生を保持する必要はない
	testCourse1 := model.NewCourse(
		&model.CourseParam{
			Code:        "X00001",
			Type:        "major-subjects",
			Name:        "ISUCON演習第一",
			Description: "この科目ではISUCONの過去問を通してサーバのチューニングアップを学びます。\n課題は講義中に出題するクイズへの回答を提出してください。\n\n本講義の成績は課題の提出状況により判断します。",
			Credit:      1,
			Teacher:     "isucon-teacher",
			Period:      0,
			DayOfWeek:   0,
			Keywords:    "ISUCON SpeedUP",
		}, "b4f7ab13-8629-420a-a173-f166b0162b56", testTeacher, 50)
	testCourse1.SetStatusToInProgress()

	testCourse2 := model.NewCourse(
		&model.CourseParam{
			Code:        "X00002",
			Type:        "major-subjects",
			Name:        "ISUCON演習第二",
			Description: "この科目ではISUCONの過去問を通してサーバのチューニングアップを学びます。\n課題は講義中に出題するクイズへの回答を提出してください。\n\n本講義の成績は課題の提出状況により判断します。",
			Credit:      1,
			Teacher:     "isucon-teacher",
			Period:      0,
			DayOfWeek:   1,
			Keywords:    "ISUCON SpeedUP",
		}, "c22a43db-e9d9-4077-9bc8-99479ef86b41", testTeacher, 50)
	testCourse2.SetStatusToInProgress()

	testCourse3 := model.NewCourse(
		&model.CourseParam{
			Code:        "X00003",
			Type:        "major-subjects",
			Name:        "ISUCON演習第三",
			Description: "この科目ではISUCONの過去問を通してサーバのチューニングアップを学びます。\n課題は講義中に出題するクイズへの回答を提出してください。\n\n本講義の成績は課題の提出状況により判断します。",
			Credit:      1,
			Teacher:     "isucon-teacher",
			Period:      0,
			DayOfWeek:   2,
			Keywords:    "ISUCON SpeedUP",
		}, "ae53eb49-0258-463f-be70-1b295d7df740", testTeacher, 50)

	initCourses = []*model.Course{testCourse1, testCourse2, testCourse3}
}

// InitialCourses は初期に追加されているコースを返す
// Loadなどでは操作されることは想定されていないので検証のみで扱う
func InitialCourses() []*model.Course {
	// 他の初期コースを追加する場合はここで追加する
	return initCourses
}
