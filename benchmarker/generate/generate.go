package generate

import (
	"fmt"

	"github.com/isucon/isucon11-final/benchmarker/model"
)

// FIXME: 全部いいカンジにする
func Announcement(classTitle string) (string, string) {
	title := fmt.Sprintf("%s 開講のおしらせ", classTitle)
	message := fmt.Sprintf("次回の講義はオンライン上で実施します。オンライン講義ルームのIDはT133FmDaです。")
	return title, message
}

func DocumentFile() []byte {
	return []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
}

func Assignment(classTitle string) (string, string) {
	return fmt.Sprintf("%s 課題1", classTitle), fmt.Sprintf("課題で出題した問題①から⑤までの解答を書いて提出せよ")
}

func Submission() (string, []byte) {
	title := "課題1.pdf"
	data := []byte{0x25, 0x50, 0x44, 0x46, 0x2D, 0x31, 0x2E, 0x34}
	return title, data
}

func ClassDetail(course *model.Course) (string, string) {
	heldCount := course.GetHeldClassCount() + 1

	title := fmt.Sprintf("第%d回講義_%s", heldCount, course.Name)
	desc := fmt.Sprintf("本回では%sについて初歩的な知識を解説する予定である。\n最後に課題を公開するため忘れず提出すること。", course.Name)
	return title, desc
}
