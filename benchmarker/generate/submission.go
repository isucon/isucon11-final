package generate

import "github.com/isucon/isucon11-final/benchmarker/model"

func Submission() *model.Submission {
	title := "test title"
	data := []byte{1, 2, 3}
	return model.NewSubmission(title, data)
}
