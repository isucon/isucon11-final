package model

import (
	"hash/crc32"
	"sync"
)

type Submission struct {
	Title    string
	Checksum uint32

	// score は課題に対する教員によって追加される採点結果
	// 提出後採点されるまではNULL
	// 採点されたら採点された点
	score *int
	rmu   sync.RWMutex
}

func NewSubmission(title string, data []byte) *Submission {
	return &Submission{
		Title:    title,
		Checksum: crc32.ChecksumIEEE(data),
		rmu:      sync.RWMutex{},
	}
}

func (s *Submission) SetScore(score int) {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	s.score = &score
}

func (s *Submission) Score() *int {
	s.rmu.RLock()
	defer s.rmu.RUnlock()

	if s.score != nil {
		scpy := *s.score
		return &scpy
	}

	return nil
}
