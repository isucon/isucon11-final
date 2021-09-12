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

func assertEqualCourseResult(expected *model.CourseResult, actual *api.CourseResult) error {
	if !AssertEqual("grade courses name", expected.Name, actual.Name) {
		return errInvalidResponse("成績確認のコースの名前が一致しません")
	}

	if !AssertEqual("grade courses code", expected.Code, actual.Code) {
		return errInvalidResponse("成績確認のコースのコードが一致しません")
	}

	if !AssertEqual("grade courses total_score", expected.TotalScore, actual.TotalScore) {
		return errInvalidResponse("成績確認のコースのTotalScoreが一致しません")
	}

	if !AssertEqual("grade courses total_score_max", expected.TotalScoreMax, actual.TotalScoreMax) {
		return errInvalidResponse("成績確認のコースのTotalScoreMaxが一致しません")
	}

	if !AssertEqual("grade courses total_score_min", expected.TotalScoreMin, actual.TotalScoreMin) {
		return errInvalidResponse("成績確認のコースのTotalScoreMinが一致しません")
	}

	if !AssertWithinTolerance("grade courses total_score_avg", expected.TotalScoreAvg, actual.TotalScoreAvg, validateTotalScoreErrorTolerance) {
		return errInvalidResponse("成績確認のコースのTotalScoreAvgが一致しません")
	}

	if !AssertWithinTolerance("grade courses total_score_t_score", expected.TotalScoreTScore, actual.TotalScoreTScore, validateTotalScoreErrorTolerance) {
		return errInvalidResponse("成績確認のコースのTotalScoreTScoreが一致しません")
	}

	if !AssertEqual("grade courses class_scores length", len(expected.ClassScores), len(actual.ClassScores)) {
		return errInvalidResponse("成績確認のClassScoresの数が一致しません")
	}

	for i := 0; i < len(expected.ClassScores); i++ {
		// webapp 側は新しい(partが大きい)classから順番に帰ってくるので古いクラスから見るようにしている
		err := assertEqualClassScore(expected.ClassScores[i], &actual.ClassScores[len(actual.ClassScores)-i-1])
		if err != nil {
			return err
		}
	}

	return nil
}

func assertEqualClassScore(expected *model.ClassScore, actual *api.ClassScore) error {
	if !AssertEqual("grade courses class_scores class_id", expected.ClassID, actual.ClassID) {
		return errInvalidResponse("成績確認のクラスのIDが一致しません")
	}

	if !AssertEqual("grade courses class_scores part", expected.Part, actual.Part) {
		return errInvalidResponse("成績確認のクラスのpartが一致しません")
	}

	if !AssertEqual("grade courses class_scores title", expected.Title, actual.Title) {
		return errInvalidResponse("成績確認のクラスのタイトルが一致しません")
	}

	if !AssertEqual("grade courses class_scores score", expected.Score, actual.Score) {
		return errInvalidResponse("成績確認のクラスのスコアが一致しません")
	}

	if !AssertEqual("grade courses class_scores submitters", expected.SubmitterCount, actual.Submitters) {
		return errInvalidResponse("成績確認のクラスの課題提出者の数が一致しません")
	}

	return nil
}
