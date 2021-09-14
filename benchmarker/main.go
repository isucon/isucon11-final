package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"net/url"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
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
	allowedTargetFQDN        = "isucholar.t.isucon.dev"
)

var (
	COMMIT           string
	targetAddress    string
	profileFile      string
	memProfileDir    string
	promOut          string
	useTLS           bool
	exitStatusOnFail bool
	noLoad           bool
	noPrepare        bool
	noResource       bool
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
	flag.StringVar(&promOut, "prom-out", "", "prometheus text-file output path")
	flag.BoolVar(&exitStatusOnFail, "exit-status", false, "set exit status non-zero when a benchmark result is failing")
	flag.BoolVar(&useTLS, "tls", false, "target server is a tls")
	flag.BoolVar(&noLoad, "no-load", false, "exit on finished prepare")
	flag.BoolVar(&noPrepare, "no-prepare", false, "only load and validation step")
	flag.BoolVar(&noResource, "no-resource", false, "do not verify page resource")
	flag.BoolVar(&isDebug, "is-debug", false, "silence debug log")
	flag.BoolVar(&showVersion, "version", false, "show version and exit 1")

	flag.Parse()

	if targetAddress == "" {
		targetAddress = "localhost:8080"
	}

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
	scoreTable := result.Score.Breakdown()

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

	totalScore, rawScore, deductScore, rawBreakdown := score.Calc(scoreTable, deductionCount, timeoutCount)
	if totalScore <= 0 {
		totalScore = 0
		if passed && !noLoad {
			passed = false
			reason = "スコアが0点以下でした"
		}
	}

	scenario.ContestantLogger.Printf("score: %d (= %d - %d) : %s", totalScore, rawScore, deductScore, reason)
	scenario.ContestantLogger.Printf("deductionCount: %d, timeoutCount: %d", deductionCount, timeoutCount)

	// 競技者には最終的な raw score の内訳のみ見せる
	if finish {
		scenario.ContestantLogger.Printf("raw score (%d) breakdown:", rawScore)
		tagFormat := fmt.Sprintf("%%-%ds : %%d 回 (%%d 点)", score.MaxTagLengthForContestant)
		for _, tag := range score.TagsForContestant {
			scenario.ContestantLogger.Printf(tagFormat, tag, scoreTable[tag], rawBreakdown[tag])
		}
	}

	if writeScoreToAdminLogger {
		tagFormat := fmt.Sprintf("tag: %%-%ds : %%d", score.MaxTagLength)
		for _, tag := range score.Tags {
			scenario.AdminLogger.Printf(tagFormat, tag, scoreTable[tag])
		}
	}

	// Prometheus metrics
	var promTags PromTags
	for _, tag := range score.Tags {
		promTags.writeTag("score_tag", strconv.Itoa(int(scoreTable[tag])), "tag", string(tag))
	}
	for _, tag := range score.Tags {
		if tagScore, ok := rawBreakdown[tag]; ok {
			promTags.writeTag("score_raw_breakdown", strconv.Itoa(int(tagScore)), "tag", string(tag))
		}
	}
	promTags.writeTag("score_total", strconv.Itoa(int(totalScore)))
	promTags.writeTag("score_raw", strconv.Itoa(int(rawScore)))
	promTags.writeTag("score_deduct", strconv.Itoa(int(deductScore)))
	promTags.writeTag("deduction_count", strconv.Itoa(int(deductionCount)))
	promTags.writeTag("timeout_count", strconv.Itoa(int(timeoutCount)))

	promTags.writePromFile()
	promTags.commit()

	// Reporter
	err := reporter.Report(&isuxportalResources.BenchmarkResult{
		SurveyResponse: &isuxportalResources.SurveyResponse{
			Language: s.Language(),
		},
		Finished: finish,
		Passed:   passed,
		Score:    totalScore, // TODO: 加点 - 減点
		ScoreBreakdown: &isuxportalResources.BenchmarkResult_ScoreBreakdown{
			Raw:       rawScore,    // TODO: 加点
			Deduction: deductScore, // TODO: 減点
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

	scheme := "http"
	if useTLS {
		scheme = "https"
	}

	baseURL, err := url.Parse(fmt.Sprintf("%s://%s/", scheme, targetAddress))
	if err != nil {
		panic(err)
	}

	if useTLS {
		agent.DefaultTLSConfig.ServerName = allowedTargetFQDN
	}

	config := &scenario.Config{
		BaseURL:          baseURL,
		UseTLS:           useTLS,
		NoLoad:           noLoad,
		NoPrepare:        noPrepare,
		NoVerifyResource: noResource,
		IsDebug:          isDebug,
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
