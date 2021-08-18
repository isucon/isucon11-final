package model

// CourseResultのうち計算しなくていいやつ
type SimpleCourseResult struct {
	Name        string // course name
	Code        string // course code
	ClassScores []*ClassScore
}

func NewSimpleCourseResult(name, code string, classScores []*ClassScore) *SimpleCourseResult {

	return &SimpleCourseResult{
		Name:        name,
		Code:        code,
		ClassScores: classScores,
	}

}

type ClassScore struct {
	// 上3つの情報はclassから取得できるので無くてもいいかもしれない
	ClassID string
	Title   string
	Part    uint8

	Score int // 0 - 100点
}

func NewClassScore(class *Class, score int) *ClassScore {
	return &ClassScore{
		ClassID: class.ID,
		Title:   class.Title,
		Part:    class.Part,
		Score:   score,
	}
}
