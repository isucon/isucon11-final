package generate

import "math/rand"

func ShuffledInts(length int) []int {
	randSlice := make([]int, length)
	for i := 0; i < length; i++ {
		randSlice[i] = i
	}

	for i := len(randSlice) - 1; i >= 0; i-- {
		j := rand.Intn(i + 1)
		randSlice[i], randSlice[j] = randSlice[j], randSlice[i]
	}
	return randSlice
}
