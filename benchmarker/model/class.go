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
	ID                string
	submissionSummary map[string]*SubmissionSummary // 学籍番号 -> 課題ファイルchecksum
	rmu               sync.RWMutex
}

func NewClass(id string, param *ClassParam) *Class {
	return &Class{
		ClassParam:        param,
		ID:                id,
		submissionSummary: make(map[string]*SubmissionSummary),
		rmu:               sync.RWMutex{},
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

func (c *Class) AddSubmissionSummary(studentCode string, summary *SubmissionSummary) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	// ここでsummary=nilをセットするとSubmissionSummary(studentCode)で存在チェックしたいときに区別つかなくなる
	c.submissionSummary[studentCode] = summary
}

func (c *Class) SubmissionSummary(studentCode string) *SubmissionSummary {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	return c.submissionSummary[studentCode]
}

func (c *Class) GetSubmittedCount() int {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	return len(c.submissionSummary)
}
