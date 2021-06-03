package scenario

import (
	"context"
	"sync"

	"github.com/isucon/isucandar"
)

type Scenario struct {
	mu       sync.RWMutex
	BaseURL  string
	UseTLS   bool
	Language string
	NoLoad   bool
}

func NewScenario() (*Scenario, error) {
	return &Scenario{}, nil
}

func (s *Scenario) Prepare(context.Context, *isucandar.BenchmarkStep) error {
	ContestantLogger.Printf("===> PREPARE")

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
