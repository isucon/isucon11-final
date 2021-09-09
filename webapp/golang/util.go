package main

import (
	"math"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
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

var (
	entropy     = ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)
	entropyLock sync.Mutex
)

func newULID() string {
	entropyLock.Lock()
	defer entropyLock.Unlock()
	return ulid.MustNew(ulid.Now(), entropy).String()
}

// ----- int -----

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

func tScoreInt(v int, arr []int) float64 {
	avg := averageInt(arr, 0)
	stdDev := stdDevInt(arr, avg)
	if stdDev == 0 {
		return 50
	} else {
		return (float64(v)-avg)/stdDev*10 + 50
	}
}

// ----- float64 -----

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

func tScoreFloat64(v float64, arr []float64) float64 {
	if isAllEqualFloat64(arr) {
		return 50
	}
	avg := averageFloat64(arr, 0)
	stdDev := stdDevFloat64(arr, avg)
	if stdDev == 0 {
		// should be unreachable
		return 50
	} else {
		return (v-avg)/stdDev*10 + 50
	}
}
