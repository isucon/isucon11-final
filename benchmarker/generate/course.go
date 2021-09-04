package generate

import (
	"fmt"
	"math/rand"
	"strings"
	"sync/atomic"

	"github.com/isucon/isucon11-final/benchmarker/model"
)

const (
	majorCourseProb = 0.7
)

var (
	liberalCode int32
	majorCode   int32
)

var (
	majorPrefix = []string{
		"先進", "量子", "知能化", "機能的", "現代",
	}
	majorMid1 = []string{
		"コンピューティング", "コンピュータ", "プログラミング", "アルゴリズム", "ディジタル",
		"マネジメント", "言語", "コミュニケーション", "統計", "椅子", "生命", "バイオ",
	}
	majorMid2 = []string{
		"ネットワーク", "モデリング", "メカトロニクス", "デザイン", "システム", "サイエンス",
		"力学", "工学", "化学", "科学", "分析", "解析", "設計", "リテラシー",
	}
	majorSuffix = []string{
		"基礎", "応用", "演習",
		"導入", "概論", "特論", "理論",
		"第一", "第二",
		"A", "B", "C",
		"Ⅰ", "Ⅱ",
	}
	liberalMid1 = []string{
		"経済", "統計", "椅子", "法学", "哲学", "宗教", "政治", "人間文化", "社会",
		"図学", "芸術", "文学", "言語", "椅子",
	}
	liberalMid2 = []string{
		"概論", "基礎", "史", "モデリング", "デザイン", "システム", "サイエンス", "科学",
	}
	liberalSuffix = []string{
		"導入", "第一", "第二",
		"A", "B", "C",
	}
	courseDescription1 = []string{
		"本講義では課題提出をもって出席の代わりとする。",
		"本講義では出席を毎回取る。",
		"本講義では出席をランダムな講義回で取る。",
	}
	courseDescription2 = []string{
		"成績は課題の提出状況により判断する。",
		"成績は出席と課題の提出状況により判断する。",
	}
)

func courseDescription() string {
	return randElt(courseDescription1) + randElt(courseDescription2)
}

func majorCourseParam(dayOfWeek, period int, teacher *model.Teacher) *model.CourseParam {
	code := atomic.AddInt32(&majorCode, 1)

	var (
		name     strings.Builder
		keywords = make([]string, 2)
	)

	if rand.Float64() < 0.5 { // 確率は適当
		name.WriteString(randElt(majorPrefix))
	}

	mid1 := randElt(majorMid1)
	name.WriteString(mid1)
	keywords[0] = mid1

	mid2 := randElt(majorMid2)
	name.WriteString(mid2)
	keywords[1] = mid2

	name.WriteString(randElt(majorSuffix))

	return &model.CourseParam{
		Code:        fmt.Sprintf("M%04d", code), // 重複不可, L,M+4桁の数字
		Type:        "major-subjects",
		Name:        name.String(),
		Description: courseDescription(),
		Credit:      rand.Intn(3) + 1, // 1-3
		Teacher:     teacher.Name,
		Period:      period,
		DayOfWeek:   dayOfWeek,
		Keywords:    strings.Join(keywords, " "),
	}
}

func liberalCourseParam(dayOfWeek, period int, teacher *model.Teacher) *model.CourseParam {
	code := atomic.AddInt32(&liberalCode, 1)

	var (
		name     strings.Builder
		keywords = make([]string, 2)
	)

	mid1 := randElt(liberalMid1)
	name.WriteString(mid1)
	keywords[0] = mid1

	mid2 := randElt(liberalMid2)
	name.WriteString(mid2)
	keywords[1] = mid2

	name.WriteString(randElt(liberalSuffix))

	return &model.CourseParam{
		Code:        fmt.Sprintf("L%04d", code), // 重複不可, L,M+4桁の数字
		Type:        "liberal-arts",
		Name:        name.String(),
		Description: courseDescription(),
		Credit:      rand.Intn(3) + 1, // 1-3
		Teacher:     teacher.Name,
		Period:      period,
		DayOfWeek:   dayOfWeek,
		Keywords:    strings.Join(keywords, " "),
	}
}

func CourseParam(dayOfWeek, period int, teacher *model.Teacher) *model.CourseParam {
	if rand.Float64() < majorCourseProb {
		return majorCourseParam(dayOfWeek, period, teacher)
	} else {
		return liberalCourseParam(dayOfWeek, period, teacher)
	}
}

var searchRandEngine = rand.New(rand.NewSource(-981435))
var keywordList = append(majorMid2, liberalMid1...)
var popularTeacherName = []string{"橋本 陸", "金城 奈菜", "山下 篤", "森 奏太", "山口 和希", "高嶺 空", "池田 大地", "石川 楓花", "高橋 華子", "佐々木 諒", "荒井 優希", "加藤 海斗", "相澤 悠太", "近藤 太郎", "伊藤 千晶", "佐々木 翼", "佐藤 大樹"}

func SearchCourseParam() *model.SearchCourseParam {
	param := model.SearchCourseParam{
		Type:      "",
		Credit:    0,
		Teacher:   "",
		Period:    -1, // 0-5, -1で指定なし
		DayOfWeek: -1, // 0-4, -1で指定なし
		Keywords:  []string{},
	}

	if percentage(searchRandEngine, 1, 2) {
		// 1/2の確率でTimeSlot指定
		param.Period = rand.Intn(6)
		param.DayOfWeek = rand.Intn(5)
	} else if percentage(searchRandEngine, 1, 2) {
		// 1/4の確率でType指定
		if rand.Intn(2) == 0 {
			param.Type = "liberal-arts"
		} else {
			param.Type = "major-subjects"
		}
	} else if percentage(searchRandEngine, 1, 2) {
		// 1/8の確率でTeacher指定
		param.Teacher = randElt(popularTeacherName)
	} else {
		// 1/8の確率でKeyword指定
		param.Keywords = []string{randElt(keywordList)}
	}
	return &param
}

func percentage(engine *rand.Rand, decimal int, parameter int) bool {
	return engine.Intn(parameter) < decimal
}
