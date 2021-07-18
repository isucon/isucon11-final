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
