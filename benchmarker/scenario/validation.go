package scenario

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucandar/parallel"

	"github.com/isucon/isucon11-final/benchmarker/api"
	"github.com/isucon/isucon11-final/benchmarker/fails"
	"github.com/isucon/isucon11-final/benchmarker/generate"
	"github.com/isucon/isucon11-final/benchmarker/model"
	"github.com/isucon/isucon11-final/benchmarker/util"
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
	errTimeout := failure.NewError(fails.ErrCritical, fmt.Errorf("時間内に Announcement の検証が完了しませんでした"))
	errNotMatchUnreadCount := failure.NewError(fails.ErrCritical, fmt.Errorf("/api/announcements の unread_count の値が不正です"))
	errNotSorted := failure.NewError(fails.ErrCritical, fmt.Errorf("/api/announcements の順序が不正です"))
	errNotMatch := failure.NewError(fails.ErrCritical, fmt.Errorf("お知らせの内容が不正です"))
	errNotMatchOver := failure.NewError(fails.ErrCritical, fmt.Errorf("最終検証にて存在しないはずの Announcement が見つかりました"))
	errNotMatchUnder := failure.NewError(fails.ErrCritical, fmt.Errorf("最終検証にて存在するはずの Announcement が見つかりませんでした"))

	sampleCount := int64(float64(s.ActiveStudentCount()) * validateAnnouncementsRate)
	sampleIndices := generate.ShuffledInts(s.ActiveStudentCount())[:sampleCount]

	wg := sync.WaitGroup{}
	wg.Add(len(sampleIndices))
	for _, sampleIndex := range sampleIndices {
		student := s.activeStudents[sampleIndex]
		go func() {
			defer wg.Done()

			// 1〜5秒ランダムに待つ
			time.Sleep(time.Duration(rand.Int63n(5)+1) * time.Second)

			// responseに含まれるunread_count
			var responseUnreadCounts []int
			actualAnnouncements := map[string]api.AnnouncementResponse{}
			lastCreatedAt := int64(math.MaxInt64)

			timer := time.After(10 * time.Second)
			var next string
			for {
				hres, res, err := GetAnnouncementListAction(ctx, student.Agent, next)
				if err != nil {
					step.AddError(failure.NewError(fails.ErrCritical, err))
					return
				}

				// UnreadCount は各ページのレスポンスですべて同じ値が返ってくることを検証
				responseUnreadCounts = append(responseUnreadCounts, res.UnreadCount)
				if responseUnreadCounts[0] != res.UnreadCount {
					step.AddError(errNotMatchUnreadCount)
					return
				}

				for _, announcement := range res.Announcements {
					actualAnnouncements[announcement.ID] = announcement

					// 順序の検証
					if lastCreatedAt < announcement.CreatedAt {
						step.AddError(errNotSorted)
						return
					}
					lastCreatedAt = announcement.CreatedAt
				}

				_, next = parseLinkHeader(hres)
				if next == "" {
					break
				}

				select {
				case <-timer:
					step.AddError(errTimeout)
					break
				default:
				}
			}

			// レスポンスのunread_countの検証
			var actualUnreadCount int
			for _, a := range actualAnnouncements {
				if a.Unread {
					actualUnreadCount++
				}
			}
			if !AssertEqual("response unread count", actualUnreadCount, responseUnreadCounts[0]) {
				step.AddError(errNotMatchUnreadCount)
				return
			}

			expectAnnouncements := student.Announcements()
			for _, expectStatus := range expectAnnouncements {
				expect := expectStatus.Announcement
				actual, ok := actualAnnouncements[expect.ID]

				if !ok {
					AdminLogger.Printf("less announcements -> name: %v, title:  %v", actual.CourseName, actual.Title)
					step.AddError(errNotMatchUnder)
					return
				}

				// Dirtyフラグが立っていない場合のみ、Unreadの検証を行う
				// 既読化RequestがTimeoutで中断された際、ベンチには既読が反映しないがwebapp側が既読化される可能性があるため。
				if !expectStatus.Dirty {
					if !AssertEqual("announcement Unread", expectStatus.Unread, actual.Unread) {
						AdminLogger.Printf("unread mismatch -> name: %v, title:  %v", actual.CourseName, actual.Title)
						step.AddError(errNotMatch)
						return
					}
				}

				if !AssertEqual("announcement ID", expect.ID, actual.ID) ||
					!AssertEqual("announcement Code", expect.CourseID, actual.CourseID) ||
					!AssertEqual("announcement Title", expect.Title, actual.Title) ||
					!AssertEqual("announcement CourseName", expect.CourseName, actual.CourseName) ||
					!AssertEqual("announcement CreatedAt", expect.CreatedAt, actual.CreatedAt) {
					AdminLogger.Printf("announcement mismatch -> name: %v, title:  %v", actual.CourseName, actual.Title)
					step.AddError(errNotMatch)
					return
				}
			}

			if !AssertEqual("announcement len", len(expectAnnouncements), len(actualAnnouncements)) {
				// 上で expect が actual の部分集合であることを確認しているので、ここで数が合わない場合は actual の方が多い
				step.AddError(errNotMatchOver)
				return
			}

			expectMinUnread, expectMaxUnread := student.ExpectUnreadRange()
			if !AssertInRange("response unread count", expectMinUnread, expectMaxUnread, actualUnreadCount) {
				step.AddError(errNotMatchUnreadCount)
				return
			}
		}()
	}
	wg.Wait()
}

