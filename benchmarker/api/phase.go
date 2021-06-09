package api

import (
	"context"
	"net/http"

	"github.com/isucon/isucandar/agent"
)

const (
	phaseReg    = "reg"
	phaseClass  = "class"
	phaseResult = "result"
)

type phaseRequest struct{}

func ChangePhaseToRegister(ctx context.Context, a *agent.Agent) (int64, error) {
	return changePhase(ctx, a, phaseReg)
}
func ChangePhaseToClasses(ctx context.Context, a *agent.Agent) (int64, error) {
	return changePhase(ctx, a, phaseClass)
}
func ChangePhaseToResult(ctx context.Context, a *agent.Agent) (int64, error) {
	return changePhase(ctx, a, phaseResult)
}

func changePhase(ctx context.Context, a *agent.Agent, phase string) (int64, error) {
	return http.StatusOK, nil
}
