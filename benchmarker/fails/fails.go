package fails

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"

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
	ErrJSON           failure.StringCode = "json-error"
	ErrInvalidStatus  failure.StringCode = "invalid-status-code"
	ErrStaticResource failure.StringCode = "invalid-resource"
)

func IsCritical(err error) bool {
	return failure.IsCode(err, ErrCritical)
}

func IsDeduction(err error) bool {
	return failure.IsCode(err, ErrApplication) ||
		failure.IsCode(err, ErrHTTP) ||
		failure.IsCode(err, ErrJSON) ||
		failure.IsCode(err, ErrInvalidStatus) ||
		failure.IsCode(err, ErrStaticResource)
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

func ErrorCritical(err error) error {
	return failure.NewError(ErrCritical, err)
}

func ErrorInvalidResponse(err error, hres *http.Response) error {
	return failure.NewError(ErrApplication, errMessageWithPath(err, hres))
}

func ErrorHTTP(err error) error {
	return failure.NewError(ErrHTTP, err)
}

func ErrorJSON(err error, hres *http.Response) error {
	switch e := err.(type) {
	case *json.SyntaxError:
		return failure.NewError(ErrJSON, errMessageWithPath(fmt.Errorf("JSONの形式が不正です (%w)", e), hres))
	case *json.UnmarshalTypeError:
		return failure.NewError(ErrJSON, errMessageWithPath(fmt.Errorf("レスポンスに含まれる %s のデータ型が不正です", e.Field), hres))
	default:
		return failure.NewError(ErrJSON, errMessageWithPath(fmt.Errorf("JSONのデコードに失敗しました (%w)", e), hres))
	}
}

func ErrorInvalidStatusCode(hres *http.Response, expected []int) error {
	str := ""
	for _, v := range expected {
		str += strconv.Itoa(v) + " or "
	}
	str = str[:len(str)-4]
	return failure.NewError(ErrInvalidStatus, errMessageWithPathAndDiff(errors.New("期待するHTTPステータスコード以外が返却されました"), hres, str, strconv.Itoa(hres.StatusCode)))
}

func ErrorStaticResource(err error) error {
	return failure.NewError(ErrStaticResource, err)
}

func errMessageWithPath(err error, hres *http.Response) error {
	if hres != nil {
		return fmt.Errorf("%w (%s %s)", err, hres.Request.Method, hres.Request.URL.Path)
	} else {
		return err
	}
}

func errMessageWithPathAndDiff(err error, hres *http.Response, expected string, actual string) error {
	if hres != nil {
		return fmt.Errorf("%w (%s %s, expected: %s, actual: %s)", err, hres.Request.Method, hres.Request.URL.Path, expected, actual)
	} else {
		return err
	}
}
