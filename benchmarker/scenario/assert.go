package scenario

import (
	"math"
	"reflect"
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
