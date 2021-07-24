package generate

import (
	"fmt"
	"math/rand"
	"strconv"

	"github.com/isucon/isucon11-final/benchmarker/model"
)

func Course(faculty *model.Faculty) *model.Course {
	randInt := rand.Intn(100)
	param := &model.CourseParam{
		Type:      "L",
		Name:      "サンプル講義",
		Credit:    1,
		Teacher:   "先生A",
		Period:    1,
		DayOfWeek: "monday",
		Keywords:  "hoge hoge",
	}
	return model.NewCourse(fmt.Sprintf("サンプル講義%s", strconv.Itoa(randInt)), param, faculty)
}

func Submission() *model.Submission {
	title := "test title"
	data := []byte{1, 2, 3}
	return model.NewSubmission(title, data)
}
