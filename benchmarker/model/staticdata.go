package model

// StaticFacultyData はアプリケーションが初期データとして持っているFaculty用Userデータ
var StaticFacultyData *Faculty

// StaticStudentsData はアプリケーションが初期データとして持っているStudent用Userデータ
var StaticStudentsData []*Student

// StaticCoursesData はアプリケーションが初期データとして持っているCourseデータ
var StaticCoursesData []*Course

func init() {
	StaticFacultyData = NewFaculty("椅子昆", "01234567-89ab-cdef-0001-000000000004", "password")

	StaticStudentsData = []*Student{
		NewStudent("佐藤太郎", "01234567-89ab-cdef-0001-000000000001", "password"),
	}

	StaticCoursesData = []*Course{
		NewCourse("01234567-89ab-cdef-0002-000000000001", "微分積分基礎", 100, 0, 1, []string{"微分積分"}, "TBD"),
	}
}
