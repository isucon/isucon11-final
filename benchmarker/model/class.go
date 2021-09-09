package model

import (
	"sync"
)

type ClassParam struct {
	Title string
	Desc  string
	Part  uint8 // n回目のクラス
}

type Class struct {
	*ClassParam
	ID          string
	submissions map[string]*Submission // map[学籍番号]*Submission
	rmu         sync.RWMutex
}

func NewClass(id string, param *ClassParam) *Class {
	return &Class{
		ClassParam:  param,
		ID:          id,
		submissions: make(map[string]*Submission),
		rmu:         sync.RWMutex{},
	}
}

func (c *Class) AddSubmission(studentCode string, summary *Submission) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	// ここでsummary=nilをセットするとGetSubmissionByStudentCode(studentCode)で存在チェックしたいときに区別つかなくなる
	c.submissions[studentCode] = summary
}

func (c *Class) GetSubmissionByStudentCode(code string) *Submission {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	return c.submissions[code]
}

func (c *Class) Submissions() map[string]*Submission {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	res := make(map[string]*Submission, len(c.submissions))
	for s, summary := range c.submissions {
		res[s] = summary
	}
	return res
}

func (c *Class) IntoSimpleClassScore(userCode string) *SimpleClassScore {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	var score *int
	if v, ok := c.submissions[userCode]; ok {
		score = v.Score()
	}

	return &SimpleClassScore{
		ClassID: c.ID,
		Title:   c.Title,
		Part:    c.Part,
		Score:   score,
	}
}

func (c *Class) IntoClassScore(userCode string) *ClassScore {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	var score *int
	if v, ok := c.submissions[userCode]; ok {
		score = v.Score()
	}

	return &ClassScore{
		ClassID:        c.ID,
		Title:          c.Title,
		Part:           c.Part,
		Score:          score,
		SubmitterCount: len(c.submissions),
	}
}
