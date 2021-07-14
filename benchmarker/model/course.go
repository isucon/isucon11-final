package model

import "sync"

type Course struct {
	Name string

	rmu sync.RWMutex
}

func NewCourse(name string) *Course {
	return &Course{
		Name: name,
		rmu:  sync.RWMutex{},
	}
}
