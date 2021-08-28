package score

import (
	"github.com/isucon/isucandar/score"
)

const (
	CountAddCourse              score.ScoreTag = "add course"
	CountAddClass               score.ScoreTag = "add class"
	CountSubmitPDF              score.ScoreTag = "submit pdf assignment"
	CountSubmitDocx             score.ScoreTag = "submit docx assignment"
	CountRegisterScore          score.ScoreTag = "register score"
	CountAddAnnouncement        score.ScoreTag = "add announcement"
	CountGetAnnouncements       score.ScoreTag = "get announcements"
	CountGetAnnouncementsDetail score.ScoreTag = "get announcement detail"
	CountDownloadSubmission     score.ScoreTag = "download submissions"
	CountGetGrades              score.ScoreTag = "get grades"
	CountSearchCourse           score.ScoreTag = "search courses"
	CountRegisterCourses        score.ScoreTag = "register courses"
)

var ScoreTags = []score.ScoreTag{
	CountAddCourse,
	CountAddClass,
	CountSubmitPDF,
	CountSubmitDocx,
	CountRegisterScore,
	CountAddAnnouncement,
	CountGetAnnouncements,
	CountGetAnnouncementsDetail,
	CountDownloadSubmission,
	CountGetGrades,
	CountSearchCourse,
	CountRegisterCourses,
}

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
