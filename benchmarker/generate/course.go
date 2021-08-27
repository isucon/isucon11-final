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

type Option func(param *model.CourseParam)

func WithDayOfWeek(d int) Option {
	return func(param *model.CourseParam) {
		param.DayOfWeek = d
	}
}

func WithPeriod(p int) Option {
	return func(param *model.CourseParam) {
		param.Period = p
	}
}

func courseDescription() string {
	return randElt(courseDescription1) + randElt(courseDescription2)
}

func majorCourseParam(teacher *model.Teacher, ops ...Option) *model.CourseParam {
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

	p := model.CourseParam{
		Code:        fmt.Sprintf("M%04d", code), // 重複不可, L,M+4桁の数字
		Type:        "major-subjects",
		Name:        name.String(),
		Description: courseDescription(),
		Credit:      rand.Intn(3) + 1, // 1-3
		Teacher:     teacher.Name,
		Period:      rand.Intn(6),     // いいカンジに分散
		DayOfWeek:   rand.Intn(5) + 1, // いいカンジに分散
		Keywords:    strings.Join(keywords, " "),
	}

	for _, option := range ops {
		option(&p)
	}

	return &p
}

func liberalCourseParam(teacher *model.Teacher, op ...Option) *model.CourseParam {
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

	p := model.CourseParam{
		Code:        fmt.Sprintf("L%04d", code), // 重複不可, L,M+4桁の数字
		Type:        "liberal-arts",
		Name:        name.String(),
		Description: courseDescription(),
		Credit:      rand.Intn(3) + 1, // 1-3
		Teacher:     teacher.Name,
		Period:      rand.Intn(6),     // いいカンジに分散
		DayOfWeek:   rand.Intn(5) + 1, // いいカンジに分散
		Keywords:    strings.Join(keywords, " "),
	}

	for _, option := range op {
		option(&p)
	}

	return &p
}

func CourseParam(teacher *model.Teacher, op ...Option) *model.CourseParam {
	if rand.Float64() < majorCourseProb {
		return majorCourseParam(teacher, op...)
	} else {
		return liberalCourseParam(teacher, op...)
	}
}

func SearchCourseParam() *model.SearchCourseParam {
	// FIXME: 検索パラメータ生成
	return &model.SearchCourseParam{
		Type:      "",
		Credit:    0,
		Teacher:   "",
		Period:    -1, // 0-6, -1で指定なし
		DayOfWeek: -1, // 0-7, -1で指定なし
		Keywords:  []string{},
	}
}
