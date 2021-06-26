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

/*
  ユーザ関連のアクセスエンドポイント
  - GET /users/{user_id}/courses  // 履修済み講義一覧取得
  - PUT /users/{user_id}/courses  // 講義履修登録
  - GET /users/{user_id}/grades   // 成績一覧取得
*/

type usersCourseResponse struct {
	ID string `json:"id"`
}

func FetchRegisteredCourses(ctx context.Context, a *agent.Agent, userID string) ([]string, error) {
	req, err := a.GET(fmt.Sprintf("/api/users/%s/courses", userID))
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := a.Do(ctx, req)
	if err != nil {
		return nil, failure.NewError(fails.ErrHTTP, err)
	}
	defer res.Body.Close()

	var registeredCourses []usersCourseResponse
	err = json.NewDecoder(res.Body).Decode(&registeredCourses)
	if err != nil {
		return nil, failure.NewError(fails.ErrHTTP, fmt.Errorf(
			"JSONのパースに失敗しました (%s: %s)", res.Request.Method, res.Request.URL.Path,
		))
	}

	var ids []string
	for _, c := range registeredCourses {
		ids = append(ids, c.ID)
	}
	return ids, nil
}

func RegisterCourses(ctx context.Context, a *agent.Agent, userID string, courses []string) ([]string, error) {
	reqBody, err := json.Marshal(courses)
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}
	req, err := a.POST(fmt.Sprintf("/api/users/%s/courses", userID), bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := a.Do(ctx, req)
	if err != nil {
		return nil, failure.NewError(fails.ErrHTTP, err)
	}
	defer res.Body.Close()

	if err := assertStatusCode(res, http.StatusBadRequest); err == nil {
		// 400エラー = 定員オーバーによる登録失敗。 FIXME: 他の400エラーと区別するためにエラーレスポンスを解析が必要っぽい
		// 登録成功したコースは0(空), 準正常系なのでエラーなし
		return []string{}, nil
	}

	if err := assertStatusCode(res, http.StatusOK); err != nil {
		return nil, err
	}
	// 登録に成功したコースを返す
	return courses, err
}
