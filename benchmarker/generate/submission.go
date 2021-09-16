package generate

import (
	"fmt"
	"math/rand"
	"strings"

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

	return PDF(genSubmissionContents(course, class, user), cyclicGetImage()), title
}

func genSubmissionContents(course *model.Course, class *model.Class, user *model.UserAccount) string {
	boolAnswers := []string{"o", "x"}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("%s part %d\n", course.Code, class.Part))
	content.WriteString(fmt.Sprintf("code %s\n\n", user.Code))
	for i := 0; i < 5; i++ {
		switch rand.Intn(2) {
		case 0:
			content.WriteString(fmt.Sprintf("Q%d. %s\n", i+1, randElt(boolAnswers)))
		case 1:
			content.WriteString(fmt.Sprintf("Q%d. %d\n", i+1, rand.Intn(1001)))
		}
	}

	return content.String()
}
