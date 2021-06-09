package scenario

import (
	"context"
	"fmt"
	"net/http"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/model"

	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucon11-final/benchmarker/api"
)

const (
	ScoreLogin = "login"
)

func errorInvalidStatusCode(res *http.Response) error {
	return failure.NewError(ErrInvalidStatusCode, fmt.Errorf("期待する HTTP ステータスコード以外が返却されました: %d (%s: %s)", res.StatusCode, res.Request.Method, res.Request.URL.Path))
}

func errorInvalidResponse(message string, args ...interface{}) error {
	return failure.NewError(ErrInvalidResponse, fmt.Errorf(message, args...))
}

func InitializeAction(ctx context.Context, a *agent.Agent, step *isucandar.BenchmarkStep) []error {
	res, hres, err := api.Initialize(ctx, a)
	if err != nil {
		return []error{failure.NewError(ErrCritical, err)}
	}

	var errors []error
	if hres.StatusCode != 200 {
		errors = append(errors, errorInvalidStatusCode(hres))
	}
	if res.Language == "" {
		errors = append(errors, errorInvalidResponse("利用言語(language)が設定されていません"))
	}

	step.AddScore("initialize")
	return errors
}

func LoginAction(ctx context.Context, u *model.User, step *isucandar.BenchmarkStep) error {
	r := &api.LoginRequest{
		Username: u.Name,
		Password: u.RawPassword,
	}
	hres, err := api.Login(ctx, u.Agent, r)
	if err != nil {
		// ログイン施行自体のアクションが失敗
		return failure.NewError(ErrCritical, err)
	}

	if hres.StatusCode != 200 && hres.StatusCode != 403 {
		// ログイン施行自体は成功している
		step.AddError(errorInvalidStatusCode(hres))
	}

	step.AddScore(ScoreLogin)
	return nil
}
