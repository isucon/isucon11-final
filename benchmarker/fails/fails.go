package fails

import "github.com/isucon/isucandar/failure"

const (
	// BenchmarkStepにAddErrorしていくエラー郡
	// ErrCritical は即失格となるエラー
	ErrCritical failure.StringCode = "critical-error"
	// ErrApplication は正しいアプリケーションの挙動と異なるときのエラー。ある程度許容される。
	ErrApplication failure.StringCode = "application-error"
	// ErrHTTP はアプリケーションへの接続周りでのエラー。ある程度許容される。
	ErrHTTP failure.StringCode = "http-error"
)
