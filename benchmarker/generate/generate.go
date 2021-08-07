package generate

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/isucon/isucon11-final/benchmarker/model"
)

const (
	majorCourseProb = 0.7
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

func randElt(arr []string) string {
	return arr[rand.Intn(len(arr))]
}

var (
	once          sync.Once
	timeGenerator *timer

	liberalCode int32
	majorCode   int32
)

func init() {
	rand.Seed(time.Now().UnixNano())
	newTimer()
}

func courseDescription() string {
	return randElt(courseDescription1) + randElt(courseDescription2)
}

func majorCourseParam(faculty *model.Faculty) *model.CourseParam {
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
		Teacher:     faculty.Name,
		Period:      rand.Intn(6),     // いいカンジに分散
		DayOfWeek:   rand.Intn(5) + 1, // いいカンジに分散
		Keywords:    strings.Join(keywords, " "),
	}
}

func liberalCourseParam(faculty *model.Faculty) *model.CourseParam {
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
		Teacher:     faculty.Name,
		Period:      rand.Intn(6),     // いいカンジに分散
		DayOfWeek:   rand.Intn(5) + 1, // いいカンジに分散
		Keywords:    strings.Join(keywords, " "),
	}
}

func CourseParam(faculty *model.Faculty) *model.CourseParam {
	if rand.Float64() < majorCourseProb {
		return majorCourseParam(faculty)
	} else {
		return liberalCourseParam(faculty)
	}
}

func Submission() *model.Submission {
	title := "test title"
	data := []byte{1, 2, 3}
	return model.NewSubmission(title, data)
}

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

var kanjiNumbers = []string{"〇", "一", "二", "三", "四", "五", "六", "七", "八", "九"}

func convertKanjiNumbers(n uint8) string {
	switch {
	case n < 10:
		return kanjiNumbers[n]
	default:
		return strconv.Itoa(int(n)) // fallback
	}
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
	createdAt := GenTime()
	return &model.ClassParam{
		Title:     title,
		Desc:      desc,
		Part:      part,
		CreatedAt: createdAt,
	}
}

type timer struct {
	base  int64 // unix time
	count int64

	mu sync.Mutex
}

func newTimer() {
	once.Do(func() {
		timeGenerator = &timer{
			base:  time.Now().Unix(),
			count: 0,
			mu:    sync.Mutex{},
		}
	})
}

func GenTime() int64 {
	timeGenerator.mu.Lock()
	defer timeGenerator.mu.Unlock()

	timeGenerator.count++
	return timeGenerator.base + timeGenerator.count
}
