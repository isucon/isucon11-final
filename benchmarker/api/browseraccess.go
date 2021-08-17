package api

import (
	"context"
	"net/http"

	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/fails"
)

func BrowserAccess(ctx context.Context, a *agent.Agent, path string) (*http.Response, agent.Resources, error) {
	req, err := a.GET(path)
	if err != nil {
		return nil, nil, failure.NewError(fails.ErrCritical, err)
	}

	res, err := a.Do(ctx, req)
	if err != nil {
		return nil, nil, failure.NewError(fails.ErrCritical, err)
	}

	if res.StatusCode == http.StatusNotModified {
		return res, nil, nil
	}

	// HTMLファイルから追加リソースを参照する
	resources, err := a.ProcessHTML(ctx, res, res.Body)
	if err != nil {
		return nil, nil, failure.NewError(fails.ErrHTTP, err)
	}

	return res, resources, nil
}
