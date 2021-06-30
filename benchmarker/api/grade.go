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

	_, err := ApiRequest(ctx, a, http.MethodPost, fmt.Sprintf("/api/courses/%s/grades", courseID), req, nil, []int{http.StatusOK})
	if err != nil {
		return err
	}

	return nil
}

type GetGradesResponse struct {
	Summary      Summary        `json:"summary"`
	CourseGrades []*CourseGrade `json:"courses"`
}

type Summary struct {
	Credits uint8   `json:"credits"`
	GPA     float64 `json:"gpa"`
}

type CourseGrade struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Credit uint8  `json:"credit"`
	Grade  uint32 `json:"grade"`
}

func GetGrades(ctx context.Context, a *agent.Agent, userID string) (*GetGradesResponse, error) {
	r := &GetGradesResponse{}
	_, err := ApiRequest(ctx, a, http.MethodGet, fmt.Sprintf("/api/users/%s/grades", userID), nil, r, []int{http.StatusOK})
	if err != nil {
		return nil, err
	}

	return r, nil
}
