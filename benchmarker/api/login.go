package api

import (
	"context"
	"net/http"

	"github.com/isucon/isucandar/agent"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func Login(ctx context.Context, a *agent.Agent, req *LoginRequest) (*http.Response, error) {
	return nil, nil
}
