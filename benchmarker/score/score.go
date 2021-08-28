package score

import (
	"github.com/isucon/isucandar/score"
)

const (
	CountAddCourse              score.ScoreTag = "01.add course"
	CountAddClass               score.ScoreTag = "02.add class"
	CountSubmitPDF              score.ScoreTag = "03.submit pdf assignment"
	CountSubmitDocx             score.ScoreTag = "04.submit docx assignment"
	CountRegisterScore          score.ScoreTag = "05.register score"
	CountAddAnnouncement        score.ScoreTag = "06.add announcement"
	CountGetAnnouncements       score.ScoreTag = "07.get announcements"
	CountGetAnnouncementsDetail score.ScoreTag = "08.get announcement detail"
	CountDownloadSubmission     score.ScoreTag = "09.download submissions"
	CountGetGrades              score.ScoreTag = "10.get grades"
	CountSearchCourse           score.ScoreTag = "11.search courses"
	CountRegisterCourses        score.ScoreTag = "12.register courses"
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
