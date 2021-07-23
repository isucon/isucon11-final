package model

import (
	"sync"
	"time"

	"github.com/isucon/isucandar/agent"
)

type UserAccount struct {
	ID          string
	Code        string
	RawPassword string
}

type Student struct {
	*UserAccount
	Agent *agent.Agent

	registeredCourses     []*Course
	announcements         []*AnnouncementStatus
	announcementIndexByID map[string]int
	submissions           []*Submission

	rmu sync.RWMutex
}
type AnnouncementStatus struct {
	Announcement *Announcement
	Unread       bool
}

func NewStudent(id, rawPW string) *Student {
	a, _ := agent.NewAgent()
	return &Student{
		UserAccount: &UserAccount{
			ID:          id,
			RawPassword: rawPW,
		},
		Agent:                 a,
		registeredCourses:     make([]*Course, 0),
		announcements:         make([]*AnnouncementStatus, 100),
		announcementIndexByID: make(map[string]int, 100),

		rmu: sync.RWMutex{},
	}
}

func (s *Student) RegisteredCoursesCount() int {
	s.rmu.RLock()
	defer s.rmu.RUnlock()

	return len(s.registeredCourses)
}

func (s *Student) AddCourse(course *Course) {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	s.registeredCourses = append(s.registeredCourses, course)
}

func (s *Student) AddSubmission(sub *Submission) {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	s.submissions = append(s.submissions, sub)
}

func (s *Student) AddAnnouncement(announcement *Announcement) {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	s.announcements = append(s.announcements, &AnnouncementStatus{announcement, false})
	s.announcementIndexByID[announcement.ID] = len(s.announcements) - 1
}

func (s *Student) ReadAnnouncement(id string) {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	s.announcements[s.announcementIndexByID[id]].Unread = true
}
func (s *Student) isUnreadAnnouncement(id string) bool {
	s.rmu.RLock()
	defer s.rmu.RUnlock()

	return s.announcements[s.announcementIndexByID[id]].Unread
}
func (s *Student) WaitReadAnnouncement(id string) <-chan struct{} {
	ch := make(chan struct{})

	if s.isUnreadAnnouncement(id) {
		go func() {
			for s.isUnreadAnnouncement(id) {
				<-time.After(1 * time.Millisecond)
			}
			close(ch)
		}()
	} else {
		close(ch)
	}
	return ch
}

type Faculty struct {
	*UserAccount
	Agent *agent.Agent
}

func NewFaculty(id, rawPW string) *Faculty {
	a, _ := agent.NewAgent()
	return &Faculty{
		UserAccount: &UserAccount{
			ID:          id,
			RawPassword: rawPW,
		},
		Agent: a,
	}
}
