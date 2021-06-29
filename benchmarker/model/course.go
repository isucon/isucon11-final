package model

import "sync"

type Course struct {
	ID             string
	Name           string
	Capacity       int
	Timeslots      []*TimeSlot
	Keyword        []string
	DetailChecksum string

	// ベンチの操作で変更されるデータ
	registeredStudents []*Student
	heldClasses        []*Class

	rmu sync.RWMutex
}

type TimeSlot struct {
	dayOfWeek int /* 0-6 Mon to Sun */
	classHour int /* 1-6 */
}

func NewCourse(id, name string, capacity, dayOfWeek, classHour int, keyword []string, checksum string) *Course {
	return &Course{
		ID:                 id,
		Name:               name,
		Capacity:           capacity,
		Timeslots:          []*TimeSlot{{dayOfWeek, classHour}},
		Keyword:            keyword,
		DetailChecksum:     checksum,
		registeredStudents: []*Student{},
		heldClasses:        []*Class{},
		rmu:                sync.RWMutex{},
	}
}

func (c *Course) isFull() bool {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	if len(c.registeredStudents) >= c.Capacity {
		return true
	}
	return false
}

func (c *Course) AddStudent(student *Student) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	c.registeredStudents = append(c.registeredStudents, student)
}
