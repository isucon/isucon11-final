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

const (
	acceptableFloatError = 0.9
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
	activeStudents := s.activeStudents
	users := make([]*model.Student, 0, len(activeStudents))
	for _, activeStudent := range activeStudents {
		if len(activeStudent.Course()) > 0 {
			users = append(users, activeStudent)
		}
	}

	p := parallel.NewParallel(ctx, int32(len(users)))

	for _, user := range users {
		p.Do(func(ctx context.Context) {
			// 1〜5秒ランダムに待つ
			<-time.After(time.Duration(rand.Int63n(5)+1) * time.Second)

			courses := user.Course()
			courseResults := make(map[string]*model.CourseResult, len(courses))
			for _, course := range courses {
				result := course.IntoCourseResult(user.Code)
				if result != nil {
					courseResults[course.Code] = result
				}
			}

			summary := calculateSummary(users, user.Code)
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

	if math.Abs(expected.GPA-actual.GPA) > acceptableFloatError {
		AdminLogger.Println("gpa. expected: ", expected.GPA, "actual: ", actual.GPA)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のsummaryのgpaが一致しません"))
	}

	if math.Abs(expected.GpaAvg-actual.GpaAvg) > acceptableFloatError {
		AdminLogger.Println("gpaavg. expected: ", expected.GpaAvg, "actual: ", actual.GpaAvg)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のsummaryのgpaAvgが一致しません"))
	}

	if math.Abs(expected.GpaMax-actual.GpaMax) > acceptableFloatError {
		AdminLogger.Println("gpamax. expected: ", expected.GpaMax, "actual: ", actual.GpaMax)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のsummaryのgpaMaxが一致しません"))
	}

	if math.Abs(expected.GpaMin-actual.GpaMin) > acceptableFloatError {
		AdminLogger.Println("gpamin. expected: ", expected.GpaMin, "actual: ", actual.GpaMin)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のsummaryのgpaMinが一致しません"))
	}

	if math.Abs(expected.GpaTScore-actual.GpaTScore) > acceptableFloatError {
		AdminLogger.Println("gpatscore. expected: ", expected.GpaTScore, "actual: ", actual.GpaTScore)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のsummaryのgpaTScoreが一致しません"))
	}

	return nil
}

func validateCourseResult(expected *model.CourseResult, actual *api.CourseResult) error {
	if expected.Name != actual.Name {
		AdminLogger.Println("name. expected: ", expected.Name, "actual: ", actual.Name)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のコースの名前が一致しません"))
	}

	if expected.Code != actual.Code {
		AdminLogger.Println("code. expected: ", expected.Code, "actual: ", actual.Code)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のコースのコードが一致しません"))
	}

	if expected.TotalScore != actual.TotalScore {
		AdminLogger.Println("TotalScore. expected: ", expected.TotalScore, "actual: ", actual.TotalScore)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のコースのTotalScoreが一致しません"))
	}

	if expected.TotalScoreMax != actual.TotalScoreMax {
		AdminLogger.Println("TotalScoreMax. expected: ", expected.TotalScoreMax, "actual: ", actual.TotalScoreMax)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のコースのTotalScoreMaxが一致しません"))
	}

	if expected.TotalScoreMin != actual.TotalScoreMin {
		AdminLogger.Println("TotalScoreMin. expected: ", expected.TotalScoreMin, "actual: ", actual.TotalScoreMin)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のコースのTotalScoreMinが一致しません"))
	}

	if math.Abs(expected.TotalScoreAvg-actual.TotalScoreAvg) > acceptableFloatError {
		AdminLogger.Println("TotalScoreAvg. expected: ", expected.TotalScoreAvg, "actual: ", actual.TotalScoreAvg)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のコースのTotalScoreAvgが一致しません"))
	}

	if math.Abs(expected.TotalScoreTScore-actual.TotalScoreTScore) > acceptableFloatError {
		AdminLogger.Println("TotalScoreTScore. expected: ", expected.TotalScoreTScore, "actual: ", actual.TotalScoreTScore)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のコースのTotalScoreTScoreが一致しません"))
	}

	if len(expected.ClassScores) != len(actual.ClassScores) {
		AdminLogger.Println("len ClassScores. expected: ", len(expected.ClassScores), "actual: ", len(actual.ClassScores))
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
		AdminLogger.Println("classID. expected: ", expected.ClassID, "actual: ", actual.ClassID)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のクラスのIDが一致しません"))
	}

	if expected.Part != actual.Part {
		AdminLogger.Println("part. expected: ", expected.Part, "actual: ", actual.Part)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のクラスのpartが一致しません"))
	}

	if expected.Title != actual.Title {
		AdminLogger.Println("title. expected: ", expected.Title, "actual: ", actual.Title)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のクラスのタイトルが一致しません"))
	}

	if expected.Score != actual.Score {
		AdminLogger.Println("score. expected: ", expected.Score, "actual: ", actual.Score)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のクラスのスコアが一致しません"))
	}

	if expected.SubmitterCount != actual.Submitters {
		AdminLogger.Println("submitters. expected: ", expected.SubmitterCount, "actual: ", actual.Submitters)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のクラスの課題の提出者の数が一致しません"))
	}

	return nil
}

func calculateSummary(students []*model.Student, userCode string) model.Summary {
	n := len(students)
	if n == 0 {
		panic("TODO: len (students) is 0")
	}

	gpas := make([]float64, n)

	targetUserGpa := 0.0
	credits := 0
	// activeStudentsをmapにするときは順番が保証されないことに注意
	for i, student := range students {
		if student.Code == userCode {
			targetUserGpa = student.GPA()
			gpas[i] = targetUserGpa
			credits = student.TotalCredit()
		} else {
			gpas[i] = student.GPA()
		}
	}

	//if targetUserGpa == 0.0 {
	//	panic("TODO: gpa is 0")
	//}

	gpaSum := 0.0
	gpaMax := 0.0
	gpaMin := math.MaxFloat64
	for _, gpa := range gpas {
		gpaSum += gpa

		if gpaMax < gpa {
			gpaMax = gpa
		}

		if gpaMin > gpa {
			gpaMin = gpa
		}
	}

	gpaAvg := gpaSum / float64(n)

	gpaStdDev := 0.0
	for _, gpa := range gpas {
		gpaStdDev += math.Pow(gpa-gpaAvg, 2) / float64(n)
	}

	gpaTScore := 0.0
	if gpaStdDev == 0 {
		gpaTScore = 50
	} else {
		gpaTScore = 10*(targetUserGpa-gpaAvg)/gpaStdDev + 50
	}

	return model.Summary{
		Credits:   credits,
		GPA:       targetUserGpa,
		GpaTScore: gpaTScore,
		GpaAvg:    gpaAvg,
		GpaMax:    gpaMax,
		GpaMin:    gpaMin,
	}
}
