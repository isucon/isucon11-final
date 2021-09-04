package score

import "github.com/isucon/isucandar/score"

const (
	SubmitAssignment    score.ScoreTag = "01.SubmitAssignment"
	GetGrades           score.ScoreTag = "02.GetGrades"
	GetAnnouncementList score.ScoreTag = "03.GetAnnouncementList"

	ActiveStudents               score.ScoreTag = "_01.ActiveStudents"
	FinishCourses                score.ScoreTag = "_02.FinishCourses"
	FinishCoursesStudents        score.ScoreTag = "_03.FinishCoursesStudents"
	GetCourseDetailVerifySkipped score.ScoreTag = "_04.GetCourseDetailVerifySkipped"
	StartCourseUnder50           score.ScoreTag = "_05.StartCourseUnder50"
	StartCourseFull              score.ScoreTag = "_06.StartCourseFull"
	RegisterCourses              score.ScoreTag = "_07.RegisterCourses"
	GetRegisteredCourses         score.ScoreTag = "_08.GetRegisteredCourses"
	SearchCourses                score.ScoreTag = "_09.SearchCourses"
	GetCourseDetail              score.ScoreTag = "_10.GetCourseDetail"
	AddCourse                    score.ScoreTag = "_11.AddCourse"
	AddClass                     score.ScoreTag = "_12.AddClass"
	GetClasses                   score.ScoreTag = "_13.GetClasses"
	DownloadSubmissions          score.ScoreTag = "_14.DownloadSubmissions"
	RegisterScore                score.ScoreTag = "_15.RegisterScore"
	AddAnnouncement              score.ScoreTag = "_16.AddAnnouncement"
	GetAnnouncementsDetail       score.ScoreTag = "_17.GetAnnouncementDetail"

	StartCourseOver50 score.ScoreTag = "!01.StartCourseOver50"
)

var Tags = []score.ScoreTag{
	SubmitAssignment,
	GetGrades,
	GetAnnouncementList,

	ActiveStudents,
	FinishCourses,
	FinishCoursesStudents,
	GetCourseDetailVerifySkipped,
	StartCourseUnder50,
	StartCourseFull,
	RegisterCourses,
	GetRegisteredCourses,
	SearchCourses,
	GetCourseDetail,
	AddCourse,
	AddClass,
	GetClasses,
	DownloadSubmissions,
	RegisterScore,
	AddAnnouncement,
	GetAnnouncementsDetail,

	StartCourseOver50,
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
