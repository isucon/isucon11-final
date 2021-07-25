package model

import (
	"math/rand"
	"net/url"
	"sync"
	"time"

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
	RegisterCourseLimit int
	Agent               *agent.Agent

	registeredCourses     []*Course
	announcements         []*AnnouncementStatus
	announcementIndexByID map[string]int
	submissions           []*Submission
	registeredSchedule    [30]bool // 空きコマ管理[DayOfWeek:5]*[Period:6]
	registeringCount      int

	rmu sync.RWMutex
}
type AnnouncementStatus struct {
	Announcement *Announcement
	Unread       bool
}

func NewStudent(userData *UserAccount, baseURL *url.URL) *Student {
	a, _ := agent.NewAgent()
	a.Name = useragent.UserAgent()
	a.BaseURL = baseURL

	return &Student{
		UserAccount:           userData,
		RegisterCourseLimit:   20,
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

func (s *Student) UnlockSchedule(dayOfWeek, period int) {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	s.registeredSchedule[dayOfWeek*5+period] = false
	s.registeringCount--
}
func (s *Student) LockRandomEmptySchedule() (dayOfWeek, period int) {
	randTable := [30]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29}
	for i := len(randTable) - 1; i >= 0; i-- {
		j := rand.Intn(i + 1)
		randTable[i], randTable[j] = randTable[j], randTable[i]
	}

	s.rmu.Lock()
	defer s.rmu.Unlock()

	if s.registeringCount < s.RegisterCourseLimit {
		return 0, 0
	}
	for i := 0; i < s.RegisterCourseLimit; i++ {
		if !s.registeredSchedule[randTable[i]] {
			dayOfWeek = randTable[i] / 5
			period = randTable[i] % 5

			s.registeringCount++
			s.registeredSchedule[randTable[i]] = true
			return
		}
	}
	return 0, 0
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
