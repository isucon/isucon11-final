package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/isucon/isucandar/agent"
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
	var registeredCourses []usersCourseResponse
	_, err := ApiRequest(ctx, a, http.MethodGet, fmt.Sprintf("/api/users/%s/courses", userID), nil, &registeredCourses, []int{http.StatusOK})
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, c := range registeredCourses {
		ids = append(ids, c.ID)
	}
	return ids, nil
}

func RegisterCourses(ctx context.Context, a *agent.Agent, userID string, courses []string) ([]string, error) {
	res, err := ApiRequest(ctx, a, http.MethodPost, fmt.Sprintf("/api/users/%s/courses", userID), courses, nil, []int{http.StatusOK, http.StatusBadRequest})
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
