package generate

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/isucon/isucon11-final/benchmarker/model"
)

var (
	once          sync.Once
	timeGenerator *timer

	liberalCode int32
	majorCode   int32
)

func init() {
	newTimer()
}

func CourseParam(faculty *model.Faculty) *model.CourseParam {
	code := atomic.AddInt32(&liberalCode, 1)
	return &model.CourseParam{
		Code:      fmt.Sprintf("L%04d", code), // 重複不可, L,M+4桁の数字
		Type:      "liberal-arts",             // or "major-subjects"
		Name:      fmt.Sprintf("サンプル講義%04d", code),
		Credit:    1, // 1~3
		Teacher:   faculty.Name,
		Period:    rand.Intn(6),     // いいカンジに分散
		DayOfWeek: rand.Intn(5) + 1, // いいカンジに分散
		Keywords:  "hoge hoge",
	}
}

func Submission() *model.Submission {
	title := "test title"
	data := []byte{1, 2, 3}
	return model.NewSubmission(title, data)
}

func ClassParam(part uint8) *model.ClassParam {
	title := "test title"
	desc := "test desc"
	createdAt := GenTime()
	return &model.ClassParam{
		Title:     title,
		Desc:      desc,
		Part:      part,
		CreatedAt: createdAt,
	}
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
