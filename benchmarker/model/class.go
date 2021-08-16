package model

import (
	"crypto/md5"
	"sync"
)

type ClassParam struct {
	Title     string
	Desc      string
	Part      uint8 // n回目のクラス
	CreatedAt int64
}

type Class struct {
	*ClassParam
	ID                   string
	submittedAssignments map[string][16]byte // 学籍番号 -> 課題ファイルhash値
	rmu                  sync.RWMutex
}

func NewClass(id string, param *ClassParam) *Class {
	return &Class{
		ClassParam:           param,
		ID:                   id,
		submittedAssignments: make(map[string][16]byte),
		rmu:                  sync.RWMutex{},
	}
}

func (c *Class) AddSubmittedAssignment(studentCode string, data []byte) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	c.submittedAssignments[studentCode] = md5.Sum(data)
}

func (c *Class) GetAssignmentHash(studentCode string) ([16]byte, bool) {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	hash, exists := c.submittedAssignments[studentCode]
	return hash, exists
}

func (c *Class) GetSubmittedCount() int {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	return len(c.submittedAssignments)
}
