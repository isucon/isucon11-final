package util

import (
	"sync"

	"github.com/isucon/isucon11-final/benchmarker/model"
)

type courseQueue struct {
	items []*model.Course
	head  int
	tail  int
	len   int
	cap   int

	rmu sync.RWMutex
}

func newCourseQueue(cap int) *courseQueue {
	m := make([]*model.Course, cap)
	return &courseQueue{
		items: m,
		head:  0,
		tail:  0,
		len:   0,
		cap:   cap,
	}
}

func (q *courseQueue) isEmpty() bool {
	return q.tail == q.head
}
func (q *courseQueue) isFull() bool {
	return q.cap-q.len == 1
}

func (q *courseQueue) Push(a *model.Course) {
	q.rmu.Lock()
	defer q.rmu.Unlock()

	if q.isFull() {
		q.grow()
	}
	q.items[q.tail] = a
	q.tail = (q.tail + 1) % q.cap
	q.len++
}

func (q *courseQueue) Pop() *model.Course {
	q.rmu.Lock()
	defer q.rmu.Unlock()

	if q.isEmpty() {
		return nil
	}

	item := q.items[q.head]
	q.head = (q.head + 1) % q.cap

	q.len--
	return item
}

func (q *courseQueue) ReferrerHead() *model.Course {
	q.rmu.RLock()
	defer q.rmu.RUnlock()

	if q.isEmpty() {
		return nil
	}

	return q.items[q.head]
}

func (q *courseQueue) Len() int {
	q.rmu.RLock()
	defer q.rmu.RUnlock()

	return q.len
}

func (q *courseQueue) Cap() int {
	q.rmu.RLock()
	defer q.rmu.RUnlock()

	return q.cap
}

func (q *courseQueue) grow() {
	curCap := q.cap
	nextCap := curCap * 2

	m := make([]*model.Course, nextCap)
	if q.head < q.tail {
		copy(m, q.items)
	} else {
		//        T H
		// [o o o . o o ]
		//  H         T
		// [o o o o o . . . . . . .]
		copy(m[0:q.cap-q.head], q.items[q.head:])
		copy(m[q.cap-q.head:q.cap-q.head+q.tail], q.items[0:q.tail])
		q.head = 0
		q.tail = q.len
	}
	q.cap = nextCap
	q.items = m
}
