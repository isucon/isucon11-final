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
	reqBody, err := json.Marshal(&registerGradeRequest{
		UserID: userID,
		Grade:  grade,
	})
	if err != nil {
		return failure.NewError(fails.ErrCritical, err)
	}

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
	req, err := a.GET(fmt.Sprintf("/api/users/%s/grades", userID))
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}
	res, err := a.Do(ctx, req)
	if err != nil {
		return nil, failure.NewError(fails.ErrHTTP, err)
	}
	defer res.Body.Close()

	if err := assertStatusCode(res, http.StatusOK); err != nil {
		return nil, err
	}
	r := GetGradesResponse{}
	err = json.NewDecoder(res.Body).Decode(&r)
	if err != nil {
		return nil, failure.NewError(fails.ErrHTTP, fmt.Errorf(
			"JSONのパースに失敗しました (%s: %s)", res.Request.Method, res.Request.URL.Path,
		))
	}
	return &r, nil
}