func (s *Scenario) validateCourses(ctx context.Context, step *isucandar.BenchmarkStep) {
	errNotMatchCount := failure.NewError(fails.ErrCritical, fmt.Errorf("最終検証にて登録されている Course の個数が一致しませんでした"))
	errNotMatch := failure.NewError(fails.ErrCritical, fmt.Errorf("最終検証にて存在しないはずの Course が見つかりました"))

	students := s.ActiveStudents()
	expectCourses := s.CourseManager.ExposeCoursesForValidation()

	if len(students) == 0 || len(expectCourses) == 0 {
		return
	}

	// searchAPIを叩くユーザ
	student := students[0]

	var actuals []*api.GetCourseDetailResponse
	// 空検索パラメータで全部ページング → 科目をすべて集める
	nextPathParam := "/api/courses"
	for nextPathParam != "" {
		hres, res, err := SearchCourseAction(ctx, student.Agent, nil, nextPathParam)
		if err != nil {
			step.AddError(failure.NewError(fails.ErrCritical, err))
			return
		}
		for _, c := range res {
			actuals = append(actuals, c)
		}

		_, nextPathParam = parseLinkHeader(hres)
	}

	if len(expectCourses) != len(actuals) {
		step.AddError(errNotMatchCount)
		return
	}

	for _, actual := range actuals {
		expect, ok := expectCourses[actual.ID.String()]
		if !ok {
			step.AddError(errNotMatch)
			return
		}

		if !AssertEqual("course ID", expect.ID, actual.ID.String()) ||
			!AssertEqual("course Code", expect.Code, actual.Code) ||
			!AssertEqual("course Name", expect.Name, actual.Name) ||
			!AssertEqual("course Type", api.CourseType(expect.Type), actual.Type) ||
			!AssertEqual("course Credit", uint8(expect.Credit), actual.Credit) ||
			!AssertEqual("course Teacher", expect.Teacher().Name, actual.Teacher) ||
			// webappは1-6, benchは0-5
			!AssertEqual("course Period", uint8(expect.Period+1), actual.Period) ||
			// webappはMonday..., benchは0-4
			!AssertEqual("course DayOfWeek", api.DayOfWeekTable[expect.DayOfWeek], actual.DayOfWeek) ||
			!AssertEqual("course Keywords", expect.Keywords, actual.Keywords) ||
			!AssertEqual("course Description", expect.Description, actual.Description) {
			step.AddError(errNotMatch)
			return
		}
	}
	return
}

func (s *Scenario) validateGrades(ctx context.Context, step *isucandar.BenchmarkStep) {
	activeStudents := s.activeStudents
	users := make(map[string]*model.Student, len(activeStudents))
	for _, activeStudent := range activeStudents {
		if activeStudent.HasFinishedCourse() {
			users[activeStudent.Code] = activeStudent
		}
	}

	p := parallel.NewParallel(ctx, int32(len(users)))
	// n回に1回validationする
	n := 10
	i := 0
	for _, user := range users {
		if i%n != 0 {
			i++
			continue
		} else {
			i++
		}
		user := user
		err := p.Do(func(ctx context.Context) {
			// 1〜5秒ランダムに待つ
			time.Sleep(time.Duration(rand.Int63n(5)+1) * time.Second)

			expected := calculateGradeRes(user, users)

			_, res, err := GetGradeAction(ctx, user.Agent)
			if err != nil {
				step.AddError(failure.NewError(fails.ErrCritical, err))
				return
			}

			err = validateUserGrade(&expected, &res)
			if err != nil {
				step.AddError(err)
				return
			}
		})
		if err != nil {
			panic(fmt.Errorf("unreachable! %w", err))
		}
	}

	p.Wait()

	return
}

