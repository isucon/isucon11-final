package fails

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/isucon/isucandar/failure"
)

// BenchmarkStepにAddErrorしていくエラー郡
const (
	// ErrCritical は即失格となるエラー。
	ErrCritical failure.StringCode = "critical-error"
	// ErrApplication は正しいアプリケーションの挙動と異なるときのエラー。ある程度許容される。
	ErrApplication failure.StringCode = "application-error"
	// ErrHTTP はアプリケーションへの接続周りでのエラー。ある程度許容される。
	ErrHTTP           failure.StringCode = "http-error"
	ErrInvalidStatus  failure.StringCode = "invalid status code"
	ErrStaticResource failure.StringCode = "invalid resource"
)

func IsCritical(err error) bool {
	return failure.IsCode(err, ErrCritical)
}

func IsDeduction(err error) bool {
	return failure.IsCode(err, ErrApplication) ||
		failure.IsCode(err, ErrHTTP) ||
		failure.IsCode(err, ErrInvalidStatus)
}

func IsTimeout(err error) bool {
	var nerr net.Error
	if failure.As(err, &nerr) {
		if nerr.Timeout() || nerr.Temporary() {
			return true
		}
	}
	if failure.Is(err, context.DeadlineExceeded) ||
		failure.Is(err, context.Canceled) {
		return true
	}
	return failure.IsCode(err, failure.TimeoutErrorCode)
}

func ErrorInvalidResponse(message string, hres *http.Response) error {
	return failure.NewError(ErrApplication, errMessageWithPath(message, hres))
}

func errMessageWithPath(message string, hres *http.Response) error {
	if hres != nil {
		return fmt.Errorf("%s (%s %s)", message, hres.Request.Method, hres.Request.URL.Path)
	} else {
		return fmt.Errorf(message)
	}
}
