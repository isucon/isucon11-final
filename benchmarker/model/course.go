package model

import (
	"context"
	"sync"

	"github.com/isucon/isucon11-final/benchmarker/util"
)

type ReservationResult string

const (
	Succeeded     ReservationResult = "successfully reserved"
	TemporaryFull ReservationResult = "this course is temporary full"
	Closed        ReservationResult = "this course is closed"
)

const (
	// StudentCapacityPerCourse は科目あたりの履修定員 -> used in model/course.go
	StudentCapacityPerCourse = 50
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
	ID                   string
	teacher              *Teacher
	registeredStudents   map[string]*Student
	reservations         int
	classes              []*Class
	isRegistrationClosed bool
	closer               chan struct{}
	rmu                  sync.RWMutex
}

type SearchCourseParam struct {
	Type      string
	Credit    int
	Teacher   string
	Period    int
	DayOfWeek int
	Keywords  []string
}

func NewCourse(param *CourseParam, id string, teacher *Teacher) *Course {
	c := &Course{
		CourseParam:        param,
		ID:                 id,
		teacher:            teacher,
		registeredStudents: make(map[string]*Student, StudentCapacityPerCourse),
		classes:            make([]*Class, 0, ClassCountPerCourse),
		closer:             make(chan struct{}, 0),
		rmu:                sync.RWMutex{},
	}
	return c
}

func (c *Course) AddClass(class *Class) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	c.classes = append(c.classes, class)
}

func (c *Course) Wait(ctx context.Context) <-chan struct{} {
	_ctx, cancel := context.WithCancel(ctx)
	go func() {
		select {
		case <-c.closer:
		case <-ctx.Done():
		}
		c.isRegistrationClosed = true
		cancel()
	}()
	return _ctx.Done()
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

// ReserveIfRegistrable は履修受付中なら1枠確保する
func (c *Course) ReserveIfRegistrable() ReservationResult {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	if c.isRegistrationClosed {
		return Closed
	}

	if len(c.registeredStudents)+c.reservations >= StudentCapacityPerCourse {
		return TemporaryFull
	}

	c.reservations++

	return Succeeded
}

func (c *Course) CommitReservation(s *Student) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	c.registeredStudents[s.Code] = s
	c.reservations--

	if c.reservations == 0 && len(c.registeredStudents) == StudentCapacityPerCourse {
		close(c.closer)
		c.isRegistrationClosed = true
	}
}

func (c *Course) RollbackReservation() {
	c.rmu.Lock()
	defer c.rmu.Unlock()
	c.reservations--
}

func (c *Course) BroadCastAnnouncement(a *Announcement) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	for _, s := range c.registeredStudents {
		s.AddAnnouncement(a)
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
