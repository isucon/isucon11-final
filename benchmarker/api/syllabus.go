package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/fails"
)

type syllabusSearchResponse []syllabusData
type syllabusData struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Classroom   string `json:"classroom"`
	Limit       int    `json:"limit"`
	Credit      int    `json:"credit"`
	Instructor  string `json:"instructor"`
	Timeslots   []struct {
		DayOfWeek int `json:"day_of_week"`
		ClassHour int `json:"class_hour"`
	} `json:"timeslots"`
}

func SearchSyllabus(ctx context.Context, a *agent.Agent, keyword string) ([]string, error) {
	req, err := a.GET(fmt.Sprintf("/api/syllabus?keyword=%s", keyword))
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}
	res, err := a.Do(ctx, req)
	if err != nil {
		return nil, failure.NewError(fails.ErrHTTP, err)
	}
	defer res.Body.Close()

	if err := assertStatusCode(res, http.StatusOK); err != nil {
		return nil, err
	}
	r := syllabusSearchResponse{}
	err = json.NewDecoder(res.Body).Decode(&r)
	if err != nil {
		return nil, failure.NewError(fails.ErrHTTP, fmt.Errorf(
			"JSONのパースに失敗しました (%s: %s)", res.Request.Method, res.Request.URL.Path,
		))
	}
	syllabusIDs := make([]string, len(r))
	for _, s := range r {
		syllabusIDs = append(syllabusIDs, s.ID)
	}
	return syllabusIDs, nil
}
