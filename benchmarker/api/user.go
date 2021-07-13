package api

import (
	"context"
	"fmt"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/fails"
	"github.com/pborman/uuid"
	"net/http"

	"github.com/isucon/isucandar/agent"
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

type registerCourseRequest struct {
	ID string `json:"id"`
}

type registerCoursesRequest []registerCourseRequest

func RegisterCourses(ctx context.Context, a *agent.Agent, courses []string) ([]string, error) {
	req := make(registerCoursesRequest, 0, len(courses))
	for _, v := range courses {
		req = append(req, registerCourseRequest{ID: v})
	}
	res, err := apiRequest(ctx, a, http.MethodPost, fmt.Sprintf("/api/users/me/courses"), req, nil, []int{http.StatusOK, http.StatusBadRequest})
	if err != nil {
		return nil, err
	}

	if res.StatusCode == http.StatusBadRequest {
		// 400エラー = 定員オーバーによる登録失敗。 FIXME: 他の400エラーと区別するためにエラーレスポンスを解析が必要っぽい
		// 登録成功したコースは0(空), 準正常系なのでエラーなし
		return []string{}, nil
	}

	// 登録に成功したコースを返す
	return courses, err
}

type GetGradesResponse struct {
	Summary      Summary        `json:"summary"`
	CourseGrades []*CourseGrade `json:"courses"`
}

type Summary struct {
	Credits uint8   `json:"credits"`
	GPA     float64 `json:"gpa"`
}

type CourseGrade struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Credit uint8  `json:"credit"`
	Grade  uint32 `json:"grade"`
}

// FIXME: レスポンスの型などはまだ修正していない
func GetGrades(ctx context.Context, a *agent.Agent) (*GetGradesResponse, error) {
	r := &GetGradesResponse{}
	_, err := apiRequest(ctx, a, http.MethodGet, fmt.Sprintf("/api/users/me/grades"), nil, r, []int{http.StatusOK})
	if err != nil {
		return nil, err
	}

	return r, nil
}
