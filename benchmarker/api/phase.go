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

type phaseRequest struct {
	Phase string `json:"phase"` // FIXME: webappと調整
}

func ChangePhaseToRegister(ctx context.Context, a *agent.Agent) error {
	return changePhase(ctx, a, phaseReg)
}
func ChangePhaseToClasses(ctx context.Context, a *agent.Agent) error {
	return changePhase(ctx, a, phaseClass)
}
func ChangePhaseToResult(ctx context.Context, a *agent.Agent) error {
	return changePhase(ctx, a, phaseResult)
}

func changePhase(ctx context.Context, a *agent.Agent, phase string) error {
	req := &phaseRequest{Phase: phase}
	_, err := apiRequest(ctx, a, http.MethodPost, "/phase", req, nil, []int{http.StatusOK})
	if err != nil {
		return err
	}

	return nil
}
