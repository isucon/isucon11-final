package model

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/isucon/isucon11-final/benchmarker/api"
	"github.com/isucon/isucon11-final/benchmarker/util"
)

type ReservationResult int

const (
	Succeeded ReservationResult = iota
	NotAvailable
)

const (
	// ClassCountPerCourse は科目あたりのクラス数 -> used in model/course.go
	ClassCountPerCourse = 5
)

type CourseParam struct {
	Code        string
	Type        string
	Name        string
	Description string
	Credit      int
	Teacher     string
	Period      int
	DayOfWeek   int
	Keywords    string
}

type Course struct {
	*CourseParam
	ID                 string
	teacher            *Teacher
	registeredStudents map[string]*Student
	reservations       int
	capacity           int // 登録学生上限
	capacityCounter    *CapacityCounter
	classes            []*Class
	status             api.CourseStatus

	closer              chan struct{}
	zeroReservationCond *sync.Cond
	once                sync.Once
	rmu                 sync.RWMutex
}

type SearchCourseParam struct {
	Type      string
	Credit    int
	Teacher   string
	Period    int
	DayOfWeek int
	Keywords  []string
	Status    string
}

func NewCourse(param *CourseParam, id string, teacher *Teacher, capacity int, capacityCounter *CapacityCounter) *Course {
	c := &Course{
		CourseParam:        param,
		ID:                 id,
		teacher:            teacher,
		registeredStudents: make(map[string]*Student, capacity),
		capacity:           capacity,
		capacityCounter:    capacityCounter,
		classes:            make([]*Class, 0, ClassCountPerCourse),
		status:             api.StatusRegistration,

		closer: make(chan struct{}, 0),
	}
	c.zeroReservationCond = sync.NewCond(&c.rmu)
	return c
}

func (c *Course) AddClass(class *Class) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	c.classes = append(c.classes, class)
}

func (c *Course) Status() api.CourseStatus {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	return c.status
}

func (c *Course) SetStatusToInProgress() {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	c.status = api.StatusInProgress
}

func (c *Course) SetStatusToClosed() {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	c.status = api.StatusClosed
}

func (c *Course) Wait(ctx context.Context, cancel context.CancelFunc, addCourseFunc func()) <-chan struct{} {
	go func() {
		select {
		case <-c.closer:
			// 科目の履修を締め切ったときに、次の科目を追加する
			addCourseFunc()
		case <-ctx.Done():
		}
		c.zeroReservationCond.L.Lock()
		for c.reservations > 0 {
			c.zeroReservationCond.Wait()
		}
		c.zeroReservationCond.L.Unlock()
		cancel()
	}()
	return ctx.Done()
}

func (c *Course) Teacher() *Teacher {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	return c.teacher
}

func (c *Course) Students() map[string]*Student {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	s := make(map[string]*Student, len(c.registeredStudents))
	for userCode, user := range c.registeredStudents {
		s[userCode] = user
	}

	return s
}

func (c *Course) Classes() []*Class {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	cs := make([]*Class, len(c.classes))
	copy(cs, c.classes[:])

	return cs
}

// ReserveIfAvailable は履修受付中なら1枠確保する
func (c *Course) ReserveIfAvailable() ReservationResult {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	select {
	case <-c.closer:
		return NotAvailable
	default:
	}
	if len(c.registeredStudents)+c.reservations >= c.capacity {
		return NotAvailable
	}

	c.reservations++

	return Succeeded
}

func (c *Course) CommitReservation(s *Student) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	c.registeredStudents[s.Code] = s
	c.reservations--
	if c.reservations == 0 {
		c.zeroReservationCond.Broadcast()
	}

	// 満員が確定した時、履修を締め切る
	if len(c.registeredStudents) == c.capacity {
		select {
		case <-c.closer:
			// close済み
		default:
			close(c.closer)
		}
	}
	// 仮登録者数がゼロで履修する可能性のある学生がいない時、履修を締め切る
	if c.reservations == 0 && c.capacityCounter.Get(c.DayOfWeek, c.Period) == 0 {
		select {
		case <-c.closer:
			// close済み
		default:
			close(c.closer)
		}
	}
}

func (c *Course) RollbackReservation() {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	c.reservations--
	if c.reservations == 0 {
		c.zeroReservationCond.Broadcast()
	}

	// 仮登録者数がゼロで履修する可能性のある学生がいない時、履修を締め切る
	if c.reservations == 0 && c.capacityCounter.Get(c.DayOfWeek, c.Period) == 0 {
		select {
		case <-c.closer:
			// close済み
		default:
			close(c.closer)
		}
	}
}

