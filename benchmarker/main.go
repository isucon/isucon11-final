package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"net/url"
	"os"
	"runtime/pprof"
	"sort"
	"sync/atomic"
	"time"

	"github.com/isucon/isucon11-final/benchmarker/fails"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"
	score2 "github.com/isucon/isucandar/score"
	"github.com/isucon/isucon10-portal/bench-tool.go/benchrun" // TODO: modify to isucon11-portal
	isuxportalResources "github.com/isucon/isucon10-portal/proto.go/isuxportal/resources"

	"github.com/isucon/isucon11-final/benchmarker/scenario"
	"github.com/isucon/isucon11-final/benchmarker/score"
)

const (
	defaultRequestTimeout = 5 * time.Second
	// loadTimeout はLoadフェーズの終了時間
	// load.goには別途「Loadのリクエストを送り続ける時間」を定義しているので注意
	loadTimeout              = 70 * time.Second
	errorFailThreshold int64 = 100
)

var (
	COMMIT           string
	targetAddress    string
	profileFile      string
	useTLS           bool
	exitStatusOnFail bool
	noLoad           bool
	isDebug          bool
	showVersion      bool
	timeoutDuration  string

	reporter benchrun.Reporter
)

func init() {
	certs, err := x509.SystemCertPool()
	if err != nil {
		panic(err)
	}

	agent.DefaultTLSConfig.ClientCAs = certs
	agent.DefaultTLSConfig.ClientAuth = tls.RequireAndVerifyClientCert
	agent.DefaultTLSConfig.MinVersion = tls.VersionTLS12
	agent.DefaultTLSConfig.InsecureSkipVerify = false

	flag.StringVar(&targetAddress, "target", benchrun.GetTargetAddress(), "ex: localhost:9292")
	flag.StringVar(&profileFile, "profile", "", "ex: cpu.out")
	flag.BoolVar(&exitStatusOnFail, "exit-status", false, "set exit status non-zero when a benchmark result is failing")
	flag.BoolVar(&useTLS, "tls", false, "target server is a tls")
	flag.BoolVar(&noLoad, "no-load", false, "exit on finished prepare")
	flag.BoolVar(&isDebug, "is-debug", false, "silence debug log")
	flag.BoolVar(&showVersion, "version", false, "show version and exit 1")

	flag.Parse()

	agent.DefaultRequestTimeout = defaultRequestTimeout
}

func checkError(err error) (critical bool, timeout bool, deduction bool) {
	critical = false  // TODO: クリティカルなエラー(起きたら即ベンチを止める)
	timeout = false   // TODO: リクエストタイムアウト(ある程度の数許容するかも)
	deduction = false // TODO: 減点対象になるエラー

	if failure.IsCode(err, fails.ErrCritical) {
		critical = true
		return
	}
	if failure.IsCode(err, failure.TimeoutErrorCode) {
		timeout = true
		return
	}

	return
}