func validateUserGrade(expected *model.GradeRes, actual *api.GetGradeResponse) error {
	if len(expected.CourseResults) != len(actual.CourseResults) {
		AdminLogger.Println("courseResult len. expected: ", len(expected.CourseResults), "actual: ", len(actual.CourseResults))
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認の courses の数が一致しません"))
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

	if math.Abs(expected.GPA-actual.GPA) > validateGPAErrorTolerance {
		AdminLogger.Println("gpa. expected: ", expected.GPA, "actual: ", actual.GPA)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のsummaryのgpaが一致しません"))
	}

	if math.Abs(expected.GpaAvg-actual.GpaAvg) > validateGPAErrorTolerance {
		AdminLogger.Println("gpaavg. expected: ", expected.GpaAvg, "actual: ", actual.GpaAvg)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のsummaryのgpa_avgが一致しません"))
	}

	if math.Abs(expected.GpaMax-actual.GpaMax) > validateGPAErrorTolerance {
		AdminLogger.Println("gpamax. expected: ", expected.GpaMax, "actual: ", actual.GpaMax)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のsummaryのgpa_maxが一致しません"))
	}

	if math.Abs(expected.GpaMin-actual.GpaMin) > validateGPAErrorTolerance {
		AdminLogger.Println("gpamin. expected: ", expected.GpaMin, "actual: ", actual.GpaMin)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のsummaryのgpa_minが一致しません"))
	}

	if math.Abs(expected.GpaTScore-actual.GpaTScore) > validateGPAErrorTolerance {
		AdminLogger.Println("gpatscore. expected: ", expected.GpaTScore, "actual: ", actual.GpaTScore)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のsummaryのgpa_t_scoreが一致しません"))
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

	if math.Abs(expected.TotalScoreAvg-actual.TotalScoreAvg) > validateTotalScoreErrorTolerance {
		AdminLogger.Println("TotalScoreAvg. expected: ", expected.TotalScoreAvg, "actual: ", actual.TotalScoreAvg)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のコースのTotalScoreAvgが一致しません"))
	}

	if math.Abs(expected.TotalScoreTScore-actual.TotalScoreTScore) > validateTotalScoreErrorTolerance {
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

	if !((expected.Score == nil && actual.Score == nil) ||
		((expected.Score != nil && actual.Score != nil) && (*expected.Score == *actual.Score))) {
		AdminLogger.Println("score. expected: ", expected.Score, "actual: ", actual.Score)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のクラスのスコアが一致しません"))
	}

	if expected.SubmitterCount != actual.Submitters {
		AdminLogger.Println("submitters. expected: ", expected.SubmitterCount, "actual: ", actual.Submitters)
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のクラスの課題提出者の数が一致しません"))
	}

	return nil
}

func calculateGradeRes(student *model.Student, students map[string]*model.Student) model.GradeRes {
	courses := student.Courses()
	courseResults := make(map[string]*model.CourseResult, len(courses))
	for _, course := range courses {
		result := course.CalcCourseResultByStudentCode(student.Code)
		if result == nil {
			panic("unreachable! userCode:" + student.Code)
		}

		courseResults[course.Code] = result
	}

	summary := calculateSummary(students, student.Code)
	return model.NewGradeRes(summary, courseResults)
}

// userCodeがstudentsの中にないとpanicしたり返り値が変な値になったりする
func calculateSummary(students map[string]*model.Student, userCode string) model.Summary {
	n := len(students)
	if n == 0 {
		panic("TODO: len (students) is 0")
	}

	if _, ok := students[userCode]; !ok {
		// ベンチのバグ用のloggerを作ったらそこに出すようにする
		// 呼び出し元が1箇所しか無くて、そこではstudentsのrangeをとってそのkeyをuserCode
		// に渡すので大丈夫なはず
		panic("unreachable! userCode: " + userCode)
	}

	gpas := make([]float64, 0, n)
	for _, student := range students {
		if student.HasFinishedCourse() {
			gpas = append(gpas, student.GPA())
		}
	}

	targetUserGpa := students[userCode].GPA()
	credits := students[userCode].TotalCredit()

	gpaAvg := util.AverageFloat64(gpas, 0)
	gpaMax := util.MaxFloat64(gpas, 0)
	gpaMin := util.MinFloat64(gpas, 0)
	gpaTScore := util.TScoreFloat64(targetUserGpa, gpas)

	return model.Summary{
		Credits:   credits,
		GPA:       targetUserGpa,
		GpaTScore: gpaTScore,
		GpaAvg:    gpaAvg,
		GpaMax:    gpaMax,
		GpaMin:    gpaMin,
	}
}
