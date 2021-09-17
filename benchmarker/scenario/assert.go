package scenario

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"

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

func errMismatch(message string, expected interface{}, actual interface{}) error {
	return fmt.Errorf("%s (expected: %v, actual: %v)", message, expected, actual)
}

func AssertEqualUserAccount(expected *model.UserAccount, actual *api.GetMeResponse) error {
	if !AssertEqual("account code", expected.Code, actual.Code) {
		return errMismatch("ユーザ情報の code が期待する値と一致しません", expected.Code, actual.Code)
	}

	if !AssertEqual("account name", expected.Name, actual.Name) {
		return errMismatch("ユーザ情報の name が期待する値と一致しません", expected.Name, actual.Name)
	}

	if !AssertEqual("account is_admin", expected.IsAdmin, actual.IsAdmin) {
		return errMismatch("ユーザ情報の is_admin が期待する値と一致しません", expected.IsAdmin, actual.IsAdmin)
	}

	return nil
}

func AssertEqualRegisteredCourse(expected *model.Course, actual *api.GetRegisteredCourseResponseContent) error {
	if !AssertEqual("registered_course id", expected.ID, actual.ID) {
		return errMismatch("科目の id が期待する値と一致しません", expected.ID, actual.ID)
	}

	if !AssertEqual("registered_course name", expected.Name, actual.Name) {
		return errMismatch("科目の name が期待する値と一致しません", expected.Name, actual.Name)
	}

	if !AssertEqual("registered_course teacher", expected.Teacher().Name, actual.Teacher) {
		return errMismatch("科目の teacher が期待する値と一致しません", expected.Teacher().Name, actual.Teacher)
	}

	if !AssertEqual("registered_course period", uint8(expected.Period+1), actual.Period) {
		return errMismatch("科目の period が期待する値と一致しません", uint8(expected.Period+1), actual.Period)
	}

	if !AssertEqual("registered_course day_of_weeek", api.DayOfWeekTable[expected.DayOfWeek], actual.DayOfWeek) {
		return errMismatch("科目の day_of_week が期待する値と一致しません", api.DayOfWeekTable[expected.DayOfWeek], actual.DayOfWeek)
	}

	return nil
}

func AssertEqualGrade(expected *model.GradeRes, actual *api.GetGradeResponse) error {
	if !AssertEqual("grade courses length", len(expected.CourseResults), len(actual.CourseResults)) {
		return errMismatch("成績取得の courses の数が期待する値と一致しません", len(expected.CourseResults), len(actual.CourseResults))
	}

	err := AssertEqualSummary(&expected.Summary, &actual.Summary)
	if err != nil {
		return err
	}

	for _, courseResult := range actual.CourseResults {
		if _, ok := expected.CourseResults[courseResult.Code]; !ok {
			return errors.New("成績取得の courses に期待しない科目が含まれています")
		}

		expected := expected.CourseResults[courseResult.Code]
		err := AssertEqualCourseResult(expected, &courseResult)
		if err != nil {
			return err
		}
	}

	return nil
}

func AssertEqualSummary(expected *model.Summary, actual *api.Summary) error {
	if !AssertEqual("grade summary credits", expected.Credits, actual.Credits) {
		return errMismatch("成績取得の summary の credits が期待する値と一致しません", expected.Credits, actual.Credits)
	}

	if !AssertWithinTolerance("grade summary gpa", expected.GPA, actual.GPA, validateGPAErrorTolerance) {
		return errMismatch("成績取得の summary の gpa が期待する値と一致しません", expected.GPA, actual.GPA)
	}

	if !AssertWithinTolerance("grade summary gpa_max", expected.GpaMax, actual.GpaMax, validateGPAErrorTolerance) {
		return errMismatch("成績取得の summary の gpa_max が期待する値と一致しません", expected.GpaMax, actual.GpaMax)
	}

	if !AssertWithinTolerance("grade summary gpa_min", expected.GpaMin, actual.GpaMin, validateGPAErrorTolerance) {
		return errMismatch("成績取得の summary の gpa_min が期待する値と一致しません", expected.GpaMin, actual.GpaMin)
	}

	if !AssertWithinTolerance("grade summary gpa_avg", expected.GpaAvg, actual.GpaAvg, validateGPAErrorTolerance) {
		return errMismatch("成績取得の summary の gpa_avg が期待する値と一致しません", expected.GpaAvg, actual.GpaAvg)
	}

	if !AssertWithinTolerance("grade summary gpa_t_score", expected.GpaTScore, actual.GpaTScore, validateGPAErrorTolerance) {
		return errMismatch("成績取得の summary の gpa_t_score が期待する値と一致しません", expected.GpaTScore, actual.GpaTScore)
	}

	return nil
}

