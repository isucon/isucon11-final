package model

import "sync"

type Course struct {
	Name           string
	DetailChecksum string

	// ベンチの操作で変更されるデータ
	registeredStudents []*Student
	heldClasses        []*Class

	rmu sync.RWMutex
}

func NewCourse(name, checksum string) *Course {
	return &Course{
		Name:               name,
		DetailChecksum:     checksum,
		registeredStudents: []*Student{},
		heldClasses:        []*Class{},
		rmu:                sync.RWMutex{},
	}
}
