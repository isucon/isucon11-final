package api

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/fails"
	"github.com/pborman/uuid"
	"net/http"
)

/*
  ユーザ関連のアクセスエンドポイント
  - GET /users/{user_id}/courses  // 履修済み講義一覧取得
  - PUT /users/{user_id}/courses  // 講義履修登録
  - GET /users/{user_id}/grades   // 成績一覧取得
*/

type DayOfWeek string

const (
	_ DayOfWeek = "sunday"
	_ DayOfWeek = "monday"
	_ DayOfWeek = "tuesday"
	_ DayOfWeek = "wednesday"
	_ DayOfWeek = "thursday"
	_ DayOfWeek = "friday"
	_ DayOfWeek = "saturday"
)

type GetRegisteredCourseResponseContent struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Teacher   string    `json:"teacher"`
	Period    uint8     `json:"period"`
	DayOfWeek DayOfWeek `json:"day_of_week"`
}

func GetRegisteredCourses(ctx context.Context, a *agent.Agent) (*http.Response, error) {
	path := "/api/users/me/courses"

	req, err := a.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	return a.Do(ctx, req)
}

type RegisterCourseRequestContent struct {
	ID string `json:"id"`
}

type RegisterCoursesErrorResponse struct {
	CourseNotFound       []string    `json:"course_not_found,omitempty"`
	NotRegistrableStatus []uuid.UUID `json:"not_registrable_status,omitempty"`
	ScheduleConflict     []uuid.UUID `json:"schedule_conflict,omitempty"`
}

func RegisterCourses(ctx context.Context, a *agent.Agent, courses []RegisterCourseRequestContent) (*http.Response, error) {
	body, err := json.Marshal(courses)
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}
	path := "/api/users/me/courses"

	req, err := a.NewRequest(http.MethodPost, path, bytes.NewReader(body))

	return a.Do(ctx, req)
}

type GetGradeResponse struct {
	Summary       Summary        `json:"summary"`
	CourseResults []CourseResult `json:"courses"`
}

type Summary struct {
	Credits int     `json:"credits"`
	GPT     int     `json:"gpt"`
	GptDev  float64 `json:"gpt_dev"` // 偏差値
	GptAvg  float64 `json:"gpt_avg"` // 平均値
	GptMax  int     `json:"gpt_max"` // 最大値
	GptMin  int     `json:"gpt_min"` // 最小値
}

type CourseResult struct {
	Name          string       `json:"name"`
	Code          string       `json:"code"`
	TotalScore    int          `json:"total_score"`
	TotalScoreDev float64      `json:"total_score_dev"` // 偏差値
	TotalScoreAvg float64      `json:"total_score_avg"` // 平均値
	TotalScoreMax int          `json:"total_score_max"` // 最大値
	TotalScoreMin int          `json:"total_score_min"` // 最小値
	ClassScores   []ClassScore `json:"class_scores"`
}

type ClassScore struct {
	ClassID    uuid.UUID `json:"class_id"`
	Title      string    `json:"title"`
	Part       uint8     `json:"part"`
	Score      int       `json:"score"`      // 0~100点
	Submitters int       `json:"submitters"` // 提出した生徒数
}

func GetGrades(ctx context.Context, a *agent.Agent) (*http.Response, error) {
	path := "/api/users/me/grades"

	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	return a.Do(ctx, req)
}
