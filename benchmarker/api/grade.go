package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/fails"
)

type RegisterScoresRequestContent struct {
	UserCode string `json:"user_code"`
	Score    int    `json:"score"`
}

func RegisterScores(ctx context.Context, a *agent.Agent, courseID, classID string, scores []RegisterScoresRequestContent) (*http.Response, error) {
	body, err := json.Marshal(scores)
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}
	path := fmt.Sprintf("/api/courses/%s/classes/%s/assignments", courseID, classID)

	req, err := a.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	return a.Do(ctx, req)
}
