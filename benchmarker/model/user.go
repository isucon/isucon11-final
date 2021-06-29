package model

import (
	"sync"

	"github.com/isucon/isucandar/agent"
)

type UserData struct {
	Name        string
	Number      string
	RawPassword string
}

type Student struct {
	*UserData

	// ベンチの操作で変更されるデータ
	registeredCourseIDs     []string
	receivedAnnouncementIDs []string
	readAnnouncementIDs     []string
	receivedAssignmentIDs   []string
	submittedAssignmentIDs  []string
	firstSemesterGrades     map[string]uint32

	Agent *agent.Agent
	rmu   sync.RWMutex
}

type Faculty struct {
	*UserData

	Agent *agent.Agent
}

func newUserData(name, number, rawPW string) *UserData {
	return &UserData{
		Name:        name,
		Number:      number,
		RawPassword: rawPW,
	}
}

func NewFaculty(name, number, rawPW string) *Faculty {
	a, _ := agent.NewAgent()

	return &Faculty{
		UserData: newUserData(name, number, rawPW),
		Agent:    a,
	}
}

func NewStudent(name, number, rawPW string) *Student {
	a, _ := agent.NewAgent()

	return &Student{
		UserData:                newUserData(name, number, rawPW),
		registeredCourseIDs:     []string{},
		receivedAnnouncementIDs: []string{},
		readAnnouncementIDs:     []string{},
		receivedAssignmentIDs:   []string{},
		submittedAssignmentIDs:  []string{},
		firstSemesterGrades:     map[string]uint32{},
		Agent:                   a,
		rmu:                     sync.RWMutex{},
	}
}

func (s *Student) AddCourses(courses []string) {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	s.registeredCourseIDs = append(s.registeredCourseIDs, courses...)
}

func (s *Student) Courses() []string {
	s.rmu.RLock()
	defer s.rmu.RUnlock()

	return s.registeredCourseIDs
}

// 引数がvalidなものかは検証しない
func (s *Student) SetGradesUnchecked(courseID string, grade uint32) {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	s.firstSemesterGrades[courseID] = grade
}

func (s *Student) FirseSemesterGrade() map[string]uint32 {
	s.rmu.RLock()
	defer s.rmu.RUnlock()

	return s.firstSemesterGrades
}
