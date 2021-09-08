package generate

import (
	"fmt"

	"github.com/isucon/isucon11-final/benchmarker/model"
)

func Announcement(course *model.Course, class *model.Class) *model.Announcement {
	return &model.Announcement{
		ID:         GenULID(),
		CourseID:   course.ID,
		CourseName: course.Name,
		Title:      fmt.Sprintf("クラス追加: %s", class.Title),
		Message:    fmt.Sprintf("クラスが新しく追加されました: %s\n%s", class.Title, class.Desc),
	}
}
