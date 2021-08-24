package scenario

import (
	"io"
	"log"
	"os"
)

var (
	// ContestantLogger は競技者に見せてもいい内容を書くロガー
	// ex: エラー内容や最終スコア
	ContestantLogger *log.Logger
	// AdminLogger は運営だけが見れる内容を書くロガー
	// ex: 本番で改善傾向追うための途中スコアやAddErrorしたログ
	AdminLogger *log.Logger
	// DebugLogger デバッグ用で仕込んでいるロガー
	// ex: リクエスト単位で仕込んでいてリクエスト数とか見たければgrepなどで調べる
	DebugLogger *log.Logger
)

func init() {
	ContestantLogger = log.New(os.Stdout, "", log.Lmicroseconds)
	AdminLogger = log.New(os.Stderr, "", log.Lmicroseconds)
	DebugLogger = log.New(os.Stderr, "", log.Lmicroseconds)
}

func SilenceDebugLog() {
	DebugLogger = log.New(io.Discard, "", log.Lmicroseconds)
}