func AssertEqualCourseResult(expected *model.CourseResult, actual *api.CourseResult) error {
	if !AssertEqual("grade courses name", expected.Name, actual.Name) {
		return errMismatch("成績取得の科目の name が期待する値と一致しません", expected.Name, actual.Name)
	}

	if !AssertEqual("grade courses code", expected.Code, actual.Code) {
		return errMismatch("成績取得の科目の code が期待する値と一致しません", expected.Code, actual.Code)
	}

	if !AssertEqual("grade courses total_score", expected.TotalScore, actual.TotalScore) {
		return errMismatch("成績取得の科目の total_score が期待する値と一致しません", expected.TotalScore, actual.TotalScore)
	}

	if !AssertEqual("grade courses total_score_max", expected.TotalScoreMax, actual.TotalScoreMax) {
		return errMismatch("成績取得の科目の total_score_max が期待する値と一致しません", expected.TotalScoreMax, actual.TotalScoreMax)
	}

	if !AssertEqual("grade courses total_score_min", expected.TotalScoreMin, actual.TotalScoreMin) {
		return errMismatch("成績取得の科目の total_score_min が期待する値と一致しません", expected.TotalScoreMin, actual.TotalScoreMin)
	}

	if !AssertWithinTolerance("grade courses total_score_avg", expected.TotalScoreAvg, actual.TotalScoreAvg, validateTotalScoreErrorTolerance) {
		return errMismatch("成績取得の科目の total_score_avg が期待する値と一致しません", expected.TotalScoreAvg, actual.TotalScoreAvg)
	}

	if !AssertWithinTolerance("grade courses total_score_t_score", expected.TotalScoreTScore, actual.TotalScoreTScore, validateTotalScoreErrorTolerance) {
		return errMismatch("成績取得の科目の total_score_t_score が期待する値と一致しません", expected.TotalScoreTScore, actual.TotalScoreTScore)
	}

	if !AssertEqual("grade courses class_scores length", len(expected.ClassScores), len(actual.ClassScores)) {
		return errMismatch("成績取得の科目の class_scores の数が期待する値と一致しません", len(expected.ClassScores), len(actual.ClassScores))
	}

	for i := 0; i < len(expected.ClassScores); i++ {
		// webapp 側は新しい(partが大きい)classから順番に帰ってくるので古い講義から見るようにしている
		err := AssertEqualClassScore(expected.ClassScores[i], &actual.ClassScores[len(actual.ClassScores)-i-1])
		if err != nil {
			return err
		}
	}

	return nil
}

func AssertEqualClassScore(expected *model.ClassScore, actual *api.ClassScore) error {
	if !AssertEqual("grade courses class_scores class_id", expected.ClassID, actual.ClassID) {
		return errMismatch("成績取得の講義の class_id が期待する値と一致しません", expected.ClassID, actual.ClassID)
	}

	if !AssertEqual("grade courses class_scores title", expected.Title, actual.Title) {
		return errMismatch("成績取得の講義の title が期待する値と一致しません", expected.Title, actual.Title)
	}

	if !AssertEqual("grade courses class_scores part", expected.Part, actual.Part) {
		return errMismatch("成績取得の講義の part が期待する値と一致しません", expected.Part, actual.Part)
	}

	if !AssertEqual("grade courses class_scores score", expected.Score, actual.Score) {
		return errMismatch("成績取得での講義の score が期待する値と一致しません", scoreToString(expected.Score), scoreToString(actual.Score))
	}

	if !AssertEqual("grade courses class_scores submitters", expected.SubmitterCount, actual.Submitters) {
		return errMismatch("成績取得の講義の submitters が期待する値と一致しません", expected.SubmitterCount, actual.Submitters)
	}

	return nil
}

