package scenario

import (
	"math"
	"reflect"

	"github.com/isucon/isucon11-final/benchmarker/api"
	"github.com/isucon/isucon11-final/benchmarker/model"
)

func AssertEqual(msg string, expected interface{}, actual interface{}) bool {
	r := assertEqual(expected, actual)
	if !r {
		AdminLogger.Printf("%s: expected: %v / actual: %v", msg, expected, actual)
	}
	return r
}

func assertEqual(expected interface{}, actual interface{}) bool {
	if expected == nil || actual == nil {
		return expected == actual
	}

	actualType := reflect.TypeOf(actual)
	if actualType == nil {
		return false
	}
	expectedValue := reflect.ValueOf(expected)
	if expectedValue.IsValid() && expectedValue.Type().ConvertibleTo(actualType) {
		return reflect.DeepEqual(expectedValue.Convert(actualType).Interface(), actual)
	}

	return false
}

func AssertGreaterOrEqual(msg string, expectMin, actual int) bool {
	r := expectMin <= actual
	if !r {
		AdminLogger.Printf("%s: expected: >= %d / actual: %d", msg, expectMin, actual)
	}
	return r
}

func AssertWithinTolerance(msg string, expect, actual, tolerance float64) bool {
	r := math.Abs(expect-actual) <= tolerance
	if !r {
		AdminLogger.Printf("%s: expected: %f ± %.2f / actual: %f", msg, expect, tolerance, actual)
	}
	return r
}

func AssertEqualUserAccount(expected *model.UserAccount, actual *api.GetMeResponse) error {
	if !AssertEqual("account code", expected.Code, actual.Code) {
		return errInvalidResponse("学内コードが期待する値と一致しません")
	}

	if !AssertEqual("account name", expected.Name, actual.Name) {
		return errInvalidResponse("氏名が期待する値と一致しません")
	}

	if !AssertEqual("account is_admin", expected.IsAdmin, actual.IsAdmin) {
		return errInvalidResponse("管理者フラグが期待する値と一致しません")
	}

	return nil
}

func AssertEqualRegisteredCourse(expected *model.Course, actual *api.GetRegisteredCourseResponseContent) error {
	if !AssertEqual("registered_course id", expected.ID, actual.ID) {
		return errInvalidResponse("科目IDが期待する値と一致しません")
	}

	if !AssertEqual("registered_course name", expected.Name, actual.Name) {
		return errInvalidResponse("科目名が期待する値と一致しません")
	}

	if !AssertEqual("registered_course teacher", expected.Teacher().Name, actual.Teacher) {
		return errInvalidResponse("科目の教員名が期待する値と一致しません")
	}

	if !AssertEqual("registered_course period", uint8(expected.Period+1), actual.Period) {
		return errInvalidResponse("科目の開講時限が期待する値と一致しません")
	}

	if !AssertEqual("registered_course day_of_weeek", api.DayOfWeekTable[expected.DayOfWeek], actual.DayOfWeek) {
		return errInvalidResponse("科目の開講曜日が期待する値と一致しません")
	}

	return nil
}

func AssertEqualSummary(expected *model.Summary, actual *api.Summary) error {
	if !AssertEqual("grade summary credits", expected.Credits, actual.Credits) {
		return errInvalidResponse("成績確認の summary の credits が一致しません")
	}

	if !AssertWithinTolerance("grade summary gpa", expected.GPA, actual.GPA, validateGPAErrorTolerance) {
		return errInvalidResponse("成績確認の summary の gpa が一致しません")
	}

	if !AssertWithinTolerance("grade summary gpa_max", expected.GpaMax, actual.GpaMax, validateGPAErrorTolerance) {
		return errInvalidResponse("成績確認の summary の gpa_max が一致しません")
	}

	if !AssertWithinTolerance("grade summary gpa_min", expected.GpaMin, actual.GpaMin, validateGPAErrorTolerance) {
		return errInvalidResponse("成績確認の summary の gpa_min が一致しません")
	}

	if !AssertWithinTolerance("grade summary gpa_avg", expected.GpaAvg, actual.GpaAvg, validateGPAErrorTolerance) {
		return errInvalidResponse("成績確認の summary の gpa_avg が一致しません")
	}

	if !AssertWithinTolerance("grade summary gpa_t_score", expected.GpaTScore, actual.GpaTScore, validateGPAErrorTolerance) {
		return errInvalidResponse("成績確認の summary の gpa_t_score が一致しません")
	}

	return nil
}

