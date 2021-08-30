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
	CountSubmitAssignment             score.ScoreTag = "09.SubmitAssignment"
	CountDownloadSubmissions          score.ScoreTag = "10.DownloadSubmissions"
	CountRegisterScore                score.ScoreTag = "11.RegisterScore"
	CountAddAnnouncement              score.ScoreTag = "12.AddAnnouncement"
	CountGetAnnouncementList          score.ScoreTag = "13.GetAnnouncementList"
	CountGetAnnouncementsDetail       score.ScoreTag = "14.GetAnnouncementDetail"
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

var Tags = []score.ScoreTag{
	CountRegisterCourses,
	CountGetRegisteredCourses,
	CountGetGrades,
	CountSearchCourses,
	CountGetCourseDetail,
	CountAddCourse,
	CountAddClass,
	CountGetClasses,
	CountSubmitAssignment,
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

var (
	// TagsForContestant 競技者に見せるタグ一覧
	// _ で始まるタグは競技者には見せない
	TagsForContestant         []score.ScoreTag
	MaxTagLength              int
	MaxTagLengthForContestant int
)

func init() {
	for _, tag := range Tags {
		if tag[0] != '_' {
			TagsForContestant = append(TagsForContestant, tag)
		}
	}
	MaxTagLength = maxLength(Tags)
	MaxTagLengthForContestant = maxLength(TagsForContestant)
}

func maxLength(arr []score.ScoreTag) int {
	max := 0
	for _, v := range arr {
		if len(v) > max {
			max = len(v)
		}
	}
	return max
}

type mag int64      // 1回でn点
type fraction int64 // n回で1点

var scoreCoefTable = map[score.ScoreTag]interface{}{
	CountRegisterCourses:  mag(10),
	CountSubmitAssignment: mag(5),

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
