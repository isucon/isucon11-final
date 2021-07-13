package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/fails"
	"net/http"

	"github.com/isucon/isucandar/agent"
	"github.com/pborman/uuid"
)

type AddAnnouncementRequest struct {
	CourseID uuid.UUID `json:"course_id"`
	Title    string    `json:"title"`
	Message  string    `json:"message"`
}

type AddAnnouncementResponse struct {
	ID uuid.UUID `json:"id"`
}

func AddAnnouncement(ctx context.Context, a *agent.Agent, announcement AddAnnouncementRequest) (*http.Response, error) {
	body, err := json.Marshal(announcement)
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}
	path := "/announcements"

	req, err := a.POST(path, bytes.NewReader(body))
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	return a.Do(ctx, req)
}

type AnnouncementsResponse struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Unread   bool   `json:"message"`
	CreateAt int64  `json:"created_at"`
}

func FetchAnnouncements(ctx context.Context, a *agent.Agent) ([]*AnnouncementsResponse, error) {
	rpath := "/api/announcements"
	var res []*AnnouncementsResponse
	_, err := apiRequest(ctx, a, http.MethodGet, rpath, nil, res, []int{http.StatusOK})
	if err != nil {
		return nil, err
	}
	// FIXME: ページングに対応 Issue:#91

	return res, nil
}

type AnnouncementDetailResponse struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Message  string `json:"message"`
	CreateAt int64  `json:"created_at"`
}

func FetchAnnouncementDetail(ctx context.Context, a *agent.Agent, id string) (*AnnouncementDetailResponse, error) {
	rpath := fmt.Sprintf("/api/announcements/%s", id)
	res := &AnnouncementDetailResponse{}
	_, err := apiRequest(ctx, a, http.MethodGet, rpath, nil, res, []int{http.StatusOK})
	if err != nil {
		return nil, err
	}

	return res, nil
}