func AssertEqualCourseResult(expected *model.CourseResult, actual *api.CourseResult) error {
	if !AssertEqual("grade courses name", expected.Name, actual.Name) {
		return errInvalidResponse("成績確認の科目名が一致しません")
	}

	if !AssertEqual("grade courses code", expected.Code, actual.Code) {
		return errInvalidResponse("成績確認の科目のコードが一致しません")
	}

	if !AssertEqual("grade courses total_score", expected.TotalScore, actual.TotalScore) {
		return errInvalidResponse("成績確認の科目の total_score が一致しません")
	}

	if !AssertEqual("grade courses total_score_max", expected.TotalScoreMax, actual.TotalScoreMax) {
		return errInvalidResponse("成績確認の科目の total_score_max が一致しません")
	}

	if !AssertEqual("grade courses total_score_min", expected.TotalScoreMin, actual.TotalScoreMin) {
		return errInvalidResponse("成績確認の科目の total_score_min が一致しません")
	}

	if !AssertWithinTolerance("grade courses total_score_avg", expected.TotalScoreAvg, actual.TotalScoreAvg, validateTotalScoreErrorTolerance) {
		return errInvalidResponse("成績確認の科目の total_score_avg が一致しません")
	}

	if !AssertWithinTolerance("grade courses total_score_t_score", expected.TotalScoreTScore, actual.TotalScoreTScore, validateTotalScoreErrorTolerance) {
		return errInvalidResponse("成績確認の科目の total_score_t_score が一致しません")
	}

	if !AssertEqual("grade courses class_scores length", len(expected.ClassScores), len(actual.ClassScores)) {
		return errInvalidResponse("成績確認の科目の class_scores の数が一致しません")
	}

	for i := 0; i < len(expected.ClassScores); i++ {
		// webapp 側は新しい(partが大きい)classから順番に帰ってくるので古いクラスから見るようにしている
		err := AssertEqualClassScore(expected.ClassScores[i], &actual.ClassScores[len(actual.ClassScores)-i-1])
		if err != nil {
			return err
		}
	}

	return nil
}

func AssertEqualClassScore(expected *model.ClassScore, actual *api.ClassScore) error {
	if !AssertEqual("grade courses class_scores class_id", expected.ClassID, actual.ClassID) {
		return errInvalidResponse("成績確認の講義IDが一致しません")
	}

	if !AssertEqual("grade courses class_scores title", expected.Title, actual.Title) {
		return errInvalidResponse("成績確認の講義のタイトルが一致しません")
	}

	if !AssertEqual("grade courses class_scores part", expected.Part, actual.Part) {
		return errInvalidResponse("成績確認の講義の part が一致しません")
	}

	if !AssertEqual("grade courses class_scores score", expected.Score, actual.Score) {
		return errInvalidResponse("成績確認の講義の採点結果が一致しません")
	}

	if !AssertEqual("grade courses class_scores submitters", expected.SubmitterCount, actual.Submitters) {
		return errInvalidResponse("成績確認の講義の課題提出者の数が一致しません")
	}

	return nil
}

func AssertEqualSimpleClassScore(expected *model.SimpleClassScore, actual *api.ClassScore) error {
	if !AssertEqual("grade courses class_scores class_id", expected.ClassID, actual.ClassID) {
		return errInvalidResponse("成績確認での講義IDが一致しません")
	}

	if !AssertEqual("grade courses class_scores title", expected.Title, actual.Title) {
		return errInvalidResponse("成績確認での講義のタイトルが一致しません")
	}

	if !AssertEqual("grade courses class_scores part", expected.Part, actual.Part) {
		return errInvalidResponse("成績確認での講義の part が一致しません")
	}

	if !AssertEqual("grade courses class_scores score", expected.Score, actual.Score) {
		return errInvalidResponse("成績確認での講義の採点結果が一致しません")
	}

	return nil
}

