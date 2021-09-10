package scenario

import (
	"context"
	"fmt"
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
	errNotMatchUnreadCountAmongPages := failure.NewError(fails.ErrCritical, fmt.Errorf("/api/announcements の各ページの unread_count の値が一致しません"))
	errNotMatchUnreadCount := failure.NewError(fails.ErrCritical, fmt.Errorf("/api/announcements の unread_count の値が不正です"))
	errNotSorted := failure.NewError(fails.ErrCritical, fmt.Errorf("/api/announcements の順序が不正です"))
	errNotMatch := failure.NewError(fails.ErrCritical, fmt.Errorf("お知らせの内容が不正です"))
	errNotMatchOver := failure.NewError(fails.ErrCritical, fmt.Errorf("最終検証にて存在しないはずの Announcement が見つかりました"))
	errNotMatchUnder := failure.NewError(fails.ErrCritical, fmt.Errorf("最終検証にて存在するはずの Announcement が見つかりませんでした"))
	errDuplicated := failure.NewError(fails.ErrCritical, fmt.Errorf("最終検証にて重複したIDの Announcement が見つかりました"))

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
			responseUnreadCounts := make([]int, 0)
			actualAnnouncements := make([]api.AnnouncementResponse, 0)
			actualAnnouncementsMap := make(map[string]api.AnnouncementResponse)

			timer := time.After(10 * time.Second)
			var next string
			for {
				hres, res, err := GetAnnouncementListAction(ctx, student.Agent, next)
				if err != nil {
					step.AddError(failure.NewError(fails.ErrCritical, err))
					return
				}

				responseUnreadCounts = append(responseUnreadCounts, res.UnreadCount)
				actualAnnouncements = append(actualAnnouncements, res.Announcements...)
				for _, a := range res.Announcements {
					actualAnnouncementsMap[a.ID] = a
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

			// UnreadCount は各ページのレスポンスですべて同じ値が返ってくることを検証
			for _, unreadCount := range responseUnreadCounts {
				if responseUnreadCounts[0] != unreadCount {
					step.AddError(errNotMatchUnreadCountAmongPages)
					return
				}
			}

			// 順序の検証
			for i := 0; i < len(actualAnnouncements)-1; i++ {
				if actualAnnouncements[i].ID < actualAnnouncements[i+1].ID {
					step.AddError(errNotSorted)
					return
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

			// actual の重複確認
			existingID := make(map[string]struct{}, len(actualAnnouncements))
			for _, a := range actualAnnouncements {
				if _, ok := existingID[a.ID]; ok {
					step.AddError(errDuplicated)
					return
				}
				existingID[a.ID] = struct{}{}
			}

			expectAnnouncements := student.Announcements()
			for _, expectStatus := range expectAnnouncements {
				expect := expectStatus.Announcement
				actual, ok := actualAnnouncementsMap[expect.ID]

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
					!AssertEqual("announcement CourseName", expect.CourseName, actual.CourseName) {
					AdminLogger.Printf("announcement mismatch -> name: %v, title:  %v", actual.CourseName, actual.Title)
					step.AddError(errNotMatch)
					return
				}
			}

			if !AssertEqual("announcement len", len(expectAnnouncements), len(actualAnnouncements)) {
				// 上で expect が actual の部分集合であることを確認しているので、ここで数が合わない場合は actual の方が多い
				AdminLogger.Printf("announcement len mismatch -> code: %v", student.Code)
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
	for _, c := range s.initCourse {
		expectCourses[c.ID] = c
	}

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

	if !AssertEqual("course count", len(expectCourses), len(actuals)) {
		step.AddError(errNotMatchCount)
		return
	}

	for _, actual := range actuals {
		expect, ok := expectCourses[actual.ID]
		if !ok {
			step.AddError(errNotMatch)
			return
		}

		if !AssertEqual("course ID", expect.ID, actual.ID) ||
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
	if !AssertEqual("grade courses length", len(expected.CourseResults), len(actual.CourseResults)) {
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
	if !AssertEqual("grade summary credits", expected.Credits, actual.Credits) {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のsummaryのcreditsが一致しません"))
	}

	if !AssertWithinTolerance("grade summary gpa", expected.GPA, actual.GPA, validateGPAErrorTolerance) {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のsummaryのgpaが一致しません"))
	}

	if !AssertWithinTolerance("grade summary gpa_avg", expected.GpaAvg, actual.GpaAvg, validateGPAErrorTolerance) {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のsummaryのgpa_avgが一致しません"))
	}

	if !AssertWithinTolerance("grade summary gpa_max", expected.GpaMax, actual.GpaMax, validateGPAErrorTolerance) {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のsummaryのgpa_maxが一致しません"))
	}

	if !AssertWithinTolerance("grade summary gpa_min", expected.GpaMin, actual.GpaMin, validateGPAErrorTolerance) {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のsummaryのgpa_minが一致しません"))
	}

	if !AssertWithinTolerance("grade summary gpa_t_score", expected.GpaTScore, actual.GpaTScore, validateGPAErrorTolerance) {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のsummaryのgpa_t_scoreが一致しません"))
	}

	return nil
}

func validateCourseResult(expected *model.CourseResult, actual *api.CourseResult) error {
	if !AssertEqual("grade courses name", expected.Name, actual.Name) {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のコースの名前が一致しません"))
	}

	if !AssertEqual("grade courses code", expected.Code, actual.Code) {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のコースのコードが一致しません"))
	}

	if !AssertEqual("grade courses total_score", expected.TotalScore, actual.TotalScore) {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のコースのTotalScoreが一致しません"))
	}

	if !AssertEqual("grade courses total_score_max", expected.TotalScoreMax, actual.TotalScoreMax) {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のコースのTotalScoreMaxが一致しません"))
	}

	if !AssertEqual("grade courses total_score_min", expected.TotalScoreMin, actual.TotalScoreMin) {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のコースのTotalScoreMinが一致しません"))
	}

	if !AssertWithinTolerance("grade courses total_score_avg", expected.TotalScoreAvg, actual.TotalScoreAvg, validateTotalScoreErrorTolerance) {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のコースのTotalScoreAvgが一致しません"))
	}

	if !AssertWithinTolerance("grade courses total_score_t_score", expected.TotalScoreTScore, actual.TotalScoreTScore, validateTotalScoreErrorTolerance) {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のコースのTotalScoreTScoreが一致しません"))
	}

	if !AssertEqual("grade courses class_scores length", len(expected.ClassScores), len(actual.ClassScores)) {
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
	if !AssertEqual("grade courses class_scores class_id", expected.ClassID, actual.ClassID) {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のクラスのIDが一致しません"))
	}

	if !AssertEqual("grade courses class_scores part", expected.Part, actual.Part) {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のクラスのpartが一致しません"))
	}

	if !AssertEqual("grade courses class_scores title", expected.Title, actual.Title) {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のクラスのタイトルが一致しません"))
	}

	if !AssertEqual("grade courses class_scores score", expected.Score, actual.Score) {
		return failure.NewError(fails.ErrCritical, errInvalidResponse("成績確認のクラスのスコアが一致しません"))
	}

	if !AssertEqual("grade courses class_scores submitters", expected.SubmitterCount, actual.Submitters) {
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
