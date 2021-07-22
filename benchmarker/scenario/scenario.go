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

const (
	RegisterCourseLimit = 20
	SearchCourseLimit   = 5
	InitialCourseCount  = 20
	CourseProcessLimit  = 5
)

type Scenario struct {
	BaseURL *url.URL
	UseTLS  bool
	NoLoad  bool

	sPubSub             *pubsub.PubSub
	cPubSub             *pubsub.PubSub
	courses             []*model.Course
	student             []*model.Student
	activeStudentCount  int // FIXME Debug
	finishedCourseCount int // FIXME Debug
	language            string

	mu sync.Mutex
}

func NewScenario() (*Scenario, error) {
	initialStudents := generate.InitialStudents()
	return &Scenario{
		sPubSub: pubsub.NewPubSub(),
		cPubSub: pubsub.NewPubSub(),
		courses: []*model.Course{},
		student: initialStudents,
	}, nil
}

func (s *Scenario) Validation(context.Context, *isucandar.BenchmarkStep) error {
	if s.NoLoad {
		return nil
	}
	ContestantLogger.Printf("===> VALIDATION")

	return nil
}

func (s *Scenario) Language() string {
	return s.language
}

func (s *Scenario) ActiveStudentCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.activeStudentCount
}

func (s *Scenario) CourseCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.courses)
}
func (s *Scenario) FinishedCourseCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.finishedCourseCount
}
