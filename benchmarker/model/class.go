package model

import "sync"

type Class struct {
	ClassID string

	// ベンチの操作で変更されるデータ
	submittedAssignmentChecksums []string // 提出課題をベンチ側はどう保持するかは要検討([]byte？)
	submittedStudentsIds         []string
	attendedStudentsIds          []string

	rmu sync.RWMutex
}
