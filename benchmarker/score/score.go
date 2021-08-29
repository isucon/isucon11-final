package score

import (
	"github.com/isucon/isucandar/score"
)

const (
	CountRegisterCourses              score.ScoreTag = "01.RegisterCourses"
	CountGetRegisteredCourses         score.ScoreTag = "02.GetRegisteredCourses"
	CountGetGrades                    score.ScoreTag = "03.GetGrades"
	CountSearchCourses                score.ScoreTag = "04.SearchCourses"
	CountGetCourseDetail              score.ScoreTag = "05.GetCourseDetail"
	CountAddCourse                    score.ScoreTag = "06.AddCourse"
	CountAddClass                     score.ScoreTag = "07.AddClass"
	CountGetClasses                   score.ScoreTag = "08.GetClasses"
	CountSubmitValidAssignment        score.ScoreTag = "09.SubmitValidAssignment"
	CountSubmitInvalidAssignment      score.ScoreTag = "10.SubmitInvalidAssignment"
	CountDownloadSubmissions          score.ScoreTag = "11.DownloadSubmissions"
	CountRegisterScore                score.ScoreTag = "12.RegisterScore"
	CountAddAnnouncement              score.ScoreTag = "13.AddAnnouncement"
	CountGetAnnouncementList          score.ScoreTag = "14.GetAnnouncementList"
	CountGetAnnouncementsDetail       score.ScoreTag = "15.GetAnnouncementDetail"
	CountActiveStudents               score.ScoreTag = "_1.ActiveStudents"
	CountFinishCourses                score.ScoreTag = "_2.FinishCourses"
	CountGetCourseDetailVerifySkipped score.ScoreTag = "_3.GetCourseDetailVerifySkipped"
)

var ScoreTags = []score.ScoreTag{
	CountRegisterCourses,
	CountGetRegisteredCourses,
	CountGetGrades,
	CountSearchCourses,
	CountGetCourseDetail,
	CountAddCourse,
	CountAddClass,
	CountGetClasses,
	CountSubmitValidAssignment,
	CountSubmitInvalidAssignment,
	CountDownloadSubmissions,
	CountRegisterScore,
	CountAddAnnouncement,
	CountGetAnnouncementList,
	CountGetAnnouncementsDetail,
	CountActiveStudents,
	CountFinishCourses,
	CountGetCourseDetailVerifySkipped,
}

type mag int64      // 1回でn点
type fraction int64 // n回で1点

var scoreCoefTable = map[score.ScoreTag]interface{}{
	CountRegisterCourses:       mag(10),
	CountSubmitValidAssignment: mag(5),

	CountGetGrades:           fraction(10),
	CountGetAnnouncementList: fraction(10),
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
