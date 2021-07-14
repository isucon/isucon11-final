package scenario

import (
	"context"
	"net/url"
	"sync"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucandar/pubsub"
	"github.com/isucon/isucon11-final/benchmarker/generate"
	"github.com/isucon/isucon11-final/benchmarker/model"
)

var (
	// Prepare, Load, Validationが返すエラー
	// Benchmarkが中断されたかどうか確認用
	Cancel failure.StringCode = "scenario-cancel"
)

type Scenario struct {
	BaseURL *url.URL
	UseTLS  bool
	NoLoad  bool

	sPubSub         *pubsub.PubSub
	cPubSub         *pubsub.PubSub
	courses         []*model.Course
	inactiveStudent []*model.Student
	activeStudent   []*model.Student
	language        string

	mu sync.Mutex
}

func NewScenario() (*Scenario, error) {
	initialStudents := generate.InitialStudents()
	return &Scenario{
		sPubSub:         pubsub.NewPubSub(),
		cPubSub:         pubsub.NewPubSub(),
		courses:         []*model.Course{},
		inactiveStudent: initialStudents,
		activeStudent:   make([]*model.Student, 0, len(initialStudents)),
	}, nil
}

func (s *Scenario) Load(parent context.Context, _ *isucandar.BenchmarkStep) error {
	if s.NoLoad {
		return nil
	}
	_, cancel := context.WithCancel(parent)
	defer cancel()

	ContestantLogger.Printf("===> LOAD")
	AdminLogger.Printf("LOAD INFO\n  No load action")

	return nil
}
func (s *Scenario) Validation(context.Context, *isucandar.BenchmarkStep) error {
	if s.NoLoad {
		return nil
	}
	ContestantLogger.Printf("===> VALIDATION")

	return nil
}

func (s *Scenario) activateStudent() *model.Student {
	s.mu.Lock()
	defer s.mu.Unlock()

	// FIXME: ちゃんとしたやつにしたいけど優先度低
	if len(s.inactiveStudent) == 0 {
		return nil
	}

	activatedStudent := s.inactiveStudent[0]
	s.activeStudent = append(s.activeStudent, activatedStudent)
	s.inactiveStudent = s.inactiveStudent[1:]
	return activatedStudent
}

func (s *Scenario) Language() string {
	return s.language
}
