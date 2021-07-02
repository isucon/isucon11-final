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

type announcementRegRequest struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}
type announcementRegResponse struct {
	ID string `json:"id"`
}

func AddAnnouncement(ctx context.Context, a *agent.Agent, courseID, title, message string) (string, error) {
	reqBody, err := json.Marshal(&announcementRegRequest{title, message})
	if err != nil {
		return "", failure.NewError(fails.ErrCritical, err)
	}
	req, err := a.POST(fmt.Sprintf("/api/courses/%s/announcements", courseID), bytes.NewBuffer(reqBody))
	if err != nil {
		return "", failure.NewError(fails.ErrCritical, err)
	}

	req.Header.Set("Content-Type", "application/json")
	res, err := a.Do(ctx, req)
	if err != nil {
		return "", failure.NewError(fails.ErrHTTP, err)
	}
	if err := assertStatusCode(res, http.StatusOK); err != nil {
		return "", err
	}

	var resp announcementRegResponse
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		return "", failure.NewError(fails.ErrHTTP, fmt.Errorf(
			"JSONのパースに失敗しました (%s: %s)", res.Request.Method, res.Request.URL.Path,
		))
	}

	return resp.ID, nil
}

type AnnouncementsResponse struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Unread   bool   `json:"message"`
	CreateAt int64  `json:"created_at"`
}

func FetchAnnouncements(ctx context.Context, a *agent.Agent) ([]*AnnouncementsResponse, error) {
	req, err := a.GET(fmt.Sprintf("/api/announcements"))
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	res, err := a.Do(ctx, req)
	if err != nil {
		return nil, failure.NewError(fails.ErrHTTP, err)
	}
	if err := assertStatusCode(res, http.StatusOK); err != nil {
		return nil, err
	}

	var resp []*AnnouncementsResponse
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		return nil, failure.NewError(fails.ErrHTTP, fmt.Errorf(
			"JSONのパースに失敗しました (%s: %s)", res.Request.Method, res.Request.URL.Path,
		))
	}

	return resp, nil
}

type AnnouncementDetailResponse struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Message  string `json:"message"`
	CreateAt int64  `json:"created_at"`
}

func FetchAnnouncementDetail(ctx context.Context, a *agent.Agent, id string) (*AnnouncementDetailResponse, error) {
	req, err := a.GET(fmt.Sprintf("/api/announcements/%s", id))
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	res, err := a.Do(ctx, req)
	if err != nil {
		return nil, failure.NewError(fails.ErrHTTP, err)
	}
	if err := assertStatusCode(res, http.StatusOK); err != nil {
		return nil, err
	}

	// FIXME: ページングに対応 Issue:#91
	var resp AnnouncementDetailResponse
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		return nil, failure.NewError(fails.ErrHTTP, fmt.Errorf(
			"JSONのパースに失敗しました (%s: %s)", res.Request.Method, res.Request.URL.Path,
		))
	}
	return &resp, nil
}
