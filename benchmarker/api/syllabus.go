package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/isucon/isucandar/agent"
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
	res := syllabusSearchResponse{}
	_, err := apiRequest(ctx, a, http.MethodGet, fmt.Sprintf("/api/syllabus?keyword=%s", keyword), nil, &res, []int{http.StatusOK})
	if err != nil {
		return nil, err
	}

	syllabusIDs := make([]string, len(res))
	for _, s := range res {
		syllabusIDs = append(syllabusIDs, s.ID)
	}
	return syllabusIDs, nil
}
