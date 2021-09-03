package scenario

import (
	"fmt"
	"math/rand"
	"net/url"
	"sync"

	"github.com/isucon/isucon11-final/benchmarker/model"
)

type userPool struct {
	studentAccounts  []*model.UserAccount
	teachers         []*model.Teacher
	index            int
	useSampleStudent bool // sampleStudentを排出したかどうかのフラグ
	teacherCount     int  // sampleTeacherの排出に使うカウント
	sampleTeacher    *model.Teacher
	baseURL          *url.URL

	rmu sync.RWMutex
}

var (
	sampleStudentID   = "S00000"
	sampleStudentName = "isucon(学生)"
	sampleStudentPass = "isucon"

	sampleTeacherID   = "T00000"
	sampleTeacherName = "isucon(教員)"
	sampleTeacherPass = "isucon"
)

func NewUserPool(studentAccounts []*model.UserAccount, teacherAccounts []*model.UserAccount, baseURL *url.URL) *userPool {
	// shuffle studentDataSet order by Fisher–Yates shuffle
	for i := len(studentAccounts) - 1; i >= 0; i-- {
		j := rand.Intn(i + 1)
		studentAccounts[i], studentAccounts[j] = studentAccounts[j], studentAccounts[i]
	}

	sampleTeacher := model.NewTeacher(&model.UserAccount{
		Code:        sampleTeacherID,
		Name:        sampleTeacherName,
		RawPassword: sampleTeacherPass,
	}, baseURL)

	teachers := make([]*model.Teacher, len(teacherAccounts))
	for i, account := range teacherAccounts {
		teachers[i] = model.NewTeacher(account, baseURL)
	}

	return &userPool{
		studentAccounts: studentAccounts,
		teachers:        teachers,
		index:           0,
		sampleTeacher:   sampleTeacher,
		baseURL:         baseURL,
		rmu:             sync.RWMutex{},
	}
}

func (p *userPool) newStudent() (*model.Student, error) {
	p.rmu.Lock()
	defer p.rmu.Unlock()

	if !p.useSampleStudent {
		p.useSampleStudent = true
		return model.NewStudent(&model.UserAccount{
			Code:        sampleStudentID,
			Name:        sampleStudentName,
			RawPassword: sampleStudentPass,
		}, p.baseURL), nil
	}

	if p.index >= len(p.studentAccounts) {
		return nil, fmt.Errorf("student data has been out of stock")
	}
	student := model.NewStudent(p.studentAccounts[p.index], p.baseURL)
	p.index++
	return student, nil
}

func (p *userPool) randomTeacher() *model.Teacher {
	p.rmu.Lock()
	defer p.rmu.Unlock()

	// 定期的にsampleTeacherを使う
	if p.teacherCount%20 == 0 {
		p.teacherCount++
		return p.sampleTeacher
	}

	p.teacherCount++
	return p.teachers[rand.Intn(len(p.teachers))]
}
