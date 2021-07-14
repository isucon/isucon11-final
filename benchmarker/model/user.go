package model

import (
	"sync"

	"github.com/isucon/isucandar/agent"
)

type UserAccount struct {
	ID          string
	RawPassword string
}

type Student struct {
	*UserAccount
	Agent *agent.Agent

	rmu sync.RWMutex
}

type Faculty struct {
	*UserAccount
	Agent *agent.Agent
}

func NewFaculty(id, rawPW string) *Faculty {
	a, _ := agent.NewAgent()
	return &Faculty{
		UserAccount: &UserAccount{
			ID:          id,
			RawPassword: rawPW,
		},
		Agent: a,
	}
}

func NewStudent(id, rawPW string) *Student {
	a, _ := agent.NewAgent()
	return &Student{
		UserAccount: &UserAccount{
			ID:          id,
			RawPassword: rawPW,
		},
		Agent: a,
		rmu:   sync.RWMutex{},
	}
}
