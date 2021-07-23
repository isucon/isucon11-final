package generate

import (
	"fmt"
	"math/rand"
	"strconv"

	"github.com/isucon/isucon11-final/benchmarker/model"
)

func Course() *model.Course {
	randInt := rand.Intn(100)
	return model.NewCourse(fmt.Sprintf("サンプル講義%s", strconv.Itoa(randInt)))
}

func Submission() *model.Submission {
	title := "test title"
	data := []byte{1, 2, 3}
	return model.NewSubmission(title, data)
}
