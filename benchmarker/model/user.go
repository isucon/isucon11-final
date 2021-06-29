package model

import (
	"fmt"
	"sync"

	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/fails"
)

type UserData struct {
	Name        string
	Number      string
	RawPassword string
}

type Student struct {
	*UserData

	// ベンチの操作で変更されるデータ
	registeredCourseIDs    []string
	announcementsByID      map[string]*Announcement
	isReadByAnnouncementID map[string]bool

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
		UserData:               newUserData(name, number, rawPW),
		registeredCourseIDs:    []string{},
		announcementsByID:      map[string]*Announcement{},
		isReadByAnnouncementID: map[string]bool{},
		Agent:                  a,
		rmu:                    sync.RWMutex{},
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

	r := make([]string, len(s.registeredCourseIDs))
	copy(r, s.registeredCourseIDs)
	return r
}

func (s *Student) AddAnnouncement(id string, announcement *Announcement) error {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	if s.announcementsByID[id] != nil {
		return failure.NewError(fails.ErrApplication, fmt.Errorf("announcementID(%s) is duplicated", id))
	}
	s.announcementsByID[id] = announcement
	return nil
}
func (s *Student) AnnouncementsCount() int {
	s.rmu.RLock()
	defer s.rmu.RUnlock()

	return len(s.announcementsByID)
}
func (s *Student) AnnouncementByID(id string) *Announcement {
	s.rmu.RLock()
	defer s.rmu.RUnlock()

	return s.announcementsByID[id]
}
func (s *Student) AddReadAnnouncement(id string) {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	s.isReadByAnnouncementID[id] = true
}
func (s *Student) IsReadAnnouncement(id string) bool {
	s.rmu.RLock()
	defer s.rmu.RUnlock()

	return s.isReadByAnnouncementID[id]
}
