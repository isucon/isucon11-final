package generate

import (
	"fmt"
	"math/rand"
	"strconv"

	"github.com/isucon/isucon11-final/benchmarker/model"
)

func InitialStudents() []*model.Student {
	// 順序バラバラで初期データをロード
	// generateなのに初期データのロードしてるじゃんなのはそのうち...
	return []*model.Student{
		model.NewStudent("学籍番号110101", "test"),
		model.NewStudent("学籍番号110102", "test"),
		model.NewStudent("学籍番号110103", "test"),
		model.NewStudent("学籍番号110104", "test"),
		model.NewStudent("学籍番号110105", "test"),
		model.NewStudent("学籍番号110106", "test"),
		model.NewStudent("学籍番号110107", "test"),
	}
}

func Course() *model.Course {
	randInt := rand.Intn(100)
	return model.NewCourse(fmt.Sprintf("サンプル講義%s", strconv.Itoa(randInt)))
}
