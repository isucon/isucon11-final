package model

import (
	"context"
	"log"
	"sync"
	"time"
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
	faculty            *Faculty
	registeredStudents []*Student
	classes            []*Class
	registeredLimit    int // 登録学生上限
	rmu                sync.RWMutex

	// コース登録を締切る際に参照
	registrationCloser chan struct{} // 登録が締め切られるとcloseする
	timerOnce          sync.Once
	tempRegStudents    sync.WaitGroup // ベンチ内で仮登録して本登録リクエストが完了していない生徒たち

	UserScores map[string]*SimpleCourseResult
}

type SearchCourseParam struct {
	Type      string
	Credit    int
	Teacher   string
	Period    int
	DayOfWeek int
	Keywords  []string
}

func NewCourse(param *CourseParam, id string, faculty *Faculty) *Course {
	return &Course{
		CourseParam:        param,
		ID:                 id,
		faculty:            faculty,
		registeredStudents: make([]*Student, 0),
		registeredLimit:    50, // 引数で渡す？
		rmu:                sync.RWMutex{},

		registrationCloser: make(chan struct{}, 0),
		timerOnce:          sync.Once{},
		tempRegStudents:    sync.WaitGroup{},

		UserScores: make(map[string]*SimpleCourseResult, 20),
	}
}

func (c *Course) AddClass(class *Class) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	c.classes = append(c.classes, class)
}

func (c *Course) WaitPreparedCourse(ctx context.Context) <-chan struct{} {
	ch := make(chan struct{}, 0)
	go func() {
		select {
		case <-ctx.Done():
			close(ch)
			return
		case <-c.registrationCloser:
		}

		// webapp側に登録完了してないのにベンチがコース処理を始めると不整合がでるため
		// 学生の登録リクエストが完了するのを待つ
		c.tempRegStudents.Wait()
		close(ch)
	}()
	return ch
}

func (c *Course) Faculty() *Faculty {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	return c.faculty
}

func (c *Course) Students() []*Student {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	s := make([]*Student, len(c.registeredStudents))
	copy(s, c.registeredStudents[:])

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

func (c *Course) RemoveStudent(student *Student) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	registeredStudents := make([]*Student, 0, len(c.registeredStudents))
	for _, s := range c.registeredStudents {
		if s != student {
			registeredStudents = append(registeredStudents, s)
		}
	}
	c.registeredStudents = registeredStudents
}

func (c *Course) RegisterStudentsIfRegistrable(student *Student) bool {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	select {
	case _, _ = <-c.registrationCloser:
		// close済み
		return false
	default:
	}

	// 履修closeしていない場合は仮登録する
	c.registeredStudents = append(c.registeredStudents, student)
	c.tempRegStudents.Add(1) // コース仮登録者+1
	if len(c.registeredStudents) >= c.registeredLimit {
		// 満員になっていたらcloseする
		close(c.registrationCloser)
	}

	return true
}

// 成功失敗に関わらず学生による本登録処理が終了した
func (c *Course) FinishRegistration() {
	c.tempRegStudents.Done() // コース仮登録者-1
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

func (c *Course) InsertUserScores(userCode string, score int, class *Class) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	log.Println("here")
	if v, ok := c.UserScores[userCode]; ok {
		log.Println("here1")
		v.ClassScores = append(v.ClassScores, NewClassScore(class, score))
	} else {
		log.Println("here2")
		c.UserScores[userCode] = NewSimpleCourseResult(c.Name, userCode, score, class)
	}
}
