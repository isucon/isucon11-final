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
	CountActiveStudents               score.ScoreTag = "_01.ActiveStudents"
	CountFinishCourses                score.ScoreTag = "_02.FinishCourses"
	CountGetCourseDetailVerifySkipped score.ScoreTag = "_03.GetCourseDetailVerifySkipped"
	CountStartCourseUnder10           score.ScoreTag = "_04.StartCourseUnder10"
	CountStartCourseUnder20           score.ScoreTag = "_05.StartCourseUnder20"
	CountStartCourseUnder30           score.ScoreTag = "_06.StartCourseUnder30"
	CountStartCourseUnder40           score.ScoreTag = "_07.StartCourseUnder40"
	CountStartCourseUnder50           score.ScoreTag = "_08.StartCourseUnder50"
	CountStartCourseFull              score.ScoreTag = "_09.StartCourseFull"
	CountStartCourseOver50            score.ScoreTag = "_10.StartCourseOver50"
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
	CountStartCourseUnder10,
	CountStartCourseUnder20,
	CountStartCourseUnder30,
	CountStartCourseUnder40,
	CountStartCourseUnder50,
	CountStartCourseFull,
	CountStartCourseOver50,
}

func MaxTagLength() int {
	maxLength := 0
	for _, tag := range ScoreTags {
		if len(tag) > maxLength {
			maxLength = len(tag)
		}
	}
	return maxLength
}

func MaxTagLengthForContestant() int {
	maxLength := 0
	for _, tag := range ScoreTags {
		if tag[0] != '_' && len(tag) > maxLength {
			maxLength = len(tag)
		}
	}
	return maxLength
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