func (c *Course) StartTimer(duration time.Duration) {
	c.once.Do(func() {
		go func() {
			time.Sleep(duration)

			c.rmu.Lock()
			defer c.rmu.Unlock()
			select {
			case <-c.closer:
				// close済み
			default:
				close(c.closer)
			}
		}()
	})
}

// for prepare
func (c *Course) AddStudent(student *Student) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	c.registeredStudents[student.Code] = student
}

func (c *Course) BroadCastAnnouncement(a *Announcement) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	for _, s := range c.registeredStudents {
		s.AddAnnouncement(a)
		s.AddUnreadAnnouncement(a)
	}
}

func (c *Course) CollectSimpleClassScores(userCode string) []*SimpleClassScore {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	res := make([]*SimpleClassScore, 0, len(c.classes))
	for _, class := range c.classes {
		res = append(res, class.IntoSimpleClassScore(userCode))
	}

	return res
}

func (c *Course) CollectClassScores(userCode string) []*ClassScore {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	res := make([]*ClassScore, 0, len(c.classes))
	for _, class := range c.classes {
		res = append(res, class.IntoClassScore(userCode))
	}

	return res
}

func (c *Course) CalcCourseResultByStudentCode(code string) *CourseResult {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	if _, ok := c.registeredStudents[code]; !ok {
		// TODO: unreachable
		return nil
	}

	totalScores := c.calcTotalScores()

	totalScoresArr := make([]int, 0, len(totalScores))
	for _, totalScore := range totalScores {
		totalScoresArr = append(totalScoresArr, totalScore)
	}
	totalAvg := util.AverageInt(totalScoresArr, 0)
	totalMax := util.MaxInt(totalScoresArr, 0)
	totalMin := util.MinInt(totalScoresArr, 0)

	totalScore, ok := totalScores[code]
	if !ok {
		panic("unreachable! userCode: " + code)
	}
	totalTScore := util.TScoreInt(totalScore, totalScoresArr)

	classScores := c.CollectClassScores(code)

	return &CourseResult{
		Name:             c.Name,
		Code:             c.Code,
		TotalScore:       totalScore,
		TotalScoreTScore: totalTScore,
		TotalScoreAvg:    totalAvg,
		TotalScoreMax:    totalMax,
		TotalScoreMin:    totalMin,
		ClassScores:      classScores,
	}
}

func (c *Course) GetTotalScoreByStudentCode(code string) int {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	score := 0
	for _, class := range c.classes {
		submission := class.GetSubmissionByStudentCode(code)
		if submission != nil && submission.score != nil {
			score += *submission.score
		}
	}

	return score
}

func (c *Course) calcTotalScores() map[string]int {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	res := make(map[string]int, len(c.registeredStudents))
	for userCode := range c.registeredStudents {
		res[userCode] = 0
	}
	for _, class := range c.classes {
		for userCode, submission := range class.Submissions() {
			if submission != nil && submission.score != nil {
				res[userCode] += *submission.score
			}
		}
	}

	return res
}

func NewCourseParam() *SearchCourseParam {
	return &SearchCourseParam{
		Type:      "",
		Credit:    0,
		Teacher:   "",
		Period:    -1, // 0-5, -1で指定なし
		DayOfWeek: -1, // 0-4, -1で指定なし
		Keywords:  []string{},
		Status:    "",
	}
}

func (p *SearchCourseParam) GetParamString() string {
	paramStrings := make([]string, 0)
	if p.Type != "" {
		paramStrings = append(paramStrings, fmt.Sprintf("type = %s", p.Type))
	}
	if p.Credit != 0 {
		paramStrings = append(paramStrings, fmt.Sprintf("credit = %d", p.Credit))
	}
	if p.Teacher != "" {
		paramStrings = append(paramStrings, fmt.Sprintf("teacher = %s", p.Teacher))
	}
	if p.Period != -1 {
		paramStrings = append(paramStrings, fmt.Sprintf("period = %d", p.Period+1))
	}
	if p.DayOfWeek != -1 {
		paramStrings = append(paramStrings, fmt.Sprintf("day_of_week = %s", api.DayOfWeekTable[p.DayOfWeek]))
	}
	if len(p.Keywords) != 0 {
		paramStrings = append(paramStrings, fmt.Sprintf("keywords = %s", strings.Join(p.Keywords, " ")))
	}
	if p.Status != "" {
		paramStrings = append(paramStrings, fmt.Sprintf("status = %s", p.Status))
	}

	if len(paramStrings) == 0 {
		return "empty"
	} else {
		return strings.Join(paramStrings, ", ")
	}
}
