package model

import "github.com/isucon/isucandar/agent"

// StaticFacultyData はアプリケーションが初期データとして持っているFaculty用Userデータ
var StaticFacultyData *User

// StaticStudentsData はアプリケーションが初期データとして持っているStudent用Userデータ
var StaticStudentsData []*Student

// StaticCoursesData はアプリケーションが初期データとして持っているCourseデータ
var StaticCoursesData []*Course

func init() {
	a, _ := agent.NewAgent()
	StaticFacultyData = &User{
		Name:        "APIForFaculty",
		Number:      "99999999",
		RawPassword: "piyopiyo",
		Agent:       a,
	}

	StaticStudentsData = []*Student{
		NewStudent("服部 夢二", "21020162", "hogehoge"),
	}

	StaticCoursesData = []*Course{
		NewCourse("00000111", "確率統計学", 30, 1, 1, []string{"確率"}, "120EA8A25E5D487BF68B5F7096440019"),
	}
}
