package model

import (
	"sync"
)

type courseList struct {
	items []*Course
	sync.RWMutex
}

func newCourseList(cap int) *courseList {
	m := make([]*Course, 0, cap)
	return &courseList{
		items: m,
	}
}

func (cl *courseList) Len() int {
	return len(cl.items)
}

func (cl *courseList) Seek(i int) *Course {
	return cl.items[i]
}

func (cl *courseList) Add(c *Course) {
	cl.items = append(cl.items, c)
}

func (cl *courseList) Remove(i int) {
	cl.items = append(cl.items[:i], cl.items[i+1:]...)
}
