package generate

import (
	"math"
	"math/rand"
)

const (
	max = 100
	min = 0

	stddev = 12.0
	mean   = 60.0
)

func Score() int {
	scoreFloat := rand.NormFloat64()*stddev + mean
	score := int(math.Round(scoreFloat))

	if score > max {
		score = 2*mean - score
	}
	if score < min {
		score = min
	}

	return score
}
