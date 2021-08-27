package main

import (
	"math"
	"os"
)

func GetEnv(key, val string) string {
	if v := os.Getenv(key); v == "" {
		return val
	} else {
		return v
	}
}

func contains(arr []DayOfWeek, day DayOfWeek) bool {
	for _, v := range arr {
		if v == day {
			return true
		}
	}
	return false
}

func averageInt(arr []int, or float64) float64 {
	if len(arr) == 0 {
		return or
	}
	var sum int
	for _, v := range arr {
		sum += v
	}
	return float64(sum) / float64(len(arr))
}

func maxInt(arr []int, or int) int {
	if len(arr) == 0 {
		return or
	}
	max := math.MinInt32
	for _, v := range arr {
		if max < v {
			max = v
		}
	}
	return max
}

func minInt(arr []int, or int) int {
	if len(arr) == 0 {
		return or
	}
	min := math.MaxInt32
	for _, v := range arr {
		if v < min {
			min = v
		}
	}
	return min
}

func stdDevInt(arr []int, avg float64) float64 {
	if len(arr) == 0 {
		return 0
	}
	var sdmSum float64
	for _, v := range arr {
		sdmSum += math.Pow(float64(v)-avg, 2)
	}
	return math.Sqrt(sdmSum / float64(len(arr)))
}

func isAllEqualFloat64(arr []float64) bool {
	for _, v := range arr {
		if arr[0] != v {
			return false
		}
	}
	return true
}

func sumFloat64(arr []float64) float64 {
	// Kahan summation
	var sum, c float64
	for _, v := range arr {
		y := v + c
		t := sum + y
		c = y - (t - sum)
		sum = t
	}
	return sum
}

func averageFloat64(arr []float64, or float64) float64 {
	if len(arr) == 0 {
		return or
	}
	if isAllEqualFloat64(arr) {
		return arr[0]
	}
	return sumFloat64(arr) / float64(len(arr))
}

func maxFloat64(arr []float64, or float64) float64 {
	if len(arr) == 0 {
		return or
	}
	max := -math.MaxFloat64
	for _, v := range arr {
		if max < v {
			max = v
		}
	}
	return max
}

func minFloat64(arr []float64, or float64) float64 {
	if len(arr) == 0 {
		return or
	}
	min := math.MaxFloat64
	for _, v := range arr {
		if v < min {
			min = v
		}
	}
	return min
}

func stdDevFloat64(arr []float64, avg float64) float64 {
	if len(arr) == 0 {
		return 0
	}
	sdm := make([]float64, len(arr))
	for i, v := range arr {
		sdm[i] = math.Pow(v-avg, 2)
	}
	return math.Sqrt(sumFloat64(sdm) / float64(len(arr)))
}
