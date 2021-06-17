package api

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/fails"
)

func assertStatusCode(res *http.Response, expectCode int) error {
	if res.StatusCode != expectCode {
		// 以降利用しないres.Bodyはコネクション維持のために読み捨てる
		_, _ = io.Copy(ioutil.Discard, res.Body)
		return failure.NewError(fails.ErrHTTP, fmt.Errorf(
			"期待する HTTP ステータスコード以外が返却されました: %d (%s: %s)",
			res.StatusCode, res.Request.Method, res.Request.URL.Path,
		))
	}
	return nil
}

func assertChecksum(_ *http.Response) error {
	return nil
}