func sendResult(s *scenario.Scenario, result *isucandar.BenchmarkResult, finish bool, writeScoreToAdminLogger bool) bool {
	logger := scenario.ContestantLogger
	passed := true
	reason := "passed"
	errors := result.Errors.All()
	breakdown := result.Score.Breakdown()

	deductionCount := int64(0)
	timeoutCount := int64(0)

	for _, err := range errors {
		isCritical, isTimeout, isDeduction := checkError(err)
		switch true {
		case isCritical:
			passed = false
			reason = "致命的なエラーが発生しました"
		case isTimeout:
			timeoutCount++
		case isDeduction:
			deductionCount++
		}
	}
	if passed && deductionCount > errorFailThreshold {
		passed = false
		reason = fmt.Sprintf("エラーが%d回以上発生しました", errorFailThreshold)
	}

	resultScore, raw, deducted := score.Calc(breakdown, deductionCount, timeoutCount)
	if resultScore <= 0 {
		resultScore = 0
		if passed {
			passed = false
			reason = "スコアが0点以下でした"
		}
	}

	logger.Printf("score: %d(%d - %d) : %s", resultScore, raw, deducted, reason)
	logger.Printf("deductionCount: %d, timeoutCount: %d", deductionCount, timeoutCount)

	scoreTags := make([]score2.ScoreTag, 0, len(breakdown))
	for k := range breakdown {
		scoreTags = append(scoreTags, k)
	}
	sort.Slice(scoreTags, func(i, j int) bool {
		return scoreTags[i] < scoreTags[j]
	})

	// 競技者には最終的なScoreTagの統計のみ見せる
	// TODO: 見せるタグを絞る
	if finish {
		for _, tag := range scoreTags {
			scenario.ContestantLogger.Printf("tag: %v: %v", tag, breakdown[tag])
		}
	}

	if writeScoreToAdminLogger {
		for _, tag := range scoreTags {
			scenario.AdminLogger.Printf("tag: %v: %v", tag, breakdown[tag])
		}
	}

	err := reporter.Report(&isuxportalResources.BenchmarkResult{
		SurveyResponse: &isuxportalResources.SurveyResponse{
			Language: s.Language(),
		},
		Finished: finish,
		Passed:   passed,
		Score:    resultScore, // TODO: 加点 - 減点
		ScoreBreakdown: &isuxportalResources.BenchmarkResult_ScoreBreakdown{
			Raw:       raw,      // TODO: 加点
			Deduction: deducted, // TODO: 減点
		},
		Execution: &isuxportalResources.BenchmarkResult_Execution{
			Reason: reason,
		},
	})
	if err != nil {
		panic(err)
	}

	return passed
}

func main() {
	scenario.AdminLogger.Printf("isucon11-final benchmarker %s", COMMIT)

	if showVersion {
		os.Exit(1)
	}

	if !isDebug {
		scenario.SilenceDebugLog()
	}

	if profileFile != "" {
		fs, err := os.Create(profileFile)
		if err != nil {
			panic(err)
		}
		_ = pprof.StartCPUProfile(fs)
		defer pprof.StopCPUProfile()
	}
	if targetAddress == "" {
		targetAddress = "localhost:8080"
	}
	scheme := "http"
	if useTLS {
		scheme = "https"
	}
	baseURL, err := url.Parse(fmt.Sprintf("%s://%s/", scheme, targetAddress))
	if err != nil {
		panic(err)
	}
	config := &scenario.Config{
		BaseURL: baseURL,
		UseTLS:  useTLS,
		NoLoad:  noLoad,
	}

	s, err := scenario.NewScenario(config)
	if err != nil {
		panic(err)
	}

	b, err := isucandar.NewBenchmark(isucandar.WithLoadTimeout(loadTimeout))
	if err != nil {
		panic(err)
	}

	reporter, err = benchrun.NewReporter(false)
	if err != nil {
		panic(err)
	}

	errorCount := int64(0)
	b.OnError(func(err error, step *isucandar.BenchmarkStep) {
		critical, timeout, deduction := checkError(err)
		// Load 中の timeout のみログから除外
		if timeout && failure.IsCode(err, isucandar.ErrLoad) {
			return
		}

		if critical || (deduction && atomic.AddInt64(&errorCount, 1) >= errorFailThreshold) {
			step.Cancel()
		}

		scenario.ContestantLogger.Printf("ERR: %+v", err)
	})

	b.AddScenario(s)

	// 経過時間の記録用
	b.Load(func(ctx context.Context, step *isucandar.BenchmarkStep) error {
		if noLoad {
			return nil
		}

		startAt := time.Now()
		// 途中経過を3秒毎に送信
		ticker := time.NewTicker(3 * time.Second)
		count := 0
		for {
			select {
			case <-ticker.C:
				scenario.DebugLogger.Printf("[debug] %.f seconds have passed\n", time.Since(startAt).Seconds())
				scenario.DebugLogger.Printf("[debug] active student: %v, course: %v, finished course: %v\n", s.ActiveStudentCount(), s.CourseCount(), s.FinishedCourseCount())
				sendResult(s, step.Result(), false, count%5 == 0)
			case <-ctx.Done():
				ticker.Stop()
				return nil
			}
			count++
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := b.Start(ctx)

	if !sendResult(s, result, true, true) && exitStatusOnFail {
		os.Exit(1)
	}
}
