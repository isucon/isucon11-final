package score

import (
	"os"
	"testing"

	"github.com/isucon/isucandar/score"
)

func TestMain(m *testing.M) {
	scoreCoefTable["magCount"] = mag(10)
	scoreCoefTable["fractionCount"] = fraction(10)
	deductionScore = int64(10)        // 1回あたり-1点
	timeoutDeductionScore = int64(10) // 10回あたり
	timeoutDeductFraction = int64(10) // -10点

	code := m.Run()
	os.Exit(code)
}

func TestCalc_RawScore(t *testing.T) {
	tests := map[string]struct {
		magCount      int64
		fractionCount int64
		expectScore   int64
	}{
		"mag(10), count(1)": {
			magCount:    1,
			expectScore: 10,
		},
		"fraction(10), count(11)": {
			fractionCount: 11,
			expectScore:   1,
		},
		"fraction(10), count(9)": {
			fractionCount: 9,
			expectScore:   0,
		},
	}

	for title, td := range tests {
		td := td
		t.Run(title, func(t *testing.T) {
			scoreTable := map[score.ScoreTag]int64{
				"magCount":      td.magCount,
				"fractionCount": td.fractionCount,
			}

			_, rawScore, _ := Calc(scoreTable, 0, 0)
			if rawScore != td.expectScore {
				t.Errorf("expect:%d, but actual:%d", td.expectScore, rawScore)
			}
		})
	}
}

func TestCalc_DeductScore(t *testing.T) {
	tests := map[string]struct {
		deductCount  int64
		timeoutCount int64
		expectScore  int64
	}{
		"deduct(1)": {
			deductCount: 1,
			expectScore: 10,
		},
		"timeout(10)": {
			timeoutCount: 10,
			expectScore:  10,
		},
		"timeout(9)": {
			timeoutCount: 9,
			expectScore:  0,
		},
	}

	for title, td := range tests {
		td := td
		t.Run(title, func(t *testing.T) {
			scoreTable := map[score.ScoreTag]int64{}

			_, _, deductCount := Calc(scoreTable, td.deductCount, td.timeoutCount)
			if deductCount != td.expectScore {
				t.Errorf("expect:%d, but actual:%d", td.expectScore, deductCount)
			}
		})
	}
}
