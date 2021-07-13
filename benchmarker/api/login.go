package api

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/fails"
	"net/http"

	"github.com/isucon/isucandar/agent"
)

type LoginRequest struct {
	Code     string `json:"code"`
	Password string `json:"password"`
}

func Login(ctx context.Context, a *agent.Agent, auth LoginRequest) (*http.Response, error) {
	body, err := json.Marshal(auth)
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}
	path := "/login"

	req, err := a.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	return a.Do(ctx, req)
}
