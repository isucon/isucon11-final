package model

import (
	"sync"

	"github.com/isucon/isucandar/agent"
)

type User struct {
	Name        string
	Number      string
	RawPassword string

	Agent *agent.Agent
}

type Student struct {
	*User

	// ベンチの操作で変更されるデータ
	registeredCourseIDs     []string
	receivedAnnouncementIDs []string
	readAnnouncementIDs     []string
	receivedAssignmentIDs   []string
	submittedAssignmentIDs  []string
	firstSemesterGrades     map[string]string

	rmu sync.RWMutex
}

func NewUser(name, number, rawPW string) *User {
	a, _ := agent.NewAgent()

	return &User{
		Name:        name,
		Number:      number,
		RawPassword: rawPW,
		Agent:       a,
	}
}

func NewStudent(name, number, rawPW string) *Student {
	return &Student{
		User:                    NewUser(name, number, rawPW),
		registeredCourseIDs:     []string{},
		receivedAnnouncementIDs: []string{},
		readAnnouncementIDs:     []string{},
		receivedAssignmentIDs:   []string{},
		submittedAssignmentIDs:  []string{},
		firstSemesterGrades:     map[string]string{},
		rmu:                     sync.RWMutex{},
	}
}
