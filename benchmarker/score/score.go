package score

import (
	"github.com/isucon/isucandar/score"
)

const (
	CountAddCourse              = "add course"
	CountAddClass               = "add class"
	CountSubmitPDF              = "submit assignment"
	CountRegisterScore          = "register score"
	CountGetAnnouncements       = "get announcements"
	CountGetAnnouncementsDetail = "get announcement detail"
	CountGetGrades              = "get grades"
	CountSearchCourse           = "search courses"
	CountRegisterCourses        = "register courses"
)

type mag int64      // 1回でn点
type fraction int64 // n回で1点

var scoreCoefTable = map[score.ScoreTag]interface{}{
	CountSubmitPDF:       mag(5),
	CountRegisterCourses: mag(10),

	CountGetAnnouncements: fraction(10),
	CountGetGrades:        fraction(10),
}

var (
	deductionScore = int64(50) // エラーの減点スコア

	// (timeoutDeductFraction)回あたり減点(timeoutDeductionScore)点
	timeoutDeductionScore = int64(100) // タイムアウトの減点スコア
	timeoutDeductFraction = int64(100) // タイムアウトで減点される回数
)

func Calc(result score.ScoreTable, deductionCount, timeoutCount int64) (totalScore, rawScore, deductScore int64) {
	for tag, coefficient := range scoreCoefTable {
		if mag, ok := coefficient.(mag); ok {
			rawScore += result[tag] * int64(mag)
		} else if fraction, ok := coefficient.(fraction); ok {
			rawScore += result[tag] / int64(fraction)
		}
	}

	deductScore += deductionCount * deductionScore
	deductScore += (timeoutCount / timeoutDeductFraction) * timeoutDeductionScore

	totalScore = rawScore - deductScore
	return
}
