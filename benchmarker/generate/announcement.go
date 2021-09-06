package generate

import (
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/isucon/isucon11-final/benchmarker/model"
	"github.com/oklog/ulid/v2"
)

var entropy io.Reader

func init() {
	entropy = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func Announcement(course *model.Course, class *model.Class) *model.Announcement {
	createdAt := GenTime()
	return &model.Announcement{
		ID:         ulid.MustNew(uint64(createdAt*1000), entropy).String(),
		CourseID:   course.ID,
		CourseName: course.Name,
		Title:      fmt.Sprintf("クラス追加: %s", class.Title),
		Message:    fmt.Sprintf("クラスが新しく追加されました: %s\n%s", class.Title, class.Desc),
		CreatedAt:  createdAt,
	}
}
