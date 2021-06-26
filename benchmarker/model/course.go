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

func (c *Course) AddHeldClasses(class *Class) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	c.heldClasses = append(c.heldClasses, class)
}
func (c *Course) HeldClasses() []*Class {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	r := make([]*Class, 0)
	copy(r, c.heldClasses)
	return r
}
func (c *Course) GetHeldClassCount() int {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	return len(c.heldClasses)
}

func (c *Course) Students() []*Student {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	var r []*Student
	copy(r, c.registeredStudents)
	return r
}
