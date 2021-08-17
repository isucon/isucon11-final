package model

import (
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
	ID string

	userScores map[string]*ClassScore

	rmu sync.RWMutex
}

func NewClass(id string, param *ClassParam) *Class {
	return &Class{
		ClassParam: param,
		ID:         id,

		userScores: make(map[string]*ClassScore, 20),

		rmu: sync.RWMutex{},
	}
}

func (c *Class) InsertUserScores(userCode string, score int) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	if _, ok := c.userScores[userCode]; !ok {
		c.userScores[userCode] = NewClassScore(c, score)
	}

	c.userScores[userCode].Score = score
}

func (c *Class) RemoveUserScores(userCode string) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	delete(c.userScores, userCode)
}
