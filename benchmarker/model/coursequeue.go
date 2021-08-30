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

func (q *courseList) Len() int {
	return len(q.items)
}

func (q *courseList) Seek(i int) *Course {
	return q.items[i]
}

func (q *courseList) Add(c *Course) {
	q.items = append(q.items, c)
}

func (q *courseList) Remove(i int) {
	q.items = append(q.items[:i], q.items[i+1:]...)
}
