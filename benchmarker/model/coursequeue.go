package model

import (
	"sync"
)

type courseQueue struct {
	items []*Course
	rmu   sync.RWMutex
}

func newCourseQueue(cap int) *courseQueue {
	m := make([]*Course, 0, cap)
	return &courseQueue{
		items: m,
	}
}

func (q *courseQueue) Lock() {
	q.rmu.Lock()
}

func (q *courseQueue) Unlock() {
	q.rmu.Unlock()
}

func (q *courseQueue) RLock() {
	q.rmu.RLock()
}

func (q *courseQueue) RUnlock() {
	q.rmu.RUnlock()
}

func (q *courseQueue) Len() int {
	return len(q.items)
}

func (q *courseQueue) Seek(i int) *Course {
	return q.items[i]
}

func (q *courseQueue) Add(c *Course) {
	q.items = append(q.items, c)
}

func (q *courseQueue) Remove(i int) {
	q.items = append(q.items[:i], q.items[i+1:]...)
}
