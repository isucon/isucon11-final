package api

import (
	"context"
	"net/http"

	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"

	"github.com/isucon/isucon11-final/benchmarker/fails"
)

type InitializeResponse struct {
	Language string `json:"language"`
}

func Initialize(ctx context.Context, a *agent.Agent) (*http.Response, error) {
	path := "/initialize"

	req, err := a.POST(path, nil)
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	req.Header.Set("Content-Type", "application/json")
	return a.Do(ctx, req)
}
