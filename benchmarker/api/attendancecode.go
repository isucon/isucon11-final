package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/fails"
)

type postAttendanceRequest struct {
	Code string `json:"code"`
}

func PostAttendance(ctx context.Context, a *agent.Agent, code string) error {
	reqBody, err := json.Marshal(&postAttendanceRequest{
		Code: code,
	})
	if err != nil {
		return failure.NewError(fails.ErrCritical, err)
	}
	req, err := a.POST("/api/attendance_codes", bytes.NewBuffer(reqBody))
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
