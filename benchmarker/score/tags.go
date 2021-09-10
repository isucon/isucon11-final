package score

import "github.com/isucon/isucandar/score"

const (
	// score
	ScoreRegisterCourseStudents score.ScoreTag = "RegisterCourseStudents"
	ScoreSubmitAssignment       score.ScoreTag = "SubmitAssignment"
	ScoreFinishCourseStudents   score.ScoreTag = "FinishCourseStudents"
	ScoreGetAnnouncementList    score.ScoreTag = "GetAnnouncementList"

	// other
	ActiveStudents           score.ScoreTag = "_O1.ActiveStudents"
	StartCourseStudents      score.ScoreTag = "_O2.StartCourseStudents"
	FinishCourses            score.ScoreTag = "_O3.FinishCourses"
	FinishCourseStudents     score.ScoreTag = "_O4.FinishCourseStudents"
	GetAnnouncementsDetail   score.ScoreTag = "_O5.GetAnnouncementDetail"
	CourseStartCourseUnder50 score.ScoreTag = "_O6.StartCourseUnder50"
	CourseStartCourseFull    score.ScoreTag = "_O7.StartCourseFull"
	CourseStartCourseOver50  score.ScoreTag = "!O8.StartCourseOver50"

	// registration scenario
	RegGetGrades                    score.ScoreTag = "_R1.GetGrades"
	RegSearchCourses                score.ScoreTag = "_R2.SearchCourses"
	RegGetCourseDetail              score.ScoreTag = "_R3.GetCourseDetail"
	RegGetCourseDetailVerifySkipped score.ScoreTag = "_R4.GetCourseDetailVerifySkipped"
	RegGetRegisteredCourses         score.ScoreTag = "_R5.GetRegisteredCourses"
	RegRegisterCourses              score.ScoreTag = "_R6.RegisterCourses"
	RegRegisterCourseStudents       score.ScoreTag = "_R7.RegisterCourseStudents"

	// read announcement scenario
	UnreadGetAnnouncementList   score.ScoreTag = "_U1.GetAnnouncementList"
	UnreadGetAnnouncementDetail score.ScoreTag = "_U2.GetAnnouncementDetail"

	// read announcement paging scenario
	PagingGetAnnouncementList   score.ScoreTag = "_P1.GetAnnouncementList"
	PagingGetAnnouncementDetail score.ScoreTag = "_P2.GetAnnouncementDetail"

	// course scenario
	CourseAddCourse           score.ScoreTag = "_C1.AddCourse"
	CourseAddClass            score.ScoreTag = "_C2.AddClass"
	CourseAddAnnouncement     score.ScoreTag = "_C3.AddAnnouncement"
	CourseGetClasses          score.ScoreTag = "_C4.GetClasses"
	CourseSubmitAssignment    score.ScoreTag = "_C5.SubmitAssignment"
	CourseDownloadSubmissions score.ScoreTag = "_C6.DownloadSubmissions"
	CourseRegisterScore       score.ScoreTag = "_C7.RegisterScore"
)

var Tags = []score.ScoreTag{
	// score
	ScoreRegisterCourseStudents,
	ScoreSubmitAssignment,
	ScoreFinishCourseStudents,
	ScoreGetAnnouncementList,

	// other
	ActiveStudents,
	StartCourseStudents,
	FinishCourses,
	FinishCourseStudents,
	GetAnnouncementsDetail,

	CourseStartCourseUnder50,
	CourseStartCourseFull,
	CourseStartCourseOver50,

	// registration scenario
	RegGetGrades,
	RegSearchCourses,
	RegGetCourseDetail,
	RegGetCourseDetailVerifySkipped,
	RegGetRegisteredCourses,
	RegRegisterCourses,
	RegRegisterCourseStudents,

	// read announcement scenario
	UnreadGetAnnouncementList,
	UnreadGetAnnouncementDetail,

	// read announcement paging scenario
	PagingGetAnnouncementList,
	PagingGetAnnouncementDetail,

	// course scenario
	CourseAddCourse,
	CourseAddClass,
	CourseAddAnnouncement,
	CourseGetClasses,
	CourseSubmitAssignment,
	CourseDownloadSubmissions,
	CourseRegisterScore,
}

var (
	// TagsForContestant 競技者に見せるタグ一覧
	// _ や ! で始まるタグは競技者には見せない
	// ! で始まるタグは一つでもあるとベンチマーカーにバグがある
	TagsForContestant         []score.ScoreTag
	MaxTagLength              int
	MaxTagLengthForContestant int
)

func init() {
	for _, tag := range Tags {
		if tag[0] != '_' && tag[0] != '!' {
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