func AssertEqualCourse(expected *model.Course, actual *api.GetCourseDetailResponse) error {
	if !AssertEqual("course id", expected.Code, actual.Code) {
		return errInvalidResponse("科目IDが期待する値と一致しません")
	}

	if !AssertEqual("course code", expected.Code, actual.Code) {
		return errInvalidResponse("科目のコードが期待する値と一致しません")
	}

	if !AssertEqual("course type", api.CourseType(expected.Type), actual.Type) {
		return errInvalidResponse("科目のタイプが期待する値と一致しません")
	}

	if !AssertEqual("course name", expected.Name, actual.Name) {
		return errInvalidResponse("科目名が期待する値と一致しません")
	}

	if !AssertEqual("course description", expected.Description, actual.Description) {
		return errInvalidResponse("科目の詳細が期待する値と一致しません")
	}

	if !AssertEqual("course credit", uint8(expected.Credit), actual.Credit) {
		return errInvalidResponse("科目の単位数が期待する値と一致しません")
	}

	if !AssertEqual("course period", uint8(expected.Period+1), actual.Period) {
		return errInvalidResponse("科目の開講時限が期待する値と一致しません")
	}

	if !AssertEqual("course day_of_week", api.DayOfWeekTable[expected.DayOfWeek], actual.DayOfWeek) {
		return errInvalidResponse("科目の開講曜日が期待する値と一致しません")
	}

	if !AssertEqual("course teacher", expected.Teacher().Name, actual.Teacher) {
		return errInvalidResponse("科目の教員名が期待する値と一致しません")
	}

	if !AssertEqual("course status", expected.Status(), actual.Status) {
		return errInvalidResponse("科目のステータスが期待する値と一致しません")
	}

	if !AssertEqual("course keywords", expected.Keywords, actual.Keywords) {
		return errInvalidResponse("科目のキーワードが期待する値と一致しません")
	}

	return nil
}

func AssertEqualClass(expected *model.Class, actual *api.GetClassResponse) error {
	if !AssertEqual("class id", expected.ID, actual.ID) {
		return errInvalidResponse("講義IDが期待する値と一致しません")
	}

	if !AssertEqual("class part", expected.Part, actual.Part) {
		return errInvalidResponse("講義のパートが期待する値と一致しません")
	}

	if !AssertEqual("class title", expected.Title, actual.Title) {
		return errInvalidResponse("講義のタイトルが期待する値と一致しません")
	}

	if !AssertEqual("class description", expected.Desc, actual.Description) {
		return errInvalidResponse("講義の説明文が期待する値と一致しません")
	}

	// TODO: SubmissionClosedAtの検証
	// TODO: Submittedの検証

	return nil
}

func AssertEqualAnnouncementListContent(expected *model.AnnouncementStatus, actual *api.AnnouncementResponse, verifyUnread bool) error {
	if !AssertEqual("announcement_list announcements ud", expected.Announcement.ID, actual.ID) {
		return errInvalidResponse("お知らせIDが期待する値と一致しません")
	}

	if !AssertEqual("announcement_list announcements course_id", expected.Announcement.CourseID, actual.CourseID) {
		return errInvalidResponse("お知らせの科目IDが期待する値と一致しません")
	}

	if !AssertEqual("announcement_list announcements course_name", expected.Announcement.CourseName, actual.CourseName) {
		return errInvalidResponse("お知らせの科目名が期待する値と一致しません")
	}

	if !AssertEqual("announcement_list announcements title", expected.Announcement.Title, actual.Title) {
		return errInvalidResponse("お知らせのタイトルが期待する値と一致しません")
	}

	if verifyUnread && !AssertEqual("announcement_list announcements unread", expected.Unread, actual.Unread) {
		return errInvalidResponse("お知らせの未読/既読状態が期待する値と一致しません")
	}

	return nil
}

func AssertEqualAnnouncementDetail(expected *model.AnnouncementStatus, actual *api.GetAnnouncementDetailResponse, verifyUnread bool) error {
	if !AssertEqual("announcement_detail id", expected.Announcement.ID, actual.ID) {
		return errInvalidResponse("お知らせのIDが期待する値と一致しません")
	}

	if !AssertEqual("announcement_detail course_id", expected.Announcement.CourseID, actual.CourseID) {
		return errInvalidResponse("お知らせの講義IDが期待する値と一致しません")
	}

	if !AssertEqual("announcement_detail course_name", expected.Announcement.CourseName, actual.CourseName) {
		return errInvalidResponse("お知らせの講義名が期待する値と一致しません")
	}

	if !AssertEqual("announcement_detail title", expected.Announcement.Title, actual.Title) {
		return errInvalidResponse("お知らせのタイトルが期待する値と一致しません")
	}

	if !AssertEqual("announcement_detail message", expected.Announcement.Message, actual.Message) {
		return errInvalidResponse("お知らせのメッセージが期待する値と一致しません")
	}

	if verifyUnread && !AssertEqual("announcement_detail unread", expected.Unread, actual.Unread) {
		return errInvalidResponse("お知らせの未読/既読状態が期待する値と一致しません")
	}

	return nil
}
