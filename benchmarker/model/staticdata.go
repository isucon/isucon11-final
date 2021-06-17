package model

// StaticFacultyData はアプリケーションが初期データとして持っているFaculty用Userデータ
var StaticFacultyData *Faculty

// StaticStudentsData はアプリケーションが初期データとして持っているStudent用Userデータ
var StaticStudentsData []*Student

// StaticCoursesData はアプリケーションが初期データとして持っているCourseデータ
var StaticCoursesData []*Course

func init() {
	StaticFacultyData = NewFaculty("APIForFaculty", "99999999", "piyopiyo")

	StaticStudentsData = []*Student{
		NewStudent("服部 夢二", "21020162", "hogehoge"),
	}

	StaticCoursesData = []*Course{
		NewCourse("00000111", "確率統計学", 30, 1, 1, []string{"確率"}, "120EA8A25E5D487BF68B5F7096440019"),
	}
}
