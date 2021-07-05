package api

import (
	"context"
	"net/http"

	"github.com/isucon/isucandar/agent"
)

type postAttendanceRequest struct {
	Code string `json:"code"`
}

func PostAttendance(ctx context.Context, a *agent.Agent, code string) error {
	rpath := "/api/attendance_codes"
	req := &postAttendanceRequest{code}
	_, err := apiRequest(ctx, a, http.MethodPost, rpath, req, nil, []int{http.StatusOK})
	if err != nil {
		return err
	}

	return nil
}
