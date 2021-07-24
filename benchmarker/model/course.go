package model

import (
	"context"
	"sync"
	"time"
)

type CourseParam struct {
	Type      string
	Name      string
	Credit    int
	Teacher   string
	Period    int
	DayOfWeek string
	Keywords  string
}

type Course struct {
	*CourseParam
	ID                 string
	faculty            *Faculty
	registeredStudents []*Student

	rmu sync.RWMutex
}

func NewCourse(id string, param *CourseParam, faculty *Faculty) *Course {
	return &Course{
		CourseParam:        param,
		ID:                 id,
		faculty:            faculty,
		registeredStudents: make([]*Student, 0),
		rmu:                sync.RWMutex{},
	}
}

func (c *Course) WaitRegister(ctx context.Context) <-chan struct{} {
	// FIXME: debug
	ch := make(chan struct{})
	go func() {
		<-time.After(1000 * time.Millisecond)
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
