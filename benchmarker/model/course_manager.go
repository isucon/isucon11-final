package model

import (
	"sync"
)

// CourseManager は科目の履修登録を行う。
// 空き科目の管理 → 履修登録までが責任範囲。
type CourseManager struct {
	courses           map[string]*Course
	waitingCourseList *courseList // 空きのある科目を優先順に並べたもの
	capacityCounter   *CapacityCounter
	rmu               sync.RWMutex
}

func NewCourseManager(cc *CapacityCounter) *CourseManager {
	m := &CourseManager{
		courses:           map[string]*Course{},
		waitingCourseList: newCourseList(1000),
		capacityCounter:   cc,
	}
	return m
}

func (m *CourseManager) AddNewCourse(course *Course) {
	m.rmu.Lock()
	defer m.rmu.Unlock()

	m.courses[course.ID] = course
	m.waitingCourseList.Lock()
	m.waitingCourseList.Add(course)
	m.waitingCourseList.Unlock()
}

func (m *CourseManager) GetCourseByID(id string) (*Course, bool) {
	m.rmu.RLock()
	defer m.rmu.RUnlock()

	course, exists := m.courses[id]
	return course, exists
}

func (m *CourseManager) GetCourseCount() int {
	m.rmu.RLock()
	defer m.rmu.RUnlock()

	return len(m.courses)
}

// ReserveCoursesForStudent は学生を受け取って、キュー内の科目に仮登録を行う
func (m *CourseManager) ReserveCoursesForStudent(student *Student, remainingRegistrationCapacity int) []*Course {
	student.LockSchedule()
	defer student.UnlockSchedule()

	temporaryReservedCourses := make([]*Course, 0, remainingRegistrationCapacity)

	// 重い時はqueueをシャーディングする
	m.waitingCourseList.RLock()
	for i := 0; i < m.waitingCourseList.Len() && len(temporaryReservedCourses) < remainingRegistrationCapacity; i++ {
		target := m.waitingCourseList.Seek(i)
		if !student.IsEmptyTimeSlots(target.DayOfWeek, target.Period) {
			continue
		}
		result := target.ReserveIfAvailable()
		switch result {
		case Succeeded:
			temporaryReservedCourses = append(temporaryReservedCourses, target)
			student.FillTimeslot(target)
			m.capacityCounter.Dec(target.DayOfWeek, target.Period)
		case NotAvailable:
			continue
		}
	}
	m.waitingCourseList.RUnlock()

	return temporaryReservedCourses
}

func (m *CourseManager) RemoveRegistrationClosedCourse(c *Course) {
	m.waitingCourseList.Lock()
	defer m.waitingCourseList.Unlock()
	for i, v := range m.waitingCourseList.items {
		if v == c {
			m.waitingCourseList.Remove(i)
			break
		}
	}
}

func (m *CourseManager) ExposeCoursesForValidation() map[string]*Course {
	return m.courses
}
