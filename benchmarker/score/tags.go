package score

import "github.com/isucon/isucandar/score"

const (
	// score
	ScoreSubmitAssignment    score.ScoreTag = "SubmitAssignment"
	ScoreGetAnnouncementList score.ScoreTag = "GetAnnouncementList"

	// other (needs attention)
	SkipRegisterNoCourseAvailable score.ScoreTag = "!1.SkipRegisterNoCourseAvailable"
	ValidateTimeout               score.ScoreTag = "!2.ValidateTimeout"

	// other
	ActiveStudents           score.ScoreTag = "_O01.ActiveStudents"
	StartCourseStudents      score.ScoreTag = "_O02.StartCourseStudents"
	FinishCourses            score.ScoreTag = "_O03.FinishCourses"
	FinishCourseStudents     score.ScoreTag = "_O04.FinishCourseStudents"
	GetAnnouncementsDetail   score.ScoreTag = "_O05.GetAnnouncementDetail"
	CourseStartCourseUnder10 score.ScoreTag = "_O06.StartCourseUnder10"
	CourseStartCourseUnder20 score.ScoreTag = "_O07.StartCourseUnder20"
	CourseStartCourseUnder30 score.ScoreTag = "_O08.StartCourseUnder30"
	CourseStartCourseUnder40 score.ScoreTag = "_O09.StartCourseUnder40"
	CourseStartCourseUnder50 score.ScoreTag = "_O10.StartCourseUnder50"
	CourseStartCourseFull    score.ScoreTag = "_O11.StartCourseFull"

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
	ScoreSubmitAssignment,
	ScoreGetAnnouncementList,

	// other (needs attention)
	SkipRegisterNoCourseAvailable,
	ValidateTimeout,

	// other
	ActiveStudents,
	StartCourseStudents,
	FinishCourses,
	FinishCourseStudents,
	GetAnnouncementsDetail,

	CourseStartCourseUnder10,
	CourseStartCourseUnder20,
	CourseStartCourseUnder30,
	CourseStartCourseUnder40,
	CourseStartCourseUnder50,
	CourseStartCourseFull,

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
