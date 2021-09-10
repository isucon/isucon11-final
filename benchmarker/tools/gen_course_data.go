//go:generate go run .
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/isucon/isucon11-final/benchmarker/api"
	"github.com/isucon/isucon11-final/benchmarker/generate"
	"github.com/isucon/isucon11-final/benchmarker/model"
)

const (
	courseCount  = 30
	teacherCount = 10
)

func main() {
	teachersData, err := generate.LoadTeachersData()
	if err != nil {
		log.Fatal(err)
	}
	teachers := make([]*model.Teacher, teacherCount)
	for i, account := range teachersData[:teacherCount] {
		teachers[i] = model.NewTeacher(account, nil)
	}

	courses := make([]*model.Course, 0)

	// 初期科目の生成
	for i := 0; i < courseCount; i++ {
		timeslot := i % 30
		dayOfWeek := timeslot / 6
		period := timeslot % 6
		teacher := teachers[i%teacherCount]
		param := generate.CourseParam(dayOfWeek, period, teacher)
		param.Code = fmt.Sprintf("A%04d", i+1)
		course := model.NewCourse(param, generate.GenULID(), teacher, 50)
		courses = append(courses, course)
		course.SetStatusToClosed()
	}

	// 動作確認用科目の教員アカウント
	// ベンチではこのアカウントを操作することはないためUserAccountのみを管理する
	testTeacher := &model.Teacher{
		UserAccount: &model.UserAccount{
			ID:          "01FF4RXEKS0DG2EG20CKDWS7CC",
			Code:        "T99999",
			Name:        "isucon-teacher",
			RawPassword: "isucon",
		},
	}

	// 動作確認用科目(in-progress/registration)の作成
	// grade計算はclosedの科目しか含まれないのでこれらの科目を履修している学生をベンチで保持する必要はない
	testCourse1 := model.NewCourse(
		&model.CourseParam{
			Code:        "X0001",
			Type:        "major-subjects",
			Name:        "ISUCON演習第一",
			Description: "この科目ではISUCONの過去問を通してサーバのチューニングアップを学びます。課題は講義中に出題するクイズへの回答を提出してください。本講義の成績は課題の提出状況により判断します。",
			Credit:      1,
			Teacher:     "isucon-teacher",
			Period:      0,
			DayOfWeek:   0,
			Keywords:    "ISUCON SpeedUP",
		}, "01FF4RXEKS0DG2EG20CWPQ60M3", testTeacher, 50)
	testCourse1.SetStatusToInProgress()

	testCourse2 := model.NewCourse(
		&model.CourseParam{
			Code:        "X0002",
			Type:        "major-subjects",
			Name:        "ISUCON演習第二",
			Description: "この科目ではISUCONの過去問を通してサーバのチューニングアップを学びます。課題は講義中に出題するクイズへの回答を提出してください。本講義の成績は課題の提出状況により判断します。",
			Credit:      1,
			Teacher:     "isucon-teacher",
			Period:      0,
			DayOfWeek:   1,
			Keywords:    "ISUCON SpeedUP",
		}, "01FF4RXEKS0DG2EG20CYAYCCGM", testTeacher, 50)
	testCourse2.SetStatusToInProgress()

	testCourse3 := model.NewCourse(
		&model.CourseParam{
			Code:        "X0003",
			Type:        "major-subjects",
			Name:        "ISUCON演習第三",
			Description: "この科目ではISUCONの過去問を通してサーバのチューニングアップを学びます。課題は講義中に出題するクイズへの回答を提出してください。本講義の成績は課題の提出状況により判断します。",
			Credit:      1,
			Teacher:     "isucon-teacher",
			Period:      0,
			DayOfWeek:   2,
			Keywords:    "ISUCON SpeedUP",
		}, "01FF4RXEKS0DG2EG20D23EQZRY", testTeacher, 50)

	courses = append(courses, testCourse1, testCourse2, testCourse3)

	saveTsv(courses, "course.tsv")
	saveSql(courses[:courseCount], "course.sql")
}

func saveTsv(courses []*model.Course, fileName string) {
	tsvFile, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer tsvFile.Close()

	for _, course := range courses {
		_, err = tsvFile.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%d\t%d\t%d\t%s\t%s\t%s\n",
			course.ID,
			course.Code,
			course.Type,
			course.Name,
			course.Description,
			course.Credit,
			course.Period,
			course.DayOfWeek,
			course.Teacher().ID,
			course.Keywords,
			course.Status()))
		if err != nil {
			log.Fatal(err)
		}
	}
}

func saveSql(courses []*model.Course, fileName string) {
	sqlFile, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer sqlFile.Close()

	sqlValues := make([]string, 0, len(courses))
	for _, course := range courses {
		sqlValues = append(sqlValues, fmt.Sprintf("('%s', '%s', '%s', '%s', '%s', %d, %d, '%s', '%s', '%s', '%s')",
			course.ID,
			course.Code,
			course.Type,
			course.Name,
			course.Description,
			course.Credit,
			course.Period+1,
			api.DayOfWeekTable[course.DayOfWeek],
			course.Teacher().ID,
			course.Keywords,
			course.Status()))
	}

	_, err = sqlFile.WriteString("INSERT INTO `courses` (`id`, `code`, `type`, `name`, `description`, `credit`, `period`, `day_of_week`, `teacher_id`, `keywords`, `status`) VALUES\n")
	if err != nil {
		log.Fatal(err)
	}

	_, err = sqlFile.WriteString(strings.Join(sqlValues, ",\n") + ";\n")
	if err != nil {
		log.Fatal(err)
	}
}
