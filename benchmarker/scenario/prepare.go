package scenario

import (
	"context"

	"github.com/isucon/isucandar"
)

func (s *Scenario) Prepare(ctx context.Context, step *isucandar.BenchmarkStep) error {
	ContestantLogger.Printf("===> PREPARE")

	AdminLogger.Printf("Not prepare Prepare")

	return nil
}
