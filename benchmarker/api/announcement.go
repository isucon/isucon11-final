package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/fails"
	"github.com/pborman/uuid"
)

type AddAnnouncementRequest struct {
	CourseID string `json:"course_id"`
	Title    string `json:"title"`
	Message  string `json:"message"`
}

type AddAnnouncementResponse struct {
	ID string `json:"id"`
}

func AddAnnouncement(ctx context.Context, a *agent.Agent, announcement AddAnnouncementRequest) (*http.Response, error) {
	body, err := json.Marshal(announcement)
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}
	path := "/api/announcements"

	req, err := a.POST(path, bytes.NewReader(body))
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	req.Header.Set("Content-Type", "application/json")
	return a.Do(ctx, req)
}

type AnnouncementResponse struct {
	ID         string `json:"id"`
	CourseID   string `json:"course_id"`
	CourseName string `json:"course_name"`
	Title      string `json:"title"`
	Message    string `json:"message"`
	Unread     bool   `json:"unread"`
	CreatedAt  int64  `json:"created_at"`
}
type GetAnnouncementsResponse struct {
	UnreadCount   int                    `json:"unread_count"`
	Announcements []AnnouncementResponse `json:"announcements"`
}

func GetAnnouncementList(ctx context.Context, a *agent.Agent, rawURL string, courseID uuid.UUID) (*http.Response, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, failure.NewError(fails.ErrHTTP, err)
	}
	q := u.Query()

	if courseID != nil {
		q.Set("course_id", courseID.String())
		u.RawQuery = q.Encode()
	}

	req, err := a.GET(u.String())
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	return a.Do(ctx, req)
}

func GetAnnouncementDetail(ctx context.Context, a *agent.Agent, id string) (*http.Response, error) {
	path := fmt.Sprintf("/api/announcements/%s", id)

	req, err := a.GET(path)
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	return a.Do(ctx, req)
}
