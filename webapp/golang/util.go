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

func contains(arr []DayOfWeek, elt DayOfWeek) bool {
	for _, s := range arr {
		if s == elt {
			return true
		}
	}
	return false
}

func averageInt(arr []int, or float64) float64 {
	if len(arr) == 0 {
		return or
	}
	var sum float64
	for _, elt := range arr {
		sum += float64(elt)
	}
	return sum / float64(len(arr))
}

func maxInt(arr []int, or int) int {
	if len(arr) == 0 {
		return or
	}
	max := -math.MinInt32
	for _, elt := range arr {
		if max < elt {
			max = elt
		}
	}
	return max
}

func minInt(arr []int, or int) int {
	if len(arr) == 0 {
		return or
	}
	min := math.MaxInt32
	for _, elt := range arr {
		if elt < min {
			min = elt
		}
	}
	return min
}

func stdDevInt(arr []int, avg float64) float64 {
	if len(arr) == 0 {
		return 0
	}
	var sdmSum float64
	for _, elt := range arr {
		sdmSum += math.Pow(float64(elt)-avg, 2)
	}
	return math.Sqrt(sdmSum / float64(len(arr)))
}

func averageFloat64(arr []float64, or float64) float64 {
	if len(arr) == 0 {
		return or
	}
	var sum float64
	for _, elt := range arr {
		sum += elt
	}
	return sum / float64(len(arr))
}

func maxFloat64(arr []float64, or float64) float64 {
	if len(arr) == 0 {
		return or
	}
	max := -math.MaxFloat64
	for _, elt := range arr {
		if max < elt {
			max = elt
		}
	}
	return max
}

func minFloat64(arr []float64, or float64) float64 {
	if len(arr) == 0 {
		return or
	}
	min := math.MaxFloat64
	for _, elt := range arr {
		if elt < min {
			min = elt
		}
	}
	return min
}

func stdDevFloat64(arr []float64, avg float64) float64 {
	if len(arr) == 0 {
		return 0
	}
	var sdmSum float64
	for _, elt := range arr {
		sdmSum += math.Pow(elt-avg, 2)
	}
	return math.Sqrt(sdmSum / float64(len(arr)))
}
