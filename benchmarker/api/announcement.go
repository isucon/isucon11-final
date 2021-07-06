package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/isucon/isucandar/agent"
)

type announcementRegRequest struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}
type announcementRegResponse struct {
	ID string `json:"id"`
}

func AddAnnouncement(ctx context.Context, a *agent.Agent, courseID, title, message string) (string, error) {
	rpath := fmt.Sprintf("/api/courses/%s/announcements", courseID)
	req := &announcementRegRequest{title, message}
	var res announcementRegResponse
	_, err := apiRequest(ctx, a, http.MethodPost, rpath, req, res, []int{http.StatusOK})
	if err != nil {
		return "", err
	}

	return res.ID, nil
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
