package scenario

import (
	"fmt"
	"math/rand"
	"sync"

	"github.com/isucon/isucon11-final/benchmarker/model"
)

type userPool struct {
	dataset []*model.UserAccount
	index   int
	done    bool // sampleUserを排出したかどうかのフラグ

	rmu sync.RWMutex
}

var (
	sampleTeacherID   = "isuT"
	sampleTeacherName = "isucon"
	sampleTeacherPass = "isucon"

	sampleStudentID   = "isucon"
	sampleStudentName = "isucon"
	sampleStudentPass = "isucon"
)

func NewUserPool(dataSet []*model.UserAccount) *userPool {
	// shuffle studentDataSet order by Fisher–Yates shuffle
	for i := len(dataSet) - 1; i >= 0; i-- {
		j := rand.Intn(i + 1)
		dataSet[i], dataSet[j] = dataSet[j], dataSet[i]
	}

	return &userPool{
		dataset: dataSet,
		index:   0,
		rmu:     sync.RWMutex{},
	}
}

func (p *userPool) newUserData() (*model.UserAccount, error) {
	p.rmu.Lock()
	defer p.rmu.Unlock()

	if !p.done {
		p.done = true
		return &model.UserAccount{
			Code:        sampleStudentID,
			Name:        sampleStudentName,
			RawPassword: sampleStudentPass,
		}, nil
	}

	if p.index >= len(p.dataset) {
		return nil, fmt.Errorf("student data has been out of stock")
	}
	d := *p.dataset[p.index]
	p.index++
	return &d, nil
}

func (p *userPool) reset() {
	p.rmu.Lock()
	defer p.rmu.Unlock()

	p.index = 0
}
