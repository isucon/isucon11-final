package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

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
	if param.Type != "" {
		query.Add("type", string(param.Type))
	}
	if param.Credit != 0 {
		query.Add("credit", strconv.Itoa((int(param.Credit))))
	}
	if param.Teacher != "" {
		query.Add("teacher", param.Teacher)
	}
	if param.Period != 0 {
		query.Add("period", strconv.Itoa((int(param.Period))))
	}
	if param.DayOfWeek != "" {
		query.Add("day_of_week", string(param.DayOfWeek))
	}
	if param.Keywords != "" {
		query.Add("keywords", param.Keywords)
	}
	req.URL.RawQuery = query.Encode()

	req.Header.Set("Content-Type", "application/json")
	return a.Do(ctx, req)
}

func GetCourseDetail(ctx context.Context, a *agent.Agent, courseID string) (*http.Response, error) {
	path := fmt.Sprintf("/api/syllabus/%s", courseID)

	req, err := a.GET(path)
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	req.Header.Set("Content-Type", "application/json")
	return a.Do(ctx, req)
}
