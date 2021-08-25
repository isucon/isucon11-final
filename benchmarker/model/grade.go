package model

// CourseResultのうち計算しなくていいやつ
type SimpleCourseResult struct {
	Name              string // course name
	Code              string // course code
	SimpleClassScores []*SimpleClassScore
}

func NewSimpleCourseResult(name, code string, classScores []*SimpleClassScore) *SimpleCourseResult {
	return &SimpleCourseResult{
		Name:              name,
		Code:              code,
		SimpleClassScores: classScores,
	}

}

type SimpleClassScore struct {
	// 上3つの情報はclassから取得できるので無くてもいいかもしれない
	ClassID string
	Title   string
	Part    uint8

	Score int // 0 - 100点
}

func NewSimpleClassScore(class *Class, score int) *SimpleClassScore {
	return &SimpleClassScore{
		ClassID: class.ID,
		Title:   class.Title,
		Part:    class.Part,
		Score:   score,
	}
}

type GradeRes struct {
	Summary       Summary
	CourseResults map[string]*CourseResult
}

func NewGradeRes(summary Summary, courseResults map[string]*CourseResult) GradeRes {
	return GradeRes{
		Summary:       summary,
		CourseResults: courseResults,
	}
}

type Summary struct {
	Credits   int
	GPA       float64
	GpaTScore float64 // 偏差値
	GpaAvg    float64 // 平均値
	GpaMax    float64 // 最大値
	GpaMin    float64 // 最小値
}

type CourseResult struct {
	Name             string
	Code             string
	TotalScore       int
	TotalScoreTScore float64 // 偏差値
	TotalScoreAvg    float64 // 平均値
	TotalScoreMax    int     // 最大値
	TotalScoreMin    int     // 最小値
	ClassScores      []*ClassScore
}

type ClassScore struct {
	// 上3つの情報はclassから取得できるので無くてもいいかもしれない
	ClassID string
	Title   string
	Part    uint8

	Score          int // 0 - 100点
	SubmitterCount int
}
