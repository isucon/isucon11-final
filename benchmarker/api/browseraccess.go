package api

import (
	"context"
	"net/http"

	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/fails"
)

func AccessLoginPage(ctx context.Context, a *agent.Agent) []error {
	return browserAccess(ctx, a, "/login")
}
func AccessSyllabusPage(ctx context.Context, a *agent.Agent, courseID string) []error {
	return browserAccess(ctx, a, "/syllabus?id="+courseID)
}

func browserAccess(ctx context.Context, a *agent.Agent, path string) []error {
	req, err := a.GET(path)
	if err != nil {
		return []error{failure.NewError(fails.ErrCritical, err)}
	}

	res, err := a.Do(ctx, req)
	if err != nil {
		return []error{failure.NewError(fails.ErrHTTP, err)}
	}
	defer res.Body.Close()

	if err := assertStatusCode(res, http.StatusOK); err != nil {
		if err := assertStatusCode(res, http.StatusNotModified); err != nil {
			return []error{err}
		}
	}

	// HTMLファイルから追加リソースを参照する
	resources, perr := a.ProcessHTML(ctx, res, res.Body)
	if perr != nil {
		return []error{failure.NewError(fails.ErrCritical, perr)}
	}

	var errs []error
	for _, resource := range resources {
		if resource.Error != nil {
			errs = append(errs, failure.NewError(fails.ErrHTTP, resource.Error))
			continue
		}

		if resource.Response.StatusCode == 304 {
			continue
		}

		if err := assertStatusCode(resource.Response, http.StatusOK); err != nil {
			errs = append(errs, err)
			continue
		}

		if err := assertChecksum(resource.Response); err != nil {
			errs = append(errs, err)
			continue
		}
	}
	if len(errs) > 0 {
		return errs
	}

	return nil
}
