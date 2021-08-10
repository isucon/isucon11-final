package generate

import (
	"math/rand"
	"strconv"
	"sync"
	"time"
)

var (
	once          sync.Once
	timeGenerator *timer
)

func init() {
	rand.Seed(time.Now().UnixNano())
	newTimer()
}

type timer struct {
	base  int64 // unix time
	count int64

	mu sync.Mutex
}

func newTimer() {
	once.Do(func() {
		timeGenerator = &timer{
			base:  time.Now().Unix(),
			count: 0,
			mu:    sync.Mutex{},
		}
	})
}

func GenTime() int64 {
	timeGenerator.mu.Lock()
	defer timeGenerator.mu.Unlock()

	timeGenerator.count++
	return timeGenerator.base + timeGenerator.count
}

func randElt(arr []string) string {
	return arr[rand.Intn(len(arr))]
}

var kanjiNumbers = [10]string{"〇", "一", "二", "三", "四", "五", "六", "七", "八", "九"}

func convertKanjiNumbers(n uint8) string {
	switch {
	case n < 10:
		return kanjiNumbers[n]
	default:
		return strconv.Itoa(int(n)) // fallback
	}
}
