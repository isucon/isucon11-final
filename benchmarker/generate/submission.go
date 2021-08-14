package generate

import (
	"fmt"
	"math/rand"

	"github.com/isucon/isucon11-final/benchmarker/model"
)

func Submission(course *model.Course, class *model.Class, user *model.UserAccount) *model.Submission {
	rand := rand.Float32()

	var title string
	var data []byte
	isValidExtension := rand > 0.9

	if isValidExtension {
		if rand > 0.7 {
			title = fmt.Sprintf("%s_%s_%s.pdf", course.Name, class.Title, user.Code)
		} else if rand > 0.4 {
			title = fmt.Sprintf("%s_%s.pdf", class.Title, user.Name)
		} else {
			title = fmt.Sprintf("%s.pdf", user.Code)
		}
		data = PDF(genSubmissionContents())
	} else {
		// TODO: 予め用意したvalidでないファイルを使用する
		title = "invalid.word"
		data = []byte{}
	}
	return &model.Submission{
		Title: title,
		Data:  data,
		Valid: isValidExtension,
	}
}

// TODO: いい感じにする
func genSubmissionContents() string {
	return "1: true\n" +
		"2: false\n" +
		"3: false\n" +
		"4: true\n" +
		"5: true"
}
