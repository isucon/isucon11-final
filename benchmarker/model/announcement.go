package model

import (
	"sync"
)

type Announcement struct {
	ID         string
	CourseID   string
	CourseName string
	Title      string
	Message    string
	CreatedAt  int64

	rmu sync.RWMutex
}

func NewAnnouncement(id, courseID, courseName, title, message string, createdAt int64) *Announcement {
	return &Announcement{
		ID:         id,
		CourseID:   courseID,
		CourseName: courseName,
		Title:      title,
		Message:    message,
		CreatedAt:  createdAt,
		rmu:        sync.RWMutex{},
	}
}
