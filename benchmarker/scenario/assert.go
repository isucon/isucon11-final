package scenario

import (
	"math"
	"reflect"
)

func AssertEqual(msg string, expected interface{}, actual interface{}) bool {
	r := assertEqual(expected, actual)
	if !r {
		ContestantLogger.Printf("%s: 期待する値: %v / 実際の値: %v", msg, expected, actual)
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

func AssertInRange(msg string, expectMin, expectMax, actual int) bool {
	r := expectMin <= actual && actual <= expectMax
	if !r {
		ContestantLogger.Printf("%s: 期待する値: %d ~ %d / 実際の値: %d", msg, expectMin, expectMax, actual)
	}
	return r
}

func AssertAbsolute(msg string, expect, actual, tolerance float64) bool {
	r := math.Abs(expect-actual) > tolerance
	if !r {
		ContestantLogger.Printf("%s: 参考値: %f / 実際の値: %f / 許容誤差: %.2f", msg, expect, actual, tolerance)
	}
	return r
}
