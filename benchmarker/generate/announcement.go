package generate

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/isucon/isucon11-final/benchmarker/model"
)

func Announcement(course *model.Course, class *model.Class) *model.Announcement {
	createdAt := GenTime()
	return &model.Announcement{
		ID:         ulid.MustNew(uint64(createdAt*1000), ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)).String(),
		CourseID:   course.ID,
		CourseName: course.Name,
		Title:      fmt.Sprintf("クラス追加: %s", class.Title),
		Message:    fmt.Sprintf("クラスが新しく追加されました: %s\n%s", class.Title, class.Desc),
		CreatedAt:  createdAt,
	}
}
