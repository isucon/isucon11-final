package model

import (
	"math/rand"
	"sync"
)

type Announcement struct {
	ID         string
	CourseID   string
	CourseName string
	Title      string
	Unread     bool
	CreatedAt  int64

	rmu sync.RWMutex
}

func NewAnnouncement(id, courseID, courseName, title string, unread bool) *Announcement {
	return &Announcement{
		ID:         id,
		CourseID:   courseID,
		CourseName: courseName,
		Title:      title,
		Unread:     unread,
		CreatedAt:  rand.Int63(),
		rmu:        sync.RWMutex{},
	}

}

type AnnouncementDeque struct {
	items []*Announcement
	head  int
	tail  int
	len   int
	cap   int

	rmu sync.RWMutex
}

func NewAnnouncementDeque(cap int) *AnnouncementDeque {
	m := make([]*Announcement, cap)
	return &AnnouncementDeque{
		items: m,
		head:  0,
		tail:  0,
		len:   0,
		cap:   cap,
	}
}

func (q *AnnouncementDeque) isEmpty() bool {
	return q.tail == q.head
}

func (q *AnnouncementDeque) IsEmpty() bool {
	q.rmu.RLock()
	defer q.rmu.RUnlock()

	return q.isEmpty()
}

func (q *AnnouncementDeque) isFull() bool {
	return q.cap-q.len == 1
}

func (q *AnnouncementDeque) PushBack(a *Announcement) {
	q.rmu.Lock()
	defer q.rmu.Unlock()

	if q.isFull() {
		q.grow()
	}
	q.items[q.tail] = a
	q.tail = (q.tail + 1) % q.cap
	q.len++
}

func (q *AnnouncementDeque) PushFront(a *Announcement) {
	q.rmu.Lock()
	defer q.rmu.Unlock()

	if q.isFull() {
		q.grow()
	}

	if q.head == 0 {
		q.head = q.cap - 1
	} else {
		q.head--
	}
	q.items[q.head] = a
	q.len++
}

func (q *AnnouncementDeque) PopBack() *Announcement {
	q.rmu.Lock()
	defer q.rmu.Unlock()
	if q.isEmpty() {
		return nil
	}

	if q.tail == 0 {
		q.tail = q.cap - 1
	} else {
		q.tail--
	}
	a := q.items[q.tail]
	q.len--
	return a

}

func (q *AnnouncementDeque) PopFront() *Announcement {
	q.rmu.Lock()
	defer q.rmu.Unlock()

	if q.isEmpty() {
		return nil
	}

	items := q.items[q.head]
	q.head = (q.head + 1) % q.cap

	q.len--
	return items
}

func (q *AnnouncementDeque) Len() int {
	q.rmu.RLock()
	defer q.rmu.RUnlock()

	return q.len
}

func (q *AnnouncementDeque) Cap() int {
	q.rmu.RLock()
	defer q.rmu.RUnlock()

	return q.cap
}

func (q *AnnouncementDeque) Items() []*Announcement {
	q.rmu.RLock()
	defer q.rmu.RUnlock()

	m := make([]*Announcement, q.Cap())
	copy(m, q.items)

	return m
}

//func (q *AnnouncementDeque) Copy() *AnnouncementDeque {
//	m := make([]*Announcement, q.Cap())
//	copy(m, q.items)
//	return &AnnouncementDeque{
//		items: m,
//		head:  q.head,
//		tail:  q.tail,
//		len:   q.len,
//		cap:   q.cap,
//		rmu:   sync.RWMutex{},
//	}
//}

func (q *AnnouncementDeque) grow() {
	curCap := q.cap
	nextCap := curCap * 2

	m := make([]*Announcement, nextCap)
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
