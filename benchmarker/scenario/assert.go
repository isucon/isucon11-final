package scenario

import (
	"math"
	"net/http"
	"reflect"

	"github.com/isucon/isucon11-final/benchmarker/api"
	"github.com/isucon/isucon11-final/benchmarker/fails"
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

func AssertEqualUserAccount(expected *model.UserAccount, actual *api.GetMeResponse, hres *http.Response) error {
	if !AssertEqual("account code", expected.Code, actual.Code) {
		return fails.ErrorInvalidResponse("学内コードが期待する値と一致しません", hres)
	}

	if !AssertEqual("account name", expected.Name, actual.Name) {
		return fails.ErrorInvalidResponse("氏名が期待する値と一致しません", hres)
	}

	if !AssertEqual("account is_admin", expected.IsAdmin, actual.IsAdmin) {
		return fails.ErrorInvalidResponse("管理者フラグが期待する値と一致しません", hres)
	}

	return nil
}

func AssertEqualRegisteredCourse(expected *model.Course, actual *api.GetRegisteredCourseResponseContent, hres *http.Response) error {
	if !AssertEqual("registered_course id", expected.ID, actual.ID) {
		return fails.ErrorInvalidResponse("科目IDが期待する値と一致しません", hres)
	}

	if !AssertEqual("registered_course name", expected.Name, actual.Name) {
		return fails.ErrorInvalidResponse("科目名が期待する値と一致しません", hres)
	}

	if !AssertEqual("registered_course teacher", expected.Teacher().Name, actual.Teacher) {
		return fails.ErrorInvalidResponse("科目の教員名が期待する値と一致しません", hres)
	}

	if !AssertEqual("registered_course period", uint8(expected.Period+1), actual.Period) {
		return fails.ErrorInvalidResponse("科目の開講時限が期待する値と一致しません", hres)
	}

	if !AssertEqual("registered_course day_of_weeek", api.DayOfWeekTable[expected.DayOfWeek], actual.DayOfWeek) {
		return fails.ErrorInvalidResponse("科目の開講曜日が期待する値と一致しません", hres)
	}

	return nil
}

func AssertEqualSummary(expected *model.Summary, actual *api.Summary, hres *http.Response) error {
	if !AssertEqual("grade summary credits", expected.Credits, actual.Credits) {
		return fails.ErrorInvalidResponse("成績確認の summary の credits が一致しません", hres)
	}

	if !AssertWithinTolerance("grade summary gpa", expected.GPA, actual.GPA, validateGPAErrorTolerance) {
		return fails.ErrorInvalidResponse("成績確認の summary の gpa が一致しません", hres)
	}

	if !AssertWithinTolerance("grade summary gpa_max", expected.GpaMax, actual.GpaMax, validateGPAErrorTolerance) {
		return fails.ErrorInvalidResponse("成績確認の summary の gpa_max が一致しません", hres)
	}

	if !AssertWithinTolerance("grade summary gpa_min", expected.GpaMin, actual.GpaMin, validateGPAErrorTolerance) {
		return fails.ErrorInvalidResponse("成績確認の summary の gpa_min が一致しません", hres)
	}

	if !AssertWithinTolerance("grade summary gpa_avg", expected.GpaAvg, actual.GpaAvg, validateGPAErrorTolerance) {
		return fails.ErrorInvalidResponse("成績確認の summary の gpa_avg が一致しません", hres)
	}

	if !AssertWithinTolerance("grade summary gpa_t_score", expected.GpaTScore, actual.GpaTScore, validateGPAErrorTolerance) {
		return fails.ErrorInvalidResponse("成績確認の summary の gpa_t_score が一致しません", hres)
	}

	return nil
}

func AssertEqualCourseResult(expected *model.CourseResult, actual *api.CourseResult, hres *http.Response) error {
	if !AssertEqual("grade courses name", expected.Name, actual.Name) {
		return fails.ErrorInvalidResponse("成績確認の科目名が一致しません", hres)
	}

	if !AssertEqual("grade courses code", expected.Code, actual.Code) {
		return fails.ErrorInvalidResponse("成績確認の科目のコードが一致しません", hres)
	}

	if !AssertEqual("grade courses total_score", expected.TotalScore, actual.TotalScore) {
		return fails.ErrorInvalidResponse("成績確認の科目の total_score が一致しません", hres)
	}

	if !AssertEqual("grade courses total_score_max", expected.TotalScoreMax, actual.TotalScoreMax) {
		return fails.ErrorInvalidResponse("成績確認の科目の total_score_max が一致しません", hres)
	}

	if !AssertEqual("grade courses total_score_min", expected.TotalScoreMin, actual.TotalScoreMin) {
		return fails.ErrorInvalidResponse("成績確認の科目の total_score_min が一致しません", hres)
	}

	if !AssertWithinTolerance("grade courses total_score_avg", expected.TotalScoreAvg, actual.TotalScoreAvg, validateTotalScoreErrorTolerance) {
		return fails.ErrorInvalidResponse("成績確認の科目の total_score_avg が一致しません", hres)
	}

	if !AssertWithinTolerance("grade courses total_score_t_score", expected.TotalScoreTScore, actual.TotalScoreTScore, validateTotalScoreErrorTolerance) {
		return fails.ErrorInvalidResponse("成績確認の科目の total_score_t_score が一致しません", hres)
	}

	if !AssertEqual("grade courses class_scores length", len(expected.ClassScores), len(actual.ClassScores)) {
		return fails.ErrorInvalidResponse("成績確認の科目の class_scores の数が一致しません", hres)
	}

	for i := 0; i < len(expected.ClassScores); i++ {
		// webapp 側は新しい(partが大きい)classから順番に帰ってくるので古い講義から見るようにしている
		err := AssertEqualClassScore(expected.ClassScores[i], &actual.ClassScores[len(actual.ClassScores)-i-1], hres)
		if err != nil {
			return err
		}
	}

	return nil
}

func AssertEqualClassScore(expected *model.ClassScore, actual *api.ClassScore, hres *http.Response) error {
	if !AssertEqual("grade courses class_scores class_id", expected.ClassID, actual.ClassID) {
		return fails.ErrorInvalidResponse("成績確認の講義IDが一致しません", hres)
	}

	if !AssertEqual("grade courses class_scores title", expected.Title, actual.Title) {
		return fails.ErrorInvalidResponse("成績確認の講義のタイトルが一致しません", hres)
	}

	if !AssertEqual("grade courses class_scores part", expected.Part, actual.Part) {
		return fails.ErrorInvalidResponse("成績確認の講義の part が一致しません", hres)
	}

	if !AssertEqual("grade courses class_scores score", expected.Score, actual.Score) {
		return fails.ErrorInvalidResponse("成績確認の講義の採点結果が一致しません", hres)
	}

	if !AssertEqual("grade courses class_scores submitters", expected.SubmitterCount, actual.Submitters) {
		return fails.ErrorInvalidResponse("成績確認の講義の課題提出者の数が一致しません", hres)
	}

	return nil
}

func AssertEqualSimpleClassScore(expected *model.SimpleClassScore, actual *api.ClassScore, hres *http.Response) error {
	if !AssertEqual("grade courses class_scores class_id", expected.ClassID, actual.ClassID) {
		return fails.ErrorInvalidResponse("成績確認での講義IDが一致しません", hres)
	}

	if !AssertEqual("grade courses class_scores title", expected.Title, actual.Title) {
		return fails.ErrorInvalidResponse("成績確認での講義のタイトルが一致しません", hres)
	}

	if !AssertEqual("grade courses class_scores part", expected.Part, actual.Part) {
		return fails.ErrorInvalidResponse("成績確認での講義の part が一致しません", hres)
	}

	if !AssertEqual("grade courses class_scores score", expected.Score, actual.Score) {
		return fails.ErrorInvalidResponse("成績確認での講義の採点結果が一致しません", hres)
	}

	return nil
}

func AssertEqualCourse(expected *model.Course, actual *api.GetCourseDetailResponse, hres *http.Response) error {
	if !AssertEqual("course id", expected.ID, actual.ID) {
		return fails.ErrorInvalidResponse("科目IDが期待する値と一致しません", hres)
	}

	if !AssertEqual("course code", expected.Code, actual.Code) {
		return fails.ErrorInvalidResponse("科目のコードが期待する値と一致しません", hres)
	}

	if !AssertEqual("course type", api.CourseType(expected.Type), actual.Type) {
		return fails.ErrorInvalidResponse("科目のタイプが期待する値と一致しません", hres)
	}

	if !AssertEqual("course name", expected.Name, actual.Name) {
		return fails.ErrorInvalidResponse("科目名が期待する値と一致しません", hres)
	}

	if !AssertEqual("course description", expected.Description, actual.Description) {
		return fails.ErrorInvalidResponse("科目の詳細が期待する値と一致しません", hres)
	}

	if !AssertEqual("course credit", uint8(expected.Credit), actual.Credit) {
		return fails.ErrorInvalidResponse("科目の単位数が期待する値と一致しません", hres)
	}

	if !AssertEqual("course period", uint8(expected.Period+1), actual.Period) {
		return fails.ErrorInvalidResponse("科目の開講時限が期待する値と一致しません", hres)
	}

	if !AssertEqual("course day_of_week", api.DayOfWeekTable[expected.DayOfWeek], actual.DayOfWeek) {
		return fails.ErrorInvalidResponse("科目の開講曜日が期待する値と一致しません", hres)
	}

	if !AssertEqual("course teacher", expected.Teacher().Name, actual.Teacher) {
		return fails.ErrorInvalidResponse("科目の教員名が期待する値と一致しません", hres)
	}

	if !AssertEqual("course status", expected.Status(), actual.Status) {
		return fails.ErrorInvalidResponse("科目のステータスが期待する値と一致しません", hres)
	}

	if !AssertEqual("course keywords", expected.Keywords, actual.Keywords) {
		return fails.ErrorInvalidResponse("科目のキーワードが期待する値と一致しません", hres)
	}

	return nil
}

func AssertEqualClass(expected *model.Class, actual *api.GetClassResponse, hres *http.Response) error {
	if !AssertEqual("class id", expected.ID, actual.ID) {
		return fails.ErrorInvalidResponse("講義IDが期待する値と一致しません", hres)
	}

	if !AssertEqual("class part", expected.Part, actual.Part) {
		return fails.ErrorInvalidResponse("講義のパートが期待する値と一致しません", hres)
	}

	if !AssertEqual("class title", expected.Title, actual.Title) {
		return fails.ErrorInvalidResponse("講義のタイトルが期待する値と一致しません", hres)
	}

	if !AssertEqual("class description", expected.Desc, actual.Description) {
		return fails.ErrorInvalidResponse("講義の説明文が期待する値と一致しません", hres)
	}

	// TODO: SubmissionClosedAtの検証
	// TODO: Submittedの検証

	return nil
}

func AssertEqualAnnouncementListContent(expected *model.AnnouncementStatus, actual *api.AnnouncementResponse, hres *http.Response, verifyUnread bool) error {
	if !AssertEqual("announcement_list announcements id", expected.Announcement.ID, actual.ID) {
		return fails.ErrorInvalidResponse("お知らせIDが期待する値と一致しません", hres)
	}

	if !AssertEqual("announcement_list announcements course_id", expected.Announcement.CourseID, actual.CourseID) {
		return fails.ErrorInvalidResponse("お知らせの科目IDが期待する値と一致しません", hres)
	}

	if !AssertEqual("announcement_list announcements course_name", expected.Announcement.CourseName, actual.CourseName) {
		return fails.ErrorInvalidResponse("お知らせの科目名が期待する値と一致しません", hres)
	}

	if !AssertEqual("announcement_list announcements title", expected.Announcement.Title, actual.Title) {
		return fails.ErrorInvalidResponse("お知らせのタイトルが期待する値と一致しません", hres)
	}

	if verifyUnread && !AssertEqual("announcement_list announcements unread", expected.Unread, actual.Unread) {
		return fails.ErrorInvalidResponse("お知らせの未読/既読状態が期待する値と一致しません", hres)
	}

	return nil
}

func AssertEqualAnnouncementDetail(expected *model.AnnouncementStatus, actual *api.GetAnnouncementDetailResponse, hres *http.Response, verifyUnread bool) error {
	if !AssertEqual("announcement_detail id", expected.Announcement.ID, actual.ID) {
		return fails.ErrorInvalidResponse("お知らせのIDが期待する値と一致しません", hres)
	}

	if !AssertEqual("announcement_detail course_id", expected.Announcement.CourseID, actual.CourseID) {
		return fails.ErrorInvalidResponse("お知らせの講義IDが期待する値と一致しません", hres)
	}

	if !AssertEqual("announcement_detail course_name", expected.Announcement.CourseName, actual.CourseName) {
		return fails.ErrorInvalidResponse("お知らせの講義名が期待する値と一致しません", hres)
	}

	if !AssertEqual("announcement_detail title", expected.Announcement.Title, actual.Title) {
		return fails.ErrorInvalidResponse("お知らせのタイトルが期待する値と一致しません", hres)
	}

	if !AssertEqual("announcement_detail message", expected.Announcement.Message, actual.Message) {
		return fails.ErrorInvalidResponse("お知らせのメッセージが期待する値と一致しません", hres)
	}

	if verifyUnread && !AssertEqual("announcement_detail unread", expected.Unread, actual.Unread) {
		return fails.ErrorInvalidResponse("お知らせの未読/既読状態が期待する値と一致しません", hres)
	}

	return nil
}
