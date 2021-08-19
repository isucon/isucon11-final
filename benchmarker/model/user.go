package model

import (
	"net/url"
	"sync"

	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/random/useragent"
)

type UserAccount struct {
	Code        string
	Name        string
	RawPassword string
}

type Student struct {
	*UserAccount
	RegisteringCourseLimit int
	Agent                  *agent.Agent

	registeredCourses     []*Course
	announcements         []*AnnouncementStatus
	announcementIndexByID map[string]int
	announcementCond      *sync.Cond
	submissions           []*Submission
	rmu                   sync.RWMutex

	registeredSchedule [7][6]*Course // 空きコマ管理[DayOfWeek:7][Period:6]
	registeringCount   int
	scheduleMutex      sync.RWMutex
}
type AnnouncementStatus struct {
	Announcement *Announcement
	Unread       bool
}

func NewStudent(userData *UserAccount, baseURL *url.URL, regLimit int) *Student {
	a, _ := agent.NewAgent()
	a.Name = useragent.UserAgent()
	a.BaseURL = baseURL

	s := &Student{
		UserAccount:            userData,
		RegisteringCourseLimit: regLimit,
		Agent:                  a,

		registeredCourses:     make([]*Course, 0, 20),
		announcements:         make([]*AnnouncementStatus, 0, 100),
		announcementIndexByID: make(map[string]int, 100),
		rmu:                   sync.RWMutex{},

		registeredSchedule: [7][6]*Course{},
		scheduleMutex:      sync.RWMutex{},
	}
	s.announcementCond = sync.NewCond(&s.rmu)
	return s
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

	s.announcements = append(s.announcements, &AnnouncementStatus{announcement, true})
	s.announcementIndexByID[announcement.ID] = len(s.announcements) - 1
}

func (s *Student) GetAnnouncement(id string) *AnnouncementStatus {
	s.rmu.RLock()
	defer s.rmu.RUnlock()

	index, exists := s.announcementIndexByID[id]
	if !exists {
		return nil
	}
	return s.announcements[index]
}

func (s *Student) ReadAnnouncement(id string) {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	s.announcements[s.announcementIndexByID[id]].Unread = false
	s.announcementCond.Broadcast()
}

func (s *Student) isUnreadAnnouncement(id string) bool {
	return s.announcements[s.announcementIndexByID[id]].Unread
}

func (s *Student) WaitReadAnnouncement(id string) <-chan struct{} {
	ch := make(chan struct{})

	s.rmu.RLock()
	if s.isUnreadAnnouncement(id) {
		go func() {
			s.announcementCond.L.Lock()
			for s.isUnreadAnnouncement(id) {
				s.announcementCond.Wait()
			}
			s.announcementCond.L.Unlock()
			close(ch)
		}()
	} else {
		close(ch)
	}
	s.rmu.RUnlock()
	return ch
}

func (s *Student) RegisteringCount() int {
	s.scheduleMutex.RLock()
	defer s.scheduleMutex.RUnlock()

	return s.registeringCount
}

func (s *Student) ReleaseTimeslot(dayOfWeek, period int) {
	s.scheduleMutex.Lock()
	defer s.scheduleMutex.Unlock()

	s.registeredSchedule[dayOfWeek][period] = nil
	s.registeringCount--
}

// ScheduleMutex はstudent内で完結しない同期処理を行う際に利用
func (s *Student) ScheduleMutex() *sync.RWMutex {
	return &s.scheduleMutex
}

// IsEmptyTimeSlots でコマを参照する場合は別途scheduleMutexで(R)Lockすること
func (s *Student) IsEmptyTimeSlots(dayOfWeek, period int) bool {
	return s.registeredSchedule[dayOfWeek][period] == nil
}

// FillTimeslot で登録処理を行う場合は別途scheduleMutexでLockすること
func (s *Student) FillTimeslot(course *Course) {
	s.registeredSchedule[course.DayOfWeek][course.Period] = course
	s.registeringCount++
}

func (s *Student) RegisteredSchedule() [7][6]*Course {
	s.scheduleMutex.RLock()
	defer s.scheduleMutex.RUnlock()

	return s.registeredSchedule
}

type Faculty struct {
	*UserAccount
	Agent *agent.Agent
}

const facultyUserAgent = "isucholar-agent-faculty/1.0.0"

func NewFaculty(userData *UserAccount, baseURL *url.URL) *Faculty {
	a, _ := agent.NewAgent()
	a.BaseURL = baseURL
	a.Name = facultyUserAgent
	return &Faculty{
		UserAccount: userData,
		Agent:       a,
	}
}
