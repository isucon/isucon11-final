package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/fails"
)

type initializeResponse struct {
	Language string `json:"language"`
}

const initializeEndpoint = "/initialize"

func Initialize(ctx context.Context, a *agent.Agent) (string, error) {
	req, err := a.GET(initializeEndpoint)
	if err != nil {
		return "", failure.NewError(fails.ErrCritical, err)
	}

	res, err := a.Do(ctx, req)
	if err != nil {
		return "", failure.NewError(fails.ErrHTTP, err)
	}
	defer res.Body.Close()

	if err := assertStatusCode(res, http.StatusOK); err != nil {
		return "", nil
	}

	r := initializeResponse{}
	err = json.NewDecoder(res.Body).Decode(&r)
	if err != nil {
		return "", failure.NewError(fails.ErrHTTP, fmt.Errorf(
			"JSONのパースに失敗しました (%s: %s)", res.Request.Method, res.Request.URL.Path,
		))
	}
	return r.Language, nil
}
