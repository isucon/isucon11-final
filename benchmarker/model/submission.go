package model

import (
	"hash/crc32"
	"sync"
)

type SubmissionSummary struct {
	Title    string
	Checksum uint32
	IsValid  bool

	// score は課題に対する講師によって追加されるスコア
	// 課題を出していて採点されていないときは0点
	// 課題を出していて採点されていればその採点された点数
	score int
	rmu   sync.RWMutex
}

func NewSubmissionSummary(title string, data []byte, isValid bool) *SubmissionSummary {
	return &SubmissionSummary{
		Title:    title,
		IsValid:  isValid,
		Checksum: crc32.ChecksumIEEE(data),
		rmu:      sync.RWMutex{},
	}
}

func (s *SubmissionSummary) SetScore(score int) {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	s.score = score
}

func (s *SubmissionSummary) Score() int {
	s.rmu.RLock()
	defer s.rmu.RUnlock()

	return s.score
}