func AssertEqualSimpleClassScore(expected *model.SimpleClassScore, actual *api.ClassScore) error {
	if !AssertEqual("grade courses class_scores class_id", expected.ClassID, actual.ClassID) {
		return errMismatch("成績取得での講義の class_id が期待する値と一致しません", expected.ClassID, actual.ClassID)
	}

	if !AssertEqual("grade courses class_scores title", expected.Title, actual.Title) {
		return errMismatch("成績取得での講義の title が期待する値と一致しません", expected.Title, actual.Title)
	}

	if !AssertEqual("grade courses class_scores part", expected.Part, actual.Part) {
		return errMismatch("成績取得での講義の part が期待する値と一致しません", expected.Part, actual.Part)
	}

	if !AssertEqual("grade courses class_scores score", expected.Score, actual.Score) {
		return errMismatch("成績取得での講義の score が期待する値と一致しません", scoreToString(expected.Score), scoreToString(actual.Score))
	}

	return nil
}

func AssertEqualCourse(expected *model.Course, actual *api.GetCourseDetailResponse, verifyStatus bool) error {
	if !AssertEqual("course id", expected.ID, actual.ID) {
		return errMismatch("科目の id が期待する値と一致しません", expected.ID, actual.ID)
	}

	if !AssertEqual("course code", expected.Code, actual.Code) {
		return errMismatch("科目の code が期待する値と一致しません", expected.Code, actual.Code)
	}

	if !AssertEqual("course type", api.CourseType(expected.Type), actual.Type) {
		return errMismatch("科目の type が期待する値と一致しません", api.CourseType(expected.Type), actual.Type)
	}

	if !AssertEqual("course name", expected.Name, actual.Name) {
		return errMismatch("科目の name が期待する値と一致しません", expected.Name, actual.Name)
	}

	if !AssertEqual("course description", expected.Description, actual.Description) {
		return errMismatch("科目の description が期待する値と一致しません", expected.Description, actual.Description)
	}

	if !AssertEqual("course credit", uint8(expected.Credit), actual.Credit) {
		return errMismatch("科目の credit が期待する値と一致しません", uint8(expected.Credit), actual.Credit)
	}

	if !AssertEqual("course period", uint8(expected.Period+1), actual.Period) {
		return errMismatch("科目の period が期待する値と一致しません", uint8(expected.Period+1), actual.Period)
	}

	if !AssertEqual("course day_of_week", api.DayOfWeekTable[expected.DayOfWeek], actual.DayOfWeek) {
		return errMismatch("科目の day_of_week が期待する値と一致しません", api.DayOfWeekTable[expected.DayOfWeek], actual.DayOfWeek)
	}

	if !AssertEqual("course teacher", expected.Teacher().Name, actual.Teacher) {
		return errMismatch("科目の teacher が期待する値と一致しません", expected.Teacher().Name, actual.Teacher)
	}

	if verifyStatus && !AssertEqual("course status", expected.Status(), actual.Status) {
		return errMismatch("科目の status が期待する値と一致しません", expected.Status(), actual.Status)
	}

	if !AssertEqual("course keywords", expected.Keywords, actual.Keywords) {
		return errMismatch("科目の keywords が期待する値と一致しません", expected.Keywords, actual.Keywords)
	}

	return nil
}

