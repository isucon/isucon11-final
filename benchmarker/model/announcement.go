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
	CreatedAt  int64

	rmu sync.RWMutex
}

func NewAnnouncement(id, courseID, courseName, title string) *Announcement {
	return &Announcement{
		ID:         id,
		CourseID:   courseID,
		CourseName: courseName,
		Title:      title,
		CreatedAt:  rand.Int63(),
		rmu:        sync.RWMutex{},
	}
}
