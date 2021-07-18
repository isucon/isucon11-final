package model

import (
	"sync"

	"github.com/isucon/isucandar/agent"
)

type UserAccount struct {
	ID          string
	RawPassword string
}

type Student struct {
	*UserAccount
	Agent *agent.Agent

	registeredCourses     []*Course
	hasUnreadAnnouncement bool
	announcements         AnnouncementDeque
	unreadAnnouncements   AnnouncementDeque

	rmu sync.RWMutex
}

func NewStudent(id, rawPW string) *Student {
	a, _ := agent.NewAgent()
	return &Student{
		UserAccount: &UserAccount{
			ID:          id,
			RawPassword: rawPW,
		},
		Agent:             a,
		registeredCourses: make([]*Course, 0),
		rmu:               sync.RWMutex{},
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

func (s *Student) HasUnreadAnnouncement() bool {
	s.rmu.RLock()
	defer s.rmu.RUnlock()

	return s.hasUnreadAnnouncement
}

func (s *Student) UnreadAnnouncements() []*Announcement {
	s.rmu.RLock()
	defer s.rmu.RUnlock()

	// dequeでもmutex取ってるけどいらないかも知れない
	return s.unreadAnnouncements.Items()
}

func (s *Student) PopOldestUnreadAnnouncements() *Announcement {
	s.rmu.RLock()
	defer s.rmu.RUnlock()

	// TODO: ここでpubsubとかで課題提出workerにおくってもいいかもしれない
	// dequeでもmutex取ってるけどいらないかも知れない
	return s.unreadAnnouncements.PopFront()
}

func (s *Student) PushOldestUnreadAnnouncements(a *Announcement) {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	s.unreadAnnouncements.PushFront(a)
}

func (s *Student) PushLatestUnreadAnnouncements(a *Announcement) {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	s.unreadAnnouncements.PushBack(a)
}

func (s *Student) RemoveUnreadAnnouncement(id string) {
	s.rmu.Lock()

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