func AssertEqualClass(expected *model.Class, actual *api.GetClassResponse, student *model.Student) error {
	if !AssertEqual("class id", expected.ID, actual.ID) {
		return errMismatch("講義の id が期待する値と一致しません", expected.ID, actual.ID)
	}

	if !AssertEqual("class part", expected.Part, actual.Part) {
		return errMismatch("講義の part が期待する値と一致しません", expected.Part, actual.Part)
	}

	if !AssertEqual("class title", expected.Title, actual.Title) {
		return errMismatch("講義の title が期待する値と一致しません", expected.Title, actual.Title)
	}

	if !AssertEqual("class description", expected.Desc, actual.Description) {
		return errMismatch("講義の description が期待する値と一致しません", expected.Desc, actual.Description)
	}

	if !AssertEqual("class submission_closed", expected.IsSubmissionClosed(), actual.SubmissionClosed) {
		return errMismatch("講義の submission_closed が期待する値と一致しません", expected.IsSubmissionClosed(), actual.SubmissionClosed)
	}

	isSubmitted := expected.GetSubmissionByStudentCode(student.Code) != nil
	if !AssertEqual("class submitted", isSubmitted, actual.Submitted) {
		return errMismatch("講義の submitted が期待する値と一致しません", isSubmitted, actual.Submitted)
	}

	return nil
}

func AssertEqualAnnouncementListContent(expected *model.AnnouncementStatus, actual *api.AnnouncementResponse, verifyUnread bool) error {
	if !AssertEqual("announcement_list announcements id", expected.Announcement.ID, actual.ID) {
		return errMismatch("お知らせの id が期待する値と一致しません", expected.Announcement.ID, actual.ID)
	}

	if !AssertEqual("announcement_list announcements course_id", expected.Announcement.CourseID, actual.CourseID) {
		return errMismatch("お知らせの course_id が期待する値と一致しません", expected.Announcement.CourseID, actual.CourseID)
	}

	if !AssertEqual("announcement_list announcements course_name", expected.Announcement.CourseName, actual.CourseName) {
		return errMismatch("お知らせの course_name が期待する値と一致しません", expected.Announcement.CourseName, actual.CourseName)
	}

	if !AssertEqual("announcement_list announcements title", expected.Announcement.Title, actual.Title) {
		return errMismatch("お知らせの title が期待する値と一致しません", expected.Announcement.Title, actual.Title)
	}

	if verifyUnread && !AssertEqual("announcement_list announcements unread", expected.Unread, actual.Unread) {
		return errMismatch("お知らせの unread が期待する値と一致しません", expected.Unread, actual.Unread)
	}

	return nil
}

func AssertEqualAnnouncementDetail(expected *model.AnnouncementStatus, actual *api.GetAnnouncementDetailResponse, verifyUnread bool) error {
	if !AssertEqual("announcement_detail id", expected.Announcement.ID, actual.ID) {
		return errMismatch("お知らせの id が期待する値と一致しません", expected.Announcement.ID, actual.ID)
	}

	if !AssertEqual("announcement_detail course_id", expected.Announcement.CourseID, actual.CourseID) {
		return errMismatch("お知らせの course_id が期待する値と一致しません", expected.Announcement.CourseID, actual.CourseID)
	}

	if !AssertEqual("announcement_detail course_name", expected.Announcement.CourseName, actual.CourseName) {
		return errMismatch("お知らせの course_name が期待する値と一致しません", expected.Announcement.CourseName, actual.CourseName)
	}

	if !AssertEqual("announcement_detail title", expected.Announcement.Title, actual.Title) {
		return errMismatch("お知らせの title が期待する値と一致しません", expected.Announcement.Title, actual.Title)
	}

	if !AssertEqual("announcement_detail message", expected.Announcement.Message, actual.Message) {
		return errMismatch("お知らせの message が期待する値と一致しません", expected.Announcement.Message, actual.Message)
	}

	if verifyUnread && !AssertEqual("announcement_detail unread", expected.Unread, actual.Unread) {
		return errMismatch("お知らせの unread が期待する値と一致しません", expected.Unread, actual.Unread)
	}

	return nil
}

func scoreToString(score *int) string {
	var str string
	if score == nil {
		str = "null"
	} else {
		str = strconv.Itoa(*score)
	}
	return str
}
