package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/fails"
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
	path := "/api/announcements"

	req, err := a.POST(path, bytes.NewReader(body))
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	return a.Do(ctx, req)
}

type Announcement struct {
	ID         uuid.UUID `json:"id"`
	CourseID   uuid.UUID `json:"course_id"`
	CourseName string    `json:"course_name"`
	Title      string    `json:"title"`
	Message    string    `json:"message"`
	Unread     bool      `json:"unread"`
	CreatedAt  time.Time `json:"created_at"`
}

func GetAnnouncementList(ctx context.Context, a *agent.Agent) (*http.Response, error) {
	path := "/api/announcements"

	req, err := a.GET(path)
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	return a.Do(ctx, req)
}
