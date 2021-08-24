package scenario

import (
	"io"
	"log"
	"os"
)

var (
	// 競技者に見せてもいい内容を書くロガー
	ContestantLogger *log.Logger
	// 運営だけが見れる内容を書くロガー
	AdminLogger *log.Logger
	// デバッグ用で仕込んでいるロガー
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
