package score

import "github.com/isucon/isucandar/score"

type mag int64      // 1回でn点
type fraction int64 // n回で1点

var scoreCoefTable = map[score.ScoreTag]interface{}{
	ScoreSubmitAssignment: mag(5),
	ScoreGetGrades:        mag(1),

	ScoreGetAnnouncementList: fraction(2),
}

var (
	deductionScore = int64(50) // エラーの減点スコア

	// (timeoutDeductFraction)回あたり減点(timeoutDeductionScore)点
	timeoutDeductionScore = int64(100) // タイムアウトの減点スコア
	timeoutDeductFraction = int64(100) // タイムアウトで減点される回数
)

func Calc(scoreTable score.ScoreTable, deductionCount, timeoutCount int64) (totalScore, rawScore, deductScore int64, rawBreakdown map[score.ScoreTag]int64) {
	rawBreakdown = make(map[score.ScoreTag]int64, len(scoreCoefTable))

	for tag, coefficient := range scoreCoefTable {
		var tagScore int64
		if mag, ok := coefficient.(mag); ok {
			tagScore = scoreTable[tag] * int64(mag)
		} else if fraction, ok := coefficient.(fraction); ok {
			tagScore = scoreTable[tag] / int64(fraction)
		}
		rawScore += tagScore
		rawBreakdown[tag] = tagScore
	}

	deductScore += deductionCount * deductionScore
	deductScore += (timeoutCount / timeoutDeductFraction) * timeoutDeductionScore

	totalScore = rawScore - deductScore
	return
}
