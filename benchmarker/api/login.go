package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/isucon/isucandar/agent"

	"github.com/isucon/isucon11-final/benchmarker/fails"
)

type LoginRequest struct {
	Code     string `json:"code"`
	Password string `json:"password"`
}

func Login(ctx context.Context, a *agent.Agent, auth LoginRequest) (*http.Response, error) {
	body, err := json.Marshal(auth)
	if err != nil {
		return nil, fails.ErrorCritical(err)
	}
	path := "/login"

	req, err := a.POST(path, bytes.NewReader(body))
	if err != nil {
		return nil, fails.ErrorCritical(err)
	}

	req.Header.Set("Content-Type", "application/json")
	return a.Do(ctx, req)
}
