package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"math"
	"net/url"
	"os"
	"runtime"
	"runtime/pprof"
	"sync/atomic"
	"time"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon10-portal/bench-tool.go/benchrun" // TODO: modify to isucon11-portal
	isuxportalResources "github.com/isucon/isucon10-portal/proto.go/isuxportal/resources"
	"github.com/pkg/profile"

	"github.com/isucon/isucon11-final/benchmarker/fails"
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
	memProfileDir    string
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
	flag.StringVar(&memProfileDir, "mem-profile", "", "path of output heap profile at max memStats.sys allocated. ex: memprof")
	flag.BoolVar(&exitStatusOnFail, "exit-status", false, "set exit status non-zero when a benchmark result is failing")
	flag.BoolVar(&useTLS, "tls", false, "target server is a tls")
	flag.BoolVar(&noLoad, "no-load", false, "exit on finished prepare")
	flag.BoolVar(&isDebug, "is-debug", false, "silence debug log")
	flag.BoolVar(&showVersion, "version", false, "show version and exit 1")

	flag.Parse()

	agent.DefaultRequestTimeout = defaultRequestTimeout
}

func checkError(err error) (critical bool, timeout bool, deduction bool) {
	if fails.IsCritical(err) {
		critical = true
		return
	}
	if fails.IsTimeout(err) {
		timeout = true
		return
	}
	if fails.IsDeduction(err) {
		deduction = true
		return
	}

	return
}

func sendResult(s *scenario.Scenario, result *isucandar.BenchmarkResult, finish bool, writeScoreToAdminLogger bool) bool {
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
		reason = fmt.Sprintf("エラーの発生回数が%d回を超えました", errorFailThreshold)
	}

	resultScore, raw, deducted := score.Calc(breakdown, deductionCount, timeoutCount)
	if resultScore <= 0 {
		resultScore = 0
		if passed {
			passed = false
			reason = "スコアが0点以下でした"
		}
	}
	bairitu := math.Round((float64(s.ActiveStudentCount()/10) * 100) / 100)
	finalScore := int64(float64(resultScore) * bairitu)

	scenario.ContestantLogger.Printf("score: %d[(%d - %d) * %f] : %s", finalScore, raw, deducted, bairitu, reason)
	scenario.ContestantLogger.Printf("deductionCount: %d, timeoutCount: %d", deductionCount, timeoutCount)

	// 競技者には最終的なScoreTagの統計のみ見せる
	if finish {
		tagFormat := fmt.Sprintf("tag: %%-%ds : %%d", score.MaxTagLengthForContestant)
		for _, tag := range score.TagsForContestant {
			scenario.ContestantLogger.Printf(tagFormat, tag, breakdown[tag])
		}
	}

	if writeScoreToAdminLogger {
		tagFormat := fmt.Sprintf("tag: %%-%ds : %%d", score.MaxTagLength)
		for _, tag := range score.Tags {
			scenario.AdminLogger.Printf(tagFormat, tag, breakdown[tag])
		}
	}

	err := reporter.Report(&isuxportalResources.BenchmarkResult{
		SurveyResponse: &isuxportalResources.SurveyResponse{
			Language: s.Language(),
		},
		Finished: finish,
		Passed:   passed,
		Score:    finalScore, // TODO: 加点 - 減点
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
	if memProfileDir != "" {
		var maxMemStats runtime.MemStats
		go func() {
			for {
				time.Sleep(5 * time.Second)

				var ms runtime.MemStats
				runtime.ReadMemStats(&ms)
				scenario.AdminLogger.Printf("system: %d Kb, heap: %d Kb", ms.Sys/1024, ms.HeapAlloc/1024)

				if ms.Sys > maxMemStats.Sys {
					profile.Start(profile.MemProfile, profile.ProfilePath(memProfileDir)).Stop()
					maxMemStats = ms
				}
			}
		}()
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
		IsDebug: isDebug,
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

		if critical || (deduction && atomic.AddInt64(&errorCount, 1) > errorFailThreshold) {
			step.Cancel()
		}

		scenario.ContestantLogger.Printf("ERR: %v", err)
		scenario.DebugLogger.Printf("ERR: %+v", err) // includes stack trace
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
				scenario.AdminLogger.Printf("[debug] %.f seconds have passed\n", time.Since(startAt).Seconds())
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
