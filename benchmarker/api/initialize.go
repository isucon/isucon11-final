package api

import (
	"context"
	"net/http"

	"github.com/isucon/isucandar/agent"
)

type initializeRequest struct{}
type InitializeResponse struct {
	Language string `json:"language"`
}

func Initialize(ctx context.Context, a *agent.Agent) (*InitializeResponse, *http.Response, error) {
	return &InitializeResponse{}, nil, nil
}
