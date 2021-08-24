package scenario

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/fails"

	"github.com/isucon/isucon11-final/benchmarker/api"

	"github.com/isucon/isucon11-final/benchmarker/model"

	"github.com/isucon/isucandar/parallel"

	"github.com/isucon/isucandar"
)

func (s *Scenario) Validation(ctx context.Context, step *isucandar.BenchmarkStep) error {
	if s.NoLoad {
		return nil
	}
	ContestantLogger.Printf("===> VALIDATION")

	s.validateAnnouncements(ctx, step)
	s.validateCourses(ctx, step)
	s.validateGrades(ctx, step)

	return nil
}

func (s *Scenario) validateAnnouncements(ctx context.Context, step *isucandar.BenchmarkStep) {
	return
}

func (s *Scenario) validateCourses(ctx context.Context, step *isucandar.BenchmarkStep) {

	return
}

func (s *Scenario) validateGrades(ctx context.Context, step *isucandar.BenchmarkStep) {
	users := s.activeStudents
	AdminLogger.Println("active students", len(users))

	p := parallel.NewParallel(ctx, int32(len(users)))

	for _, user := range users {
		p.Do(func(ctx context.Context) {
			// 1〜5秒ランダムに待つ
			<-time.After(time.Duration(rand.Int63n(5)+1) * time.Second)

			courses := user.Course()
			AdminLogger.Println("courses", len(courses))
			courseResults := make(map[string]*model.CourseResult, len(courses))
			for _, course := range courses {
				result := course.IntoCourseResult(user.Code)
				if result != nil {
					courseResults[course.Code] = result
				}
			}

			summary := calculateSummary(s.activeStudents, user.Code)
			expected := model.NewGradeRes(summary, courseResults)

			_, res, err := GetGradeAction(ctx, user.Agent)
			if err != nil {
				step.AddError(err)
				return
			}

			err = validateUserGrade(&expected, &res)
			if err != nil {
				step.AddError(err)
				return
			}
		})
	}

	p.Wait()

	return
}

