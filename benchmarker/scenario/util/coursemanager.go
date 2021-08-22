package util

import (
	"github.com/isucon/isucon11-final/benchmarker/model"
)

// CourseManager は登録に空きのあるコースをコマごとにQueue(FIFO)で管理しておりコースが履修できなくなったらDequeueする
type CourseManager struct {
	// EmptyCoursesQueues has queues by timeslots [DayOfWeek:7][Period:6]
	EmptyCoursesQueues [7][6]*courseQueue
}

func NewCourseManager() *CourseManager {
	m := &CourseManager{}
	for dayOfWeek, queueByPeriod := range m.EmptyCoursesQueues {
		for period := range queueByPeriod {
			m.EmptyCoursesQueues[dayOfWeek][period] = newCourseQueue(30)
		}
	}
	return m
}

func (m *CourseManager) AddEmptyCourse(course *model.Course) {
	m.EmptyCoursesQueues[course.DayOfWeek][course.Period].Push(course)
}

// AddStudentForRegistrableCourse timeslotを受け取って、適切なコースに学生を登録する
func (m *CourseManager) AddStudentForRegistrableCourse(student *model.Student, dayOfWeek, period int) *model.Course {
	// 1. 渡された学生と希望Timeslotで登録できるコースがあれば、そのコースに学生を登録
	// 2. 登録できたコースを返却

	queue := m.EmptyCoursesQueues[dayOfWeek][period]

	// queueのコースに対して登録を試みる
	wishTakeCourse := queue.ReferrerHead()
	if wishTakeCourse == nil { // queueに登録可能コースがない
		return nil
	}
	isSuccess := wishTakeCourse.TempRegisterIfRegistrable()
	if !isSuccess { // 満員になった or もう登録できなくなってた
		queue.Pop()
		// popしたので同じqueueで再度挑戦
		return m.AddStudentForRegistrableCourse(student, dayOfWeek, period)
	}

	return wishTakeCourse
}
