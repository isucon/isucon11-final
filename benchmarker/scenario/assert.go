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

func AssertInRange(msg string, expectMin, expectMax, actual int) bool {
	r := expectMin <= actual && actual <= expectMax
	if !r {
		AdminLogger.Printf("%s: expected: %d ~ %d / actual: %d", msg, expectMin, expectMax, actual)
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

func AssertEqualCourse(expected *model.Course, actual *api.GetCourseDetailResponse, verifyStatus bool) bool {
	return AssertEqual("course id", expected.ID, actual.ID) &&
		AssertEqual("course code", expected.Code, actual.Code) &&
		AssertEqual("course type", api.CourseType(expected.Type), actual.Type) &&
		AssertEqual("course name", expected.Name, actual.Name) &&
		AssertEqual("course description", expected.Description, actual.Description) &&
		AssertEqual("course credit", uint8(expected.Credit), actual.Credit) &&
		AssertEqual("course period", uint8(expected.Period+1), actual.Period) &&
		AssertEqual("course day_of_week", api.DayOfWeekTable[expected.DayOfWeek], actual.DayOfWeek) &&
		AssertEqual("course teacher", expected.Teacher().Name, actual.Teacher) &&
		(!verifyStatus || AssertEqual("course status", expected.Status(), actual.Status)) &&
		AssertEqual("course keywords", expected.Keywords, actual.Keywords)
}

func MatchCourse(course *model.Course, param *model.SearchCourseParam) bool {
	return (param.Type == "" || course.Type == param.Type) &&
		(param.Credit == 0 || course.Credit == param.Credit) &&
		(param.Teacher == "" || course.Teacher().Name == param.Teacher) &&
		(param.Period == -1 || course.Period == param.Period) &&
		(param.DayOfWeek == -1 || course.DayOfWeek == param.DayOfWeek) &&
		(containsAll(course.Name, param.Keywords) || containsAll(course.Keywords, param.Keywords)) &&
		(param.Status == "" || string(course.Status()) == param.Status)
}
