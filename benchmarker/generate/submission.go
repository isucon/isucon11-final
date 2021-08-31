package generate

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/isucon/isucon11-final/benchmarker/model"
)

func SubmissionData(course *model.Course, class *model.Class, user *model.UserAccount) ([]byte, string) {
	tRand := rand.Float32()

	var title string
	if tRand > 0.7 {
		title = fmt.Sprintf("%s_%s_%s.pdf", course.Name, class.Title, user.Code)
	} else if tRand > 0.4 {
		title = fmt.Sprintf("%s_%s.pdf", class.Title, user.Name)
	} else {
		title = fmt.Sprintf("%s.pdf", user.Code)
	}

	return PDF(genSubmissionContents()), title
}

// TODO: いい感じにする
func genSubmissionContents() string {
	return "1: true\n" +
		"2: false\n" +
		"3: false\n" +
		"4: true\n" +
		"5: true\n" +
		"timestamp: " + strconv.Itoa(int(time.Now().UnixNano())) // FIXME: hashを変えるための一時措置
}
