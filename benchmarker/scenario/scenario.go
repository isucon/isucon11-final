package scenario

import (
	"context"
	"net/url"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/failure"
)

var (
	// Prepare, Load, Validationが返すエラー
	// Benchmarkが中断されたかどうか確認用
	Cancel failure.StringCode = "scenario-cancel"

	// BenchmarkStepにAddErrorしていくエラー郡
	// 最終的にエラースコア計算に利用
	ErrCritical          failure.StringCode = "critical" // スコア0点
	ErrInvalidStatusCode failure.StringCode = "invalid-status-code"
	ErrInvalidResponse   failure.StringCode = "invalid-response"
)

type Scenario struct {
	BaseURL *url.URL
	UseTLS  bool
	NoLoad  bool

	language string
}

func NewScenario() (*Scenario, error) {
	return &Scenario{}, nil
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
