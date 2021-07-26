package util

import (
	"github.com/isucon/isucon11-final/benchmarker/model"
)

// CourseManager は登録に空きのあるコースをコマごとにQueue(FIFO)で管理しておりコースが履修できなくなったらDequeueする
type CourseManager struct {
	// EmptyCoursesQueues has queues of timeslot:30 = [DayOfWeek:5]*[Period:6]
	EmptyCoursesQueues [30]*courseQueue
}

func NewCourseManager() *CourseManager {
	m := &CourseManager{}
	for i := range m.EmptyCoursesQueues {
		m.EmptyCoursesQueues[i] = newCourseQueue(30)
	}
	return m
}

func (m *CourseManager) AddEmptyCourse(course *model.Course) {
	timeslot := course.DayOfWeek*5 + course.Period
	m.EmptyCoursesQueues[timeslot].Push(course)
}

func (m *CourseManager) AddStudentForRegistrableCourse(student *model.Student, timeslot int) *model.Course {
	// 1. 学生の空きコマ群を取得
	// 2. 空きコマの中で登録できるコースがあれば、そのコースに学生を登録
	// 3. 学生に登録成功したコースを登録 & スケジュール更新

	queue := m.EmptyCoursesQueues[timeslot]

	// queueのコースに対して登録を試みる
	queue.rmu.Lock()
	wishTakeCourse := queue.ReferrerHead()
	if wishTakeCourse == nil { // queueに登録可能コースがない
		queue.rmu.Unlock()
		return nil
	}
	isSuccess, isRegistrable := wishTakeCourse.RegisterStudentsIfRegistrable(student)
	if !isRegistrable { // 満員になった or もう登録できなくなってた
		queue.Pop()
	}
	queue.rmu.Unlock()

	if !isSuccess {
		// popしたので同じqueueで再度挑戦
		return m.AddStudentForRegistrableCourse(student, timeslot)
	}
	return wishTakeCourse
}
