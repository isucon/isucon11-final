package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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

	return a.Do(ctx, req)
}

type Announcement struct {
	ID         string    `json:"id"`
	CourseID   string    `json:"course_id"`
	CourseName string    `json:"course_name"`
	Title      string    `json:"title"`
	Message    string    `json:"message"`
	Unread     bool      `json:"unread"`
	CreatedAt  time.Time `json:"created_at"`
}
type AnnouncementList []*Announcement

func GetAnnouncementList(ctx context.Context, a *agent.Agent, page int, courseID uuid.UUID) (*http.Response, error) {
	path := fmt.Sprintf("/api/announcements?page=%v", page)
	if courseID != nil {
		path += fmt.Sprintf("&course_id=%v", courseID)
	}

	req, err := a.GET(path)
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	return a.Do(ctx, req)
}
