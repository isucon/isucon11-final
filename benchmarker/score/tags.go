package score

import "github.com/isucon/isucandar/score"

const (
	RegisterCourses        score.ScoreTag = "01.RegisterCourses"
	GetRegisteredCourses   score.ScoreTag = "02.GetRegisteredCourses"
	GetGrades              score.ScoreTag = "03.GetGrades"
	SearchCourses          score.ScoreTag = "04.SearchCourses"
	GetCourseDetail        score.ScoreTag = "05.GetCourseDetail"
	AddCourse              score.ScoreTag = "06.AddCourse"
	AddClass               score.ScoreTag = "07.AddClass"
	GetClasses             score.ScoreTag = "08.GetClasses"
	SubmitAssignment       score.ScoreTag = "09.SubmitAssignment"
	DownloadSubmissions    score.ScoreTag = "10.DownloadSubmissions"
	RegisterScore          score.ScoreTag = "11.RegisterScore"
	AddAnnouncement        score.ScoreTag = "12.AddAnnouncement"
	GetAnnouncementList    score.ScoreTag = "13.GetAnnouncementList"
	GetAnnouncementsDetail score.ScoreTag = "14.GetAnnouncementDetail"

	ActiveStudents               score.ScoreTag = "_01.ActiveStudents"
	FinishCourses                score.ScoreTag = "_02.FinishCourses"
	GetCourseDetailVerifySkipped score.ScoreTag = "_03.GetCourseDetailVerifySkipped"
	StartCourseUnder50           score.ScoreTag = "_04.StartCourseUnder50"
	StartCourseFull              score.ScoreTag = "_05.StartCourseFull"

	StartCourseOver50 score.ScoreTag = "!01.StartCourseOver50"
)

var Tags = []score.ScoreTag{
	RegisterCourses,
	GetRegisteredCourses,
	GetGrades,
	SearchCourses,
	GetCourseDetail,
	AddCourse,
	AddClass,
	GetClasses,
	SubmitAssignment,
	DownloadSubmissions,
	RegisterScore,
	AddAnnouncement,
	GetAnnouncementList,
	GetAnnouncementsDetail,

	ActiveStudents,
	FinishCourses,
	GetCourseDetailVerifySkipped,
	StartCourseUnder50,
	StartCourseFull,

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
