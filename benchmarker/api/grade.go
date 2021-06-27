package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/fails"

	"github.com/isucon/isucandar/agent"
)

type registerGradeRequest struct {
	UserID string `json:"user_id"`
	Grade  uint32 `json:"grade"`
}

func RegisterGrades(ctx context.Context, a *agent.Agent, courseID, userID string, grade uint32) error {
	// MEMO: エラー無視していいのか
	reqBody, _ := json.Marshal(&registerGradeRequest{
		UserID: userID,
		Grade:  grade,
	})
	req, err := a.POST(fmt.Sprintf("/api/courses/%s/grades", courseID), bytes.NewBuffer(reqBody))
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
		return err
	}
	return nil
}
