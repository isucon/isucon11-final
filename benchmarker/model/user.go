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
	RegisteringCourseLimit int
	Agent                  *agent.Agent

	registeredCourses     []*Course
	announcements         []*AnnouncementStatus
	announcementIndexByID map[string]int
	submissions           []*Submission
	rmu                   sync.RWMutex

	registeredSchedule [7][6]bool // 空きコマ管理[DayOfWeek:7][Period:6]
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

	return &Student{
		UserAccount:            userData,
		RegisteringCourseLimit: regLimit,
		Agent:                  a,

		registeredCourses:     make([]*Course, 0),
		announcements:         make([]*AnnouncementStatus, 100),
		announcementIndexByID: make(map[string]int, 100),
		rmu:                   sync.RWMutex{},

		registeredSchedule: [7][6]bool{},
		scheduleMutex:      sync.RWMutex{},
	}
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

func (s *Student) RegisteringCount() int {
	s.scheduleMutex.RLock()
	defer s.scheduleMutex.RUnlock()

	return s.registeringCount
}

func (s *Student) ReleaseTimeslot(dayOfWeek, period int) {
	s.scheduleMutex.Lock()
	defer s.scheduleMutex.Unlock()

	s.registeredSchedule[dayOfWeek][period] = false
	s.registeringCount--
}

// ScheduleMutex はstudent内で完結しない同期処理を行う際に利用
func (s *Student) ScheduleMutex() *sync.RWMutex {
	return &s.scheduleMutex
}

type Timeslot struct {
	DayOfWeek int
	Period    int
}

// RandomEmptyTimeSlots を参照して登録処理を行う場合は別途scheduleMutexでLockすること
func (s *Student) RandomEmptyTimeSlots() []Timeslot {
	if s.registeringCount >= s.RegisteringCourseLimit {
		return nil
	}

	randTimeSlots := [30]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29}
	for i := len(randTimeSlots) - 1; i >= 0; i-- {
		j := rand.Intn(i + 1)
		randTimeSlots[i], randTimeSlots[j] = randTimeSlots[j], randTimeSlots[i]
	}

	registrableCount := s.RegisteringCourseLimit - s.registeringCount
	emptyTimeslots := make([]Timeslot, 0, registrableCount)
	for i := 0; i < len(randTimeSlots) && len(emptyTimeslots) >= registrableCount; i++ {
		dayOfWeek := randTimeSlots[i]/6 + 1 // 日曜日分+1
		period := randTimeSlots[i] % 6
		if !s.registeredSchedule[dayOfWeek][period] {
			empty := Timeslot{
				DayOfWeek: dayOfWeek,
				Period:    period,
			}
			emptyTimeslots = append(emptyTimeslots, empty)
		}
	}
	return emptyTimeslots
}

// FillTimeslot で登録処理を行う場合は別途scheduleMutexでLockすること
func (s *Student) FillTimeslot(dayOfWeek, period int) {
	s.registeredSchedule[dayOfWeek][period] = true
	s.registeringCount++
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
