package model

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/fails"
)

type hash []byte

type Class struct {
	ID          string
	Title       string
	CourseID    string
	Description string

	// ベンチの操作で変更されるデータ
	announcement          []*Announcement
	documentHashByDocID   map[string]hash
	attendedByStudentsIds map[string]bool
	assignmentID          string // MEMO: 取り急ぎ1Class1Assignmentとしておく
	submissionHashByName  map[string]hash

	rmu sync.RWMutex
}

func NewClass(id, courseID, title, desc string) *Class {
	return &Class{
		ID:          id,
		Title:       title,
		CourseID:    courseID,
		Description: desc,
	}
}

func (c *Class) AddAnnouncement(a *Announcement) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	c.announcement = append(c.announcement, a)
}
func (c *Class) Announcement() []*Announcement {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	r := make([]*Announcement, len(c.announcement))
	copy(r, c.announcement)
	return r
}

func (c *Class) AddDocHash(id string, hash []byte) error {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	if c.documentHashByDocID[id] != nil {
		return failure.NewError(fails.ErrApplication, fmt.Errorf("documentID(%s) is duplicated", id))
	}
	c.documentHashByDocID[id] = hash
	return nil
}
func (c *Class) HasDocumentHash(id string, hash []byte) bool {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	if bytes.Equal(c.documentHashByDocID[id], hash) {
		return true
	}
	return false
}
func (c *Class) EqualDocumentIDs(ids []string) bool {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	if len(c.documentHashByDocID) != len(ids) {
		return false
	}
	for _, id := range ids {
		if _, ok := c.documentHashByDocID[id]; !ok {
			return false
		}
	}
	return true
}

func (c *Class) AddAttendedStudentsID(id string) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	c.attendedByStudentsIds[id] = true
}
func (c *Class) IsAttendedByStudentsID(id string) bool {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	return c.attendedByStudentsIds[id]
}
func (c *Class) AttendedStudentsIDCount() int {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	return len(c.attendedByStudentsIds)
}

func (c *Class) AddAssignmentID(id string) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	c.assignmentID = id
}
func (c *Class) AssignmentID() string {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	return c.assignmentID
}
func (c *Class) AddSubmission(name string, hash []byte) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	c.submissionHashByName[name] = hash
}
func (c *Class) Submissions() map[string]hash {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	// Submissionのチェックは1クラス1回しか行われないのでhmapオブジェクト(ポインタ)をそのまま渡す
	return c.submissionHashByName
}
