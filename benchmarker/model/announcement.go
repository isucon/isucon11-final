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

	rmu sync.RWMutex
}
