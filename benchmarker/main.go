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
	"sync"
	"sync/atomic"
	"time"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon10-portal/bench-tool.go/benchrun"                            // TODO: modify to isucon11-portal
	isuxportalResources "github.com/isucon/isucon10-portal/proto.go/isuxportal/resources" // TODO: modify to isucon11-portal

	"github.com/isucon/isucon11-final/benchmarker/scenario"
)

const (
	benchTimeout   string = "70s"
	errorThreshold int64  = 100
)

var (
	COMMIT           string
	targetAddress    string
	profileFile      string
	useTLS           bool
	exitStatusOnFail bool
	noLoad           bool
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
	flag.StringVar(&timeoutDuration, "timeout", "10s", "request timeout duration")
	flag.BoolVar(&exitStatusOnFail, "exit-status", false, "set exit status non-zero when a benchmark result is failing")
	flag.BoolVar(&useTLS, "tls", false, "target server is a tls")
	flag.BoolVar(&noLoad, "no-load", false, "exit on finished prepare")
	flag.BoolVar(&showVersion, "version", false, "show version and exit 1")

	flag.Parse()

	timeout, err := time.ParseDuration(timeoutDuration)
	if err != nil {
		panic(err)
	}
	agent.DefaultRequestTimeout = timeout
}

func checkError(err error) (critical bool, timeout bool, deduction bool) {
	critical = false  // TODO: クリティカルなエラー(起きたら即ベンチを止める)
	timeout = false   // TODO: リクエストタイムアウト(ある程度の数許容するかも)
	deduction = false // TODO: 減点対象になるエラー

	return
}

func sendResult(s *scenario.Scenario, result *isucandar.BenchmarkResult, finish bool) bool {
	passed := true
	reason := ""
	errors := result.Errors.All()

	deduction := int64(0)
	timeoutCount := int64(0)

	for _, err := range errors {
		isCritical, isTimeout, isDeduction := checkError(err)

		switch true {
		case isCritical:
			passed = false
			reason = "Critical error"
		case isTimeout:
			timeoutCount++
		case isDeduction:
			deduction++
		}
	}

	err := reporter.Report(&isuxportalResources.BenchmarkResult{
		SurveyResponse: &isuxportalResources.SurveyResponse{
			Language: s.Language(),
		},
		Finished: finish,
		Passed:   passed,
		Score:    0, // TODO: 加点 - 減点
		ScoreBreakdown: &isuxportalResources.BenchmarkResult_ScoreBreakdown{
			Raw:       0, // TODO: 加点
			Deduction: 0, // TODO: 減点
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

	if profileFile != "" {
		fs, err := os.Create(profileFile)
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(fs)
		defer pprof.StopCPUProfile()
	}
	if targetAddress == "" {
		targetAddress = "localhost:9292"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := scenario.NewScenario()
	if err != nil {
		panic(err)
	}
	scheme := "http"
	if useTLS {
		scheme = "https"
	}
	s.BaseURL, err = url.Parse(fmt.Sprintf("%s://%s/", scheme, targetAddress))
	if err != nil {
		panic(err)
	}
	s.NoLoad = noLoad

	benchTimeout, err := time.ParseDuration(benchTimeout)
	if err != nil {
		panic(err)
	}
	b, err := isucandar.NewBenchmark(isucandar.WithLoadTimeout(benchTimeout))
	if err != nil {
		panic(err)
	}

	reporter, err = benchrun.NewReporter(false)
	if err != nil {
		panic(err)
	}

	errorCount := int64(0)
	b.OnError(func(err error, step *isucandar.BenchmarkStep) {
		// Load 中の timeout のみログから除外
		if failure.IsCode(err, failure.TimeoutErrorCode) && failure.IsCode(err, isucandar.ErrLoad) {
			return
		}

		critical, _, deduction := checkError(err)

		if critical || (deduction && atomic.AddInt64(&errorCount, 1) >= errorThreshold) {
			step.Cancel()
		}

		scenario.ContestantLogger.Printf("ERR: %v", err)
	})

	b.AddScenario(s)

	wg := sync.WaitGroup{}
	b.Load(func(ctx context.Context, step *isucandar.BenchmarkStep) error {
		if noLoad {
			return nil
		}

		wg.Add(1)
		defer wg.Done()

		startAt := time.Now()
		// 途中経過を3秒毎に送信
		ticker := time.NewTicker(3 * time.Second)
		for {
			select {
			case <-ticker.C:
				scenario.AdminLogger.Printf("[debug] %.f seconds have passed\n", time.Since(startAt).Seconds())
			case <-ctx.Done():
				ticker.Stop()
				return nil
			}
		}
	})

	result := b.Start(ctx)

	wg.Wait()

	if !sendResult(s, result, true) && exitStatusOnFail {
		os.Exit(1)
	}
}
