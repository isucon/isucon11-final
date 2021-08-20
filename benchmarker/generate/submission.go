package generate

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/isucon/isucon11-final/benchmarker/model"
)

var docxFiles [][]byte

func init() {
	// TODO: initで事前に用意したinvalidなデータを読み込んでおく
	docxFiles[0] = []byte{0x50, 0x4b, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	docxFiles[1] = []byte{0x50, 0x4b, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	docxFiles[2] = []byte{0x50, 0x4b, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
}

func Submission(course *model.Course, class *model.Class, user *model.UserAccount) *model.Submission {
	tRand := rand.Float32()

	var title string
	if tRand > 0.7 {
		title = fmt.Sprintf("%s_%s_%s.pdf", course.Name, class.Title, user.Code)
	} else if tRand > 0.4 {
		title = fmt.Sprintf("%s_%s.pdf", class.Title, user.Name)
	} else {
		title = fmt.Sprintf("%s.pdf", user.Code)
	}

	return &model.Submission{
		Title: title,
		Data:  PDF(genSubmissionContents()),
		Valid: true,
	}
}

func InvalidSubmission(course *model.Course, class *model.Class, user *model.UserAccount) *model.Submission {
	tRand := rand.Float32()
	var title string
	if tRand > 0.7 {
		title = fmt.Sprintf("%s_%s_%s.docx", course.Name, class.Title, user.Code)
	} else if tRand > 0.4 {
		title = fmt.Sprintf("%s_%s.docx", class.Title, user.Name)
	} else {
		title = fmt.Sprintf("%s.docx", user.Code)
	}

	return &model.Submission{
		Title: title,
		Data:  docxFiles[rand.Intn(len(docxFiles))],
		Valid: true,
	}
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
