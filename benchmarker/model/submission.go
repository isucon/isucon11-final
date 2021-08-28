package model

import (
	"hash/crc32"
	"sync"
)

type Submission struct {
	Title    string
	Checksum uint32
	IsValid  bool

	// score は課題に対する講師によって追加されるスコア
	// 提出後採点されるまではNULL
	// 採点されたら採点された点
	score *int
	rmu   sync.RWMutex
}

func NewSubmission(title string, data []byte, isValid bool) *Submission {
	return &Submission{
		Title:    title,
		IsValid:  isValid,
		Checksum: crc32.ChecksumIEEE(data),
		rmu:      sync.RWMutex{},
	}
}

func (s *Submission) SetScore(score int) {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	s.score = &score
}
