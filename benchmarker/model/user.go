package model

import (
	"context"
	"net/url"
	"sync"

	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/random/useragent"

	"github.com/isucon/isucon11-final/benchmarker/api"
)

type UserAccount struct {
	ID          string // for generate course
	Code        string
	Name        string
	RawPassword string
	IsAdmin     bool
}

type Student struct {
	*UserAccount
	Agent *agent.Agent

	registeredCourses     []*Course
	announcements         []*AnnouncementStatus // announcements は生成順でソートされている保証はない
	announcementIndexByID map[string]int
	readAnnouncementCond  *sync.Cond // おしらせの既読を監視するCond
	addAnnouncementCond   *sync.Cond // おしらせの追加を監視するCond
	rmu                   sync.RWMutex

	registeredSchedule [5][6]*Course // 空きコマ管理[DayOfWeek:5][Period:6]
	registeringCount   int
	scheduleMutex      sync.RWMutex
	scheduleCond       *sync.Cond // スケジュールの空きを監視するCond
}
type AnnouncementStatus struct {
	Announcement *Announcement
	Dirty        bool // リクエストを送ったがタイムアウトになってしまったため、webapp側で既読になったかが定かではないことを表す
	Unread       bool
}

func NewStudent(userData *UserAccount, baseURL *url.URL) *Student {
	a, _ := agent.NewAgent()
	a.Name = useragent.UserAgent()
	a.BaseURL = baseURL

	s := &Student{
		UserAccount: userData,
		Agent:       a,

		registeredCourses:     make([]*Course, 0, 20),
		announcements:         make([]*AnnouncementStatus, 0, 100),
		announcementIndexByID: make(map[string]int, 100),
		rmu:                   sync.RWMutex{},

		registeredSchedule: [5][6]*Course{},
		scheduleMutex:      sync.RWMutex{},
	}
	s.readAnnouncementCond = sync.NewCond(&s.rmu)
	s.addAnnouncementCond = sync.NewCond(&s.rmu)
	s.scheduleCond = sync.NewCond(&s.scheduleMutex)
	return s
}

func (s *Student) AddCourse(course *Course) {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	s.registeredCourses = append(s.registeredCourses, course)
}

func (s *Student) Announcements() []*AnnouncementStatus {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	return s.announcements
}

func (s *Student) AnnouncementsMap() map[string]*AnnouncementStatus {
	s.rmu.RLock()
	defer s.rmu.RUnlock()

	result := make(map[string]*AnnouncementStatus, len(s.announcements))
	for _, announcement := range s.announcements {
		tmp := *announcement
		result[announcement.Announcement.ID] = &tmp
	}
	return result
}

func (s *Student) AddAnnouncement(announcement *Announcement) {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	s.announcements = append(s.announcements, &AnnouncementStatus{
		Announcement: announcement,
		Dirty:        false,
		Unread:       true,
	})
	s.announcementIndexByID[announcement.ID] = len(s.announcements) - 1
	s.addAnnouncementCond.Broadcast()
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

func (s *Student) AnnouncementCount() int {
	s.rmu.RLock()
	defer s.rmu.RUnlock()

	return len(s.announcements)
}

func (s *Student) MarkAnnouncementReadDirty(id string) {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	s.announcements[s.announcementIndexByID[id]].Dirty = true
}

func (s *Student) ReadAnnouncement(id string) {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	s.announcements[s.announcementIndexByID[id]].Dirty = false
	s.announcements[s.announcementIndexByID[id]].Unread = false
	s.readAnnouncementCond.Broadcast()
}

func (s *Student) isUnreadAnnouncement(id string) bool {
	return s.announcements[s.announcementIndexByID[id]].Unread
}

func (s *Student) HasUnreadAnnouncement() bool {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	for _, anc := range s.announcements {
		if anc.Unread {
			return true
		}
	}
	return false
}

func (s *Student) WaitNewUnreadAnnouncement(ctx context.Context) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
		case <-s.waitAddAnnouncement():
		}
		close(ch)
	}()
	return ch
}

func (s *Student) waitAddAnnouncement() <-chan struct{} {
	ch := make(chan struct{})
	// MEMO: このgoroutineはWaitNewUnreadAnnouncementがctx.Done()で抜けた場合放置される
	go func() {
		s.addAnnouncementCond.L.Lock()
		s.addAnnouncementCond.Wait()
		s.addAnnouncementCond.L.Unlock()
		close(ch)
	}()
	return ch
}

