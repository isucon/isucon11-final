package model

import (
	"context"
	"sync"
	"time"

	"github.com/isucon/isucon11-final/benchmarker/util"
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
	classes            []*Class
	registeredLimit    int // 登録学生上限
	rmu                sync.RWMutex

	// コース登録を締切る際に参照
	registrationCloser   chan struct{} // 登録が締め切られるとcloseする
	tempRegCount         int
	tempRegZeroCountCond *sync.Cond
	timerOnce            sync.Once
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
		registeredStudents: make(map[string]*Student, 0),
		registeredLimit:    50, // 引数で渡す？
		rmu:                sync.RWMutex{},

		registrationCloser: make(chan struct{}, 0),
		tempRegCount:       0,
		timerOnce:          sync.Once{},
	}
	c.tempRegZeroCountCond = sync.NewCond(&c.rmu)
	return c
}

func (c *Course) AddClass(class *Class) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	c.classes = append(c.classes, class)
}

// WaitPreparedCourse はコースに学生が追加されなくなるか、ctx.Done()になるのを待つ
func (c *Course) WaitPreparedCourse(ctx context.Context) <-chan struct{} {
	ch := make(chan struct{}, 0)
	go func() {
		// 内部的な履修締切（時間 or 人数）までwaitする
		select {
		case <-ctx.Done():
			close(ch)
			return
		case <-c.registrationCloser:
		}

		// 全員の仮登録が完了する(=仮登録者が0になる)のを待つ
		// webapp側に登録完了してないのにベンチがコース処理を始めると不整合がでるため
		select {
		case <-ctx.Done():
			close(ch)
			return
		case <-c.waitTempRegCountIsZero():
		}

		close(ch)
	}()
	return ch
}

func (c *Course) waitTempRegCountIsZero() <-chan struct{} {
	ch := make(chan struct{}, 0)
	// MEMO: このgoroutineはWaitPreparedCourseがctx.Done()で抜けた場合放置される
	go func() {
		c.tempRegZeroCountCond.L.Lock()
		for c.tempRegCount > 0 {
			c.tempRegZeroCountCond.Wait()
		}
		c.tempRegZeroCountCond.L.Unlock()
		close(ch)
	}()
	return ch
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

func (c *Course) BroadCastAnnouncement(a *Announcement) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	for _, s := range c.registeredStudents {
		s.AddAnnouncement(a)
	}
}

// TempRegisterIfRegistrable は履修受付中なら仮登録者を1人増やす
func (c *Course) TempRegisterIfRegistrable() bool {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	select {
	case _, _ = <-c.registrationCloser:
		// close済み
		return false
	default:
	}

	// 履修closeしていない場合は仮登録する
	c.tempRegCount++ // コース仮登録者+1
	if len(c.registeredStudents)+c.tempRegCount >= c.registeredLimit {
		// 本登録 + 仮登録が上限以上ならcloseする
		close(c.registrationCloser)
	}

	return true
}

func (c *Course) SuccessRegistration(student *Student) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	c.registeredStudents[student.Code] = student
	c.tempRegCount--
	if c.tempRegCount <= 0 {
		c.tempRegZeroCountCond.Broadcast()
	}
}

func (c *Course) FailRegistration() {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	c.tempRegCount--
	if c.tempRegCount <= 0 {
		c.tempRegZeroCountCond.Broadcast()
	}
}

func (c *Course) SetClosingAfterSecAtOnce(duration time.Duration) {
	c.timerOnce.Do(func() {
		go func() {
			time.Sleep(duration)

			c.rmu.Lock()
			defer c.rmu.Unlock()

			select {
			case _, _ = <-c.registrationCloser:
				// close済み
			default:
				close(c.registrationCloser)
			}
		}()
	})
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

	totalScores := c.TotalScores()

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
		sub := class.GetSubmissionByStudentCode(code)
		if sub != nil {
			score += sub.score
		}
	}

	return score
}

func (c *Course) TotalScores() map[string]int {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	res := make(map[string]int, len(c.registeredStudents))
	for userCode := range c.registeredStudents {
		res[userCode] = 0
	}
	for _, class := range c.classes {
		for userCode, summary := range class.Submissions() {
			res[userCode] += summary.score
		}
	}

	return res
}
