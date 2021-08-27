package generate

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/isucon/isucon11-final/benchmarker/model"
)

var (
	classRooms = []string{
		"H101", "S323", "S423", "S512", "S513", "W933", "M011",
	}
	classDescription1 = []string{
		"課題は講義の内容について300字以下でまとめてください。",
		"課題は講義の内容についてあなたが調べたことについて500字以上1000字以内でまとめてください。",
		"課題は講義中に出題するクイズへの回答を提出してください。",
	}
)

func classDescription(course *model.Course, part uint8) string {
	var desc strings.Builder
	switch rand.Intn(10) {
	case 0:
		desc.WriteString(fmt.Sprintf("%s 第%d回の講義です。", course.Name, part))
	case 1:
		desc.WriteString(fmt.Sprintf("%s 第%s回の講義です。", course.Name, convertKanjiNumbers(part)))
	case 2:
		desc.WriteString(fmt.Sprintf("%s の講義です。", course.Name))
	case 3:
		desc.WriteString(fmt.Sprintf("第%d回の講義です。", part))
	case 4:
		desc.WriteString(fmt.Sprintf("第%s回の講義です。", convertKanjiNumbers(part)))
	}
	switch rand.Intn(5) {
	case 0:
		desc.WriteString(fmt.Sprintf("今日のMTG-Room IDは %03d-%03d-%04d です。", rand.Intn(1000), rand.Intn(1000), rand.Intn(10000)))
	case 1:
		desc.WriteString(fmt.Sprintf("今日の講義室は %s です。", randElt(classRooms)))
	}
	desc.WriteString(randElt(classDescription1))
	return desc.String()
}

func ClassParam(course *model.Course, part uint8) *model.ClassParam {
	var title string
	switch rand.Intn(5) {
	case 0:
		title = fmt.Sprintf("%s 第%d回講義", course.Name, part)
	case 1:
		title = fmt.Sprintf("%s 第%s回講義", course.Name, convertKanjiNumbers(part))
	case 2:
		title = fmt.Sprintf("第%d回講義", part)
	case 3:
		title = fmt.Sprintf("第%s回講義", convertKanjiNumbers(part))
	case 4:
		title = "新規講義"
	}

	desc := classDescription(course, part)
	return &model.ClassParam{
		Title: title,
		Desc:  desc,
		Part:  part,
	}
}
