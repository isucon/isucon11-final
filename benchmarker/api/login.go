package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/fails"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func Login(ctx context.Context, a *agent.Agent, id, pw string) error {
	reqBody, err := json.Marshal(&loginRequest{
		Username: id,
		Password: pw,
	})
	if err != nil {
		return failure.NewError(fails.ErrCritical, err)
	}
	req, err := a.POST("/login", bytes.NewBuffer(reqBody))
	if err != nil {
		// リクエスト生成に失敗はほぼありえないのでCritical
		return failure.NewError(fails.ErrCritical, err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := a.Do(ctx, req)
	if err != nil {
		// net.Error.Timeoutなども雑にErrHTTPでwrapしちゃう
		// 呼び出し側でいい感じに優先度をつけてエラー判定するので
		return failure.NewError(fails.ErrHTTP, err)
	}
	defer res.Body.Close()

	if err := assertStatusCode(res, http.StatusOK); err != nil {
		return err
	}
	return nil
}

func LoginFail(ctx context.Context, a *agent.Agent, id, pw string) error {
	reqBody, err := json.Marshal(&loginRequest{
		Username: id,
		Password: pw,
	})
	if err != nil {
		return failure.NewError(fails.ErrCritical, err)
	}
	req, err := a.POST("/login", bytes.NewBuffer(reqBody))
	if err != nil {
		return failure.NewError(fails.ErrCritical, err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := a.Do(ctx, req)
	if err != nil {
		return failure.NewError(fails.ErrHTTP, err)
	}
	defer res.Body.Close()

	if err := assertStatusCode(res, http.StatusUnauthorized); err != nil {
		return err
	}
	return nil
}