func (s *Student) WaitReadAnnouncement(ctx context.Context, id string) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			close(ch)
			return
		case <-s.waitReadAnnouncement(id):
		}
		close(ch)
	}()
	return ch
}

func (s *Student) waitReadAnnouncement(id string) <-chan struct{} {
	ch := make(chan struct{})

	// MEMO: このgoroutineはWaitReadAnnouncementがctx.Done()で抜けた場合放置される
	s.rmu.RLock()
	if s.isUnreadAnnouncement(id) {
		go func() {
			s.readAnnouncementCond.L.Lock()
			for s.isUnreadAnnouncement(id) {
				s.readAnnouncementCond.Wait()
			}
			s.readAnnouncementCond.L.Unlock()
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

func (s *Student) LockSchedule() {
	s.scheduleMutex.Lock()
}

func (s *Student) UnlockSchedule() {
	s.scheduleMutex.Unlock()
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

func (s *Student) ReleaseTimeslot(dayOfWeek, period int) {
	s.scheduleMutex.Lock()
	defer s.scheduleMutex.Unlock()

	s.registeredSchedule[dayOfWeek][period] = nil
	s.registeringCount--
	s.scheduleCond.Broadcast()
}

func (s *Student) Courses() []*Course {
	s.rmu.RLock()
	defer s.rmu.RUnlock()

	res := make([]*Course, len(s.registeredCourses))
	copy(res, s.registeredCourses[:])

	return res
}

func (s *Student) HasFinishedCourse() bool {
	s.rmu.RLock()
	defer s.rmu.RUnlock()

	for _, course := range s.registeredCourses {
		if course.Status() == api.StatusClosed {
			return true
		}
	}
	return false
}

func (s *Student) RegisteredSchedule() [5][6]*Course {
	s.scheduleMutex.RLock()
	defer s.scheduleMutex.RUnlock()

	return s.registeredSchedule
}

func (s *Student) WaitReleaseTimeslot(ctx context.Context, cancel context.CancelFunc, registerCourseLimit int) <-chan struct{} {
	go func() {
		select {
		case <-ctx.Done():
		case <-s.waitReleaseTimeslot(registerCourseLimit):
		}
		cancel()
	}()
	return ctx.Done()
}

func (s *Student) waitReleaseTimeslot(registerCourseLimit int) <-chan struct{} {
	ch := make(chan struct{})
	// MEMO: このgoroutineはWaitReleaseTimeslotがctx.Done()で抜けた場合放置される
	go func() {
		s.scheduleCond.L.Lock()
		for s.registeringCount >= registerCourseLimit {
			s.scheduleCond.Wait()
		}
		s.scheduleCond.L.Unlock()
		close(ch)
	}()
	return ch
}

func (s *Student) GPA() float64 {
	s.rmu.RLock()
	defer s.rmu.RUnlock()

	tmp := 0
	for _, course := range s.registeredCourses {
		if course.Status() == api.StatusClosed {
			tmp += course.GetTotalScoreByStudentCode(s.Code) * course.Credit
		}
	}

	gpt := float64(tmp) / 100.0
	credits := s.TotalCredit()

	if credits == 0 {
		return 0
	}
	return gpt / float64(credits)
}

func (s *Student) TotalCredit() int {
	s.rmu.RLock()
	defer s.rmu.RUnlock()

	res := 0
	for _, course := range s.registeredCourses {
		if course.Status() == api.StatusClosed {
			res += course.Credit
		}
	}

	return res
}

type Teacher struct {
	*UserAccount
	Agent      *agent.Agent
	IsLoggedIn bool

	mu sync.Mutex
}

const teacherUserAgent = "isucholar-agent-teacher/1.0.0"

func NewTeacher(userData *UserAccount, baseURL *url.URL) *Teacher {
	a, _ := agent.NewAgent()
	a.BaseURL = baseURL
	a.Name = teacherUserAgent
	return &Teacher{
		UserAccount: userData,
		Agent:       a,
	}
}

func (t *Teacher) LoginOnce(f func(teacher *Teacher)) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.IsLoggedIn {
		return true
	}
	f(t)

	return t.IsLoggedIn
}
