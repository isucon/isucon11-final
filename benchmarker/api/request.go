package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/isucon/isucon11-final/benchmarker/fails"

	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"
)

func apiRequest(ctx context.Context, a *agent.Agent, method string, rpath string, req, res interface{}, allowedStatuCodes []int) (*http.Response, error) {
	var body io.Reader = nil
	if req != nil {
		reqBody, err := json.Marshal(req)
		if err != nil {
			return nil, failure.NewError(fails.ErrCritical, err)
		}
		body = bytes.NewReader(reqBody)
	}

	httpreq, err := a.NewRequest(method, rpath, body)
	if err != nil {
		return nil, failure.NewError(fails.ErrHTTP, err)
	}

	httpreq.Header.Set("Content-Type", "application/json")

	httpres, err := a.Do(ctx, httpreq)
	if err != nil {
		return nil, failure.NewError(fails.ErrHTTP, err)
	}
	defer httpres.Body.Close()

	invalidStatusCode := true
	for _, statusCode := range allowedStatuCodes {
		if httpres.StatusCode == statusCode {
			invalidStatusCode = false
		}
	}
	if invalidStatusCode {
		// 以降利用しないres.Bodyはコネクション維持のために読み捨てる
		_, _ = io.Copy(ioutil.Discard, httpres.Body)
		return nil, failure.NewError(fails.ErrHTTP, fmt.Errorf("不正な HTTP ステータスコード: %d (%s: %s)", httpres.StatusCode, httpreq.Method, httpreq.URL.Path))
	}

	err = json.NewDecoder(httpres.Body).Decode(res)
	if err != nil {
		return nil, failure.NewError(fails.ErrHTTP, fmt.Errorf(
			"JSONのパースに失敗しました (%s: %s)", httpres.Request.Method, httpres.Request.URL.Path,
		))
	}

	return httpres, nil
}
