package api

import (
	"context"
	"net/http"

	"github.com/isucon/isucandar/agent"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func Login(ctx context.Context, a *agent.Agent, id, pw string) error {
	req := &loginRequest{
		Username: id,
		Password: pw,
	}
	_, err := apiRequest(ctx, a, http.MethodPost, "/login", req, nil, []int{http.StatusOK})
	if err != nil {
		return err
	}

	return nil
}

func LoginFail(ctx context.Context, a *agent.Agent, id, pw string) error {
	req := &loginRequest{
		Username: id,
		Password: pw,
	}

	_, err := apiRequest(ctx, a, http.MethodPost, "/login", req, nil, []int{http.StatusUnauthorized})
	if err != nil {
		return err
	}

	return nil
}
