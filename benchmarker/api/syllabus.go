package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/fails"
	"github.com/pborman/uuid"
)

type SearchCourseRequest struct {
	Type      CourseType
	Credit    uint8
	Teacher   string
	Period    uint8
	DayOfWeek DayOfWeek
	Keywords  string
}
type GetCourseDetailResponse struct {
	ID          uuid.UUID  `json:"id"`
	Code        string     `json:"code"`
	Type        CourseType `json:"type"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Credit      uint8      `json:"credit"`
	Period      uint8      `json:"period"`
	DayOfWeek   DayOfWeek  `json:"day_of_week"`
	Teacher     string     `json:"teacher"`
	Keywords    string     `json:"keywords"`
}

func SearchCourse(ctx context.Context, a *agent.Agent, param *SearchCourseRequest) (*http.Response, error) {
	req, err := a.GET("/api/syllabus")
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}
	query := req.URL.Query()
	query.Add("type", string(param.Type))
	query.Add("credit", string(param.Credit))
	query.Add("teacher", param.Teacher)
	query.Add("period", string(param.Period))
	query.Add("day_of_week", string(param.DayOfWeek))
	query.Add("keywords", param.Keywords)
	req.URL.RawQuery = query.Encode()

	req.Header.Set("Content-Type", "application/json")
	return a.Do(ctx, req)
}

func GetCourseDetail(ctx context.Context, a *agent.Agent, courseID uuid.UUID) (*http.Response, error) {
	path := fmt.Sprintf("/api/syllabus/%s", courseID)

	req, err := a.GET(path)
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	req.Header.Set("Content-Type", "application/json")
	return a.Do(ctx, req)
}
