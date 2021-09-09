package generate

import (
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	entropy     = ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)
	entropyLock sync.Mutex
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GenULID() string {
	entropyLock.Lock()
	defer entropyLock.Unlock()
	return ulid.MustNew(ulid.Now(), entropy).String()
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
