package model

import "sync"

// CourseManager は科目の履修管理を行う
// 科目の追加 → 履修登録 → 科目の開始までが責任範囲
// 科目の終了は科目用のgoroutine内で行われ、新規科目の追加が呼ばれる
type CourseManager struct {
	courses map[string]*Course
	queue   *courseQueue // 優先的に履修させる科目を各timeslotごとに保持
	rmu     sync.RWMutex
}

const queueLength = 6 * 7 * 5 // Period * DayOfWeek * 5 (この値を調整する)

func NewCourseManager() *CourseManager {
	m := &CourseManager{
		courses: map[string]*Course{},
		queue:   newCourseQueue(queueLength),
	}
	return m
}

func (m *CourseManager) AddNewCourse(course *Course) {
	m.rmu.Lock()
	defer m.rmu.Unlock()

	m.courses[course.ID] = course
	m.queue.Lock()
	if m.queue.Len() < queueLength {
		m.queue.Add(course)
	}
	m.queue.Unlock()
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
	m.queue.Lock()
	for i := 0; i < m.queue.Len() && len(temporaryReservedCourses) < remainingRegistrationCapacity; i++ {
		target := m.queue.Seek(i) // i
		if !student.IsEmptyTimeSlots(target.DayOfWeek, target.Period) {
			continue
		}
		result := target.ReserveIfRegistrable()
		switch result {
		case Succeeded:
			temporaryReservedCourses = append(temporaryReservedCourses, target)
			student.FillTimeslot(target)
		case TemporaryFull:
			continue
		case Closed:
			// 対象が履修登録を締め切っていた場合、キューから除き、現在のインデックスから再開する
			m.queue.Remove(i)
			picked := m.getRandomCourse(target)
			if picked == nil {
				panic("available course not found. improvement required.")
			}
			m.queue.Add(picked)
			i--
		}
	}
	m.queue.Unlock()

	return temporaryReservedCourses
}

// 遅い時はもう少し考える
func (m *CourseManager) getRandomCourse(old *Course) (new *Course) {
	for _, course := range m.courses {
		if !course.isRegistrationClosed && course.DayOfWeek == old.DayOfWeek && course.Period == old.Period {
			return course
		}
	}
	return nil
}

func (m *CourseManager) ExposeCoursesForValidation() map[string]*Course {
	return m.courses
}
