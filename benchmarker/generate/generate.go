package generate

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/isucon/isucon11-final/benchmarker/model"
)

var (
	once          sync.Once
	timeGenerator *timer
)

func init() {
	newTimer()
}

func Course() *model.Course {
	randInt := rand.Intn(100)
	return model.NewCourse(fmt.Sprintf("サンプル講義%s", strconv.Itoa(randInt)))
}

func Submission() *model.Submission {
	title := "test title"
	data := []byte{1, 2, 3}
	return model.NewSubmission(title, data)
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
