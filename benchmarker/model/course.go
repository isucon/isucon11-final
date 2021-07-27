package model

import (
	"context"
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
	registeredLimit    int // 登録学生上限
	registrable        bool
	tempRegistered     uint32

	closer chan struct{}

	once sync.Once
	rmu  sync.RWMutex
}

func NewCourse(param *CourseParam, id string, faculty *Faculty) *Course {
	return &Course{
		CourseParam:        param,
		ID:                 id,
		faculty:            faculty,
		registeredStudents: make([]*Student, 0),
		registeredLimit:    50, // 引数で渡す？
		registrable:        true,

		tempRegistered: 0,
		rmu:            sync.RWMutex{},
	}
}

func (c *Course) WaitFullOrUnRegistrable(ctx context.Context) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
		case <-c.closer:
		}
		ch <- struct{}{}
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

func (c *Course) BroadCastAnnouncement(a *Announcement) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	for _, s := range c.registeredStudents {
		s.AddAnnouncement(a)
	}
}

func (c *Course) RegisterStudentsIfRegistrable(student *Student) (isSuccess, isRegistrable bool) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	if c.registrable && len(c.registeredStudents) >= c.registeredLimit {
		isSuccess = false
		isRegistrable = false
		c.setToUnRegistrable()
		return
	}

	c.registeredStudents = append(c.registeredStudents, student)
	c.tempRegistered++
	isSuccess = true
	if len(c.registeredStudents) >= c.registeredLimit {
		isRegistrable = false
		c.setToUnRegistrable()
	} else {
		isRegistrable = true
	}
	return
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
	c.tempRegistered--
}

func (c *Course) ReduceTempRegistered() {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	c.tempRegistered--
}

func (c *Course) SetUnRegistrableAfterSecAtOnce(sec time.Duration) {
	c.once.Do(func() {
		go func() {
			<-time.After(sec)

			c.rmu.Lock()
			defer c.rmu.Unlock()
			c.setToUnRegistrable()
		}()
	})
}

func (c *Course) setToUnRegistrable() {
	c.registrable = false
	close(c.closer)
}
