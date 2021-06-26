package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/fails"
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
	reqObj := &phaseRequest{Phase: phase}
	reqBody, err := json.Marshal(reqObj)
	if err != nil {
		return failure.NewError(fails.ErrCritical, err)
	}
	req, err := a.POST("/phase", bytes.NewBuffer(reqBody))
	if err != nil {
		return failure.NewError(fails.ErrCritical, err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := a.Do(ctx, req)
	if err != nil {
		return failure.NewError(fails.ErrHTTP, err)
	}
	defer res.Body.Close()

	if err := assertStatusCode(res, http.StatusOK); err != nil {
		return nil
	}
	return nil
}
