package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/isucon/isucandar/agent"
)

type registerGradeRequest struct {
	UserID string `json:"user_id"`
	Grade  uint32 `json:"grade"`
}

func RegisterGrades(ctx context.Context, a *agent.Agent, courseID, userID string, grade uint32) error {
	req := &registerGradeRequest{
		UserID: userID,
		Grade:  grade,
	}

	_, err := apiRequest(ctx, a, http.MethodPost, fmt.Sprintf("/api/courses/%s/grades", courseID), req, nil, []int{http.StatusOK})
	if err != nil {
		return err
	}

	return nil
}