func validateUserGrade(expected *model.GradeRes, actual *api.GetGradeResponse) error {
	if len(expected.CourseResults) != len(actual.CourseResults) {
		AdminLogger.Println("courseResult len. expected: ", len(expected.CourseResults), "actual: ", len(actual.CourseResults))
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のcourseResultsの数が一致しません"))
	}

	err := validateSummary(&expected.Summary, &actual.Summary)
	if err != nil {
		return err
	}

	for _, courseResult := range actual.CourseResults {
		if _, ok := expected.CourseResults[courseResult.Code]; !ok {
			AdminLogger.Println(courseResult.Code, "は予期せぬコースです")
			return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認に意図しないcourseの結果が含まれています"))
		}

		expected := expected.CourseResults[courseResult.Code]
		err := validateCourseResult(expected, &courseResult)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateSummary(expected *model.Summary, actual *api.Summary) error {
	if expected.Credits != actual.Credits {
		AdminLogger.Println("credits. expected: ", expected.Credits, "actual: ", actual.Credits)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のsummaryのcreditsが一致しません"))
	}

	if expected.GPT != actual.GPT {
		AdminLogger.Println("gpt. expected: ", expected.GPT, "actual: ", actual.GPT)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のsummaryのgptが一致しません"))
	}

	if expected.GptAvg != actual.GptAvg {
		AdminLogger.Println("gptavg. expected: ", expected.GptAvg, "actual: ", actual.GptAvg)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のsummaryのGptAvgが一致しません"))
	}

	if expected.GptMax != actual.GptMax {
		AdminLogger.Println("gptmax. expected: ", expected.GptMax, "actual: ", actual.GptMax)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のsummaryのgptMaxが一致しません"))
	}

	if expected.GptMin != actual.GptMin {
		AdminLogger.Println("gptmin. expected: ", expected.GptMin, "actual: ", actual.GptMin)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のsummaryのgptMinが一致しません"))
	}

	if expected.GptTScore != actual.GptTScore {
		AdminLogger.Println("gpttscore. expected: ", expected.GptTScore, "actual: ", actual.GptTScore)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のsummaryのgptTScoreが一致しません"))
	}

	return nil
}

func validateCourseResult(expected *model.CourseResult, actual *api.CourseResult) error {
	if expected.Name != actual.Name {
		AdminLogger.Println("name. expected: ", expected.Name, "actual: ", actual.Name)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のコースの名前が一致しません"))
	}

	if expected.Code != actual.Code {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のコースのコードが一致しません"))
	}

	if expected.TotalScore != actual.TotalScore {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のコースのTotalScoreが一致しません"))
	}

	if expected.TotalScoreAvg != actual.TotalScoreAvg {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のコースのTotalScoreAvgが一致しません"))
	}

	if expected.TotalScoreMax != actual.TotalScoreMax {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のコースのTotalScoreMaxが一致しません"))
	}

	if expected.TotalScoreMin != actual.TotalScoreMin {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のコースのTotalScoreMinが一致しません"))
	}

	if expected.TotalScoreTScore != actual.TotalScoreTScore {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のコースのTotalScoreTScoreが一致しません"))
	}

	if len(expected.ClassScores) != len(actual.ClassScores) {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のClassScoresの数が一致しません"))
	}

	for i := 0; i < len(expected.ClassScores); i++ {
		// webapp 側は新しい(partが大きい)classから順番に帰ってくるので古いクラスから見るようにしている
		err := validateClassScore(expected.ClassScores[i], &actual.ClassScores[len(actual.ClassScores)-i-1])
		if err != nil {
			return err
		}
	}

	return nil
}

func validateClassScore(expected *model.ClassScore, actual *api.ClassScore) error {
	if expected.ClassID != actual.ClassID {
		AdminLogger.Println("classid. expected: ", expected.ClassID, "actual: ", actual.ClassID)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のクラスのIDが一致しません"))
	}

	if expected.Part != actual.Part {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のクラスのpartが一致しません"))
	}

	if expected.Title != actual.Title {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のクラスのタイトルが一致しません"))
	}

	if expected.Score != actual.Score {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のクラスのスコアが一致しません"))
	}

	if expected.SubmitterCount != actual.Submitters {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のクラスの課題の提出者の数が一致しません"))
	}

	return nil
}

func calculateSummary(activeStudents []*model.Student, userCode string) model.Summary {
	n := len(activeStudents)
	if n == 0 {
		panic("TODO: len (active student) is 0")
	}

	gpts := make([]float64, n)

	targetUserGpt := 0.0
	credits := 0
	// activeStudentsをmapにするときは順番が保証されないことに注意
	for i, student := range activeStudents {
		if student.Code == userCode {
			targetUserGpt = student.GPT()
			gpts[i] = targetUserGpt
			credits = student.TotalCredit()
		} else {
			gpts[i] = student.GPT()
		}
	}

	//if targetUserGpt == 0.0 {
	//	panic("TODO: gpt is 0")
	//}

	gptSum := 0.0
	gptMax := 0.0
	gptMin := math.MaxFloat64
	for _, gpt := range gpts {
		gptSum += gpt

		if gptMax < gpt {
			gptMax = gpt
		}

		if gptMin > gpt {
			gptMin = gpt
		}
	}

	gptAvg := gptSum / float64(n)

	gptStdDev := 0.0
	for _, gpt := range gpts {
		gptStdDev += math.Pow(gpt-gptAvg, 2) / float64(n)
	}

	gptTScore := 0.0
	if gptStdDev == 0 {
		gptTScore = 50
	} else {
		gptTScore = 10*(targetUserGpt-gptAvg)/gptStdDev + 50
	}

	return model.Summary{
		Credits:   credits,
		GPT:       targetUserGpt,
		GptTScore: gptTScore,
		GptAvg:    gptAvg,
		GptMax:    gptMax,
		GptMin:    gptMin,
	}
}
