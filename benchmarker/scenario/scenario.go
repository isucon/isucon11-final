package scenario

import (
	"context"

	"github.com/isucon/isucandar"
)

type Scenario struct {
	BaseURL string
	UseTLS  bool
	NoLoad  bool

	language string
}

func NewScenario() (*Scenario, error) {
	return &Scenario{}, nil
}

func (s *Scenario) Prepare(context.Context, *isucandar.BenchmarkStep) error {
	ContestantLogger.Printf("===> PREPARE")

	s.language = "" // TODO: set from /initialize

	return nil
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

func (s *Scenario) Language() string {
	return s.language
}
