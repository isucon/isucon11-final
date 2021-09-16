package scenario

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/parallel"

	"github.com/isucon/isucon11-final/benchmarker/api"
	"github.com/isucon/isucon11-final/benchmarker/fails"
	"github.com/isucon/isucon11-final/benchmarker/generate"
	"github.com/isucon/isucon11-final/benchmarker/model"
	"github.com/isucon/isucon11-final/benchmarker/util"
)

const (
	validationRequestTime = 30 * time.Second
	validationTime        = 60 * time.Second
)

func (s *Scenario) Validation(parent context.Context, step *isucandar.BenchmarkStep) error {
	if s.NoLoad {
		return nil
	}
	ContestantLogger.Printf("===> VALIDATION")

	// これは 10 秒 + ベンチの計算分しかかからないので先にやる
	s.validateGrades(parent, step)

	ctx, cancel := context.WithTimeout(parent, validationTime)
	defer cancel()

	// それぞれ 30 秒以上かかったらリクエストをやめて通す
	s.validateAnnouncements(ctx, step, validationRequestTime)
	s.validateCourses(ctx, step, validationRequestTime)

	return nil
}

func errValidation(err error) error {
	return fails.ErrorCritical(fmt.Errorf("整合性チェックに失敗しました: %w", err))
}

func (s *Scenario) validateAnnouncements(ctx context.Context, step *isucandar.BenchmarkStep, requestTime time.Duration) {
	errNotMatchUnreadCountAmongPages := func(hres *http.Response) error {
		return errValidation(fails.ErrorInvalidResponse(errors.New("各ページの unread_count の値が一致しません"), hres))
	}
	errNotMatchUnreadCount := func(hres *http.Response) error {
		return errValidation(fails.ErrorInvalidResponse(errors.New("unread_count の値が不正です"), hres))
	}
	errNotSorted := func(hres *http.Response) error {
		return errValidation(fails.ErrorInvalidResponse(errors.New("お知らせの順序が不正です"), hres))
	}
	errNotMatch := func(hres *http.Response) error {
		return errValidation(fails.ErrorInvalidResponse(errors.New("お知らせの内容が不正です"), hres))
	}
	errNotMatchOver := func(hres *http.Response) error {
		return errValidation(fails.ErrorInvalidResponse(errors.New("存在しないはずのお知らせが見つかりました"), hres))
	}
	errNotMatchUnder := func(hres *http.Response) error {
		return errValidation(fails.ErrorInvalidResponse(errors.New("存在するはずのお知らせが見つかりませんでした"), hres))
	}
	errDuplicated := func(hres *http.Response) error {
		return errValidation(fails.ErrorInvalidResponse(errors.New("重複したIDのお知らせが見つかりました"), hres))
	}

	endTime := time.Now().Add(requestTime)
	isNoRequestTime := func(ctx context.Context) bool {
		return time.Now().After(endTime) || ctx.Err() != nil
	}

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

			var hresSample *http.Response
			var next string
			couldSeeAll := false
			for {
				if isNoRequestTime(ctx) {
					break
				}

				hres, res, err := GetAnnouncementListAction(ctx, student.Agent, next, "")
				if err != nil {
					step.AddError(errValidation(err))
					return
				}

				responseUnreadCounts = append(responseUnreadCounts, res.UnreadCount)
				actualAnnouncements = append(actualAnnouncements, res.Announcements...)
				for _, a := range res.Announcements {
					actualAnnouncementsMap[a.ID] = a
				}

				hresSample = hres
				_, next, err = parseLinkHeader(hres)
				if err != nil {
					step.AddError(fails.ErrorCritical(err))
				}

				if next == "" {
					couldSeeAll = true
					break
				}

			}

			// UnreadCount は各ページのレスポンスですべて同じ値が返ってくることを検証
			for _, unreadCount := range responseUnreadCounts {
				if responseUnreadCounts[0] != unreadCount {
					step.AddError(errNotMatchUnreadCountAmongPages(hresSample))
					return
				}
			}

			// 順序の検証
			for i := 0; i < len(actualAnnouncements)-1; i++ {
				if actualAnnouncements[i].ID < actualAnnouncements[i+1].ID {
					step.AddError(errNotSorted(hresSample))
					return
				}
			}

			// actual の重複確認
			existingID := make(map[string]struct{}, len(actualAnnouncements))
			for _, a := range actualAnnouncements {
				if _, ok := existingID[a.ID]; ok {
					step.AddError(errDuplicated(hresSample))
					return
				}
				existingID[a.ID] = struct{}{}
			}

			expectAnnouncementsMap := student.AnnouncementsMap()
			if couldSeeAll {
				// レスポンスのunread_countの検証
				var actualUnreadCount int
				for _, a := range actualAnnouncements {
					if a.Unread {
						actualUnreadCount++
					}
				}
				if !AssertEqual("response unread count", actualUnreadCount, responseUnreadCounts[0]) {
					step.AddError(errNotMatchUnreadCount(hresSample))
					return
				}

				if len(expectAnnouncementsMap) > len(actualAnnouncements) {
					step.AddError(errNotMatchUnder(hresSample))
					return
				}

				if len(expectAnnouncementsMap) < len(actualAnnouncements) {
					step.AddError(errNotMatchOver(hresSample))
					return
				}
			}

			for _, actual := range actualAnnouncements {
				expectStatus, ok := expectAnnouncementsMap[actual.ID]
				if !ok {
					AdminLogger.Printf("less announcements -> name: %v, title:  %v", actual.CourseName, actual.Title)
					step.AddError(errNotMatchOver(hresSample))
					return
				}

				// Dirtyフラグが立っていない場合のみ、Unreadの検証を行う
				// 既読化RequestがTimeoutで中断された際、ベンチには既読が反映しないがwebapp側が既読化される可能性があるため。
				if err := AssertEqualAnnouncementListContent(expectStatus, &actual, hresSample, !expectStatus.Dirty); err != nil {
					step.AddError(errNotMatch(hresSample))
					return
				}

				delete(expectAnnouncementsMap, actual.ID)
			}
		}()
	}
	wg.Wait()
}

func (s *Scenario) validateCourses(ctx context.Context, step *isucandar.BenchmarkStep, requestTime time.Duration) {
	errNotMatchCount := func(hres *http.Response) error {
		return errValidation(fails.ErrorInvalidResponse(errors.New("登録されている Course の個数が一致しませんでした"), hres))
	}
	errNotMatch := func(hres *http.Response) error {
		return errValidation(fails.ErrorInvalidResponse(errors.New("存在しないはずの Course が見つかりました"), hres))
	}

	endTime := time.Now().Add(requestTime)
	isNoRequestTime := func(ctx context.Context) bool {
		return time.Now().After(endTime) || ctx.Err() != nil
	}

	students := s.ActiveStudents()
	expectCourses := s.CourseManager.ExposeCoursesForValidation()
	for _, c := range s.initCourses {
		expectCourses[c.ID] = c
	}

	if len(students) == 0 || len(expectCourses) == 0 {
		return
	}

	// searchAPIを叩くユーザ
	student := students[0]

	couldSeeAll := false
	var actuals []*api.GetCourseDetailResponse
	// 空検索パラメータで全部ページング → 科目をすべて集める
	var hresSample *http.Response
	nextPathParam := "/api/courses"
	for {
		if isNoRequestTime(ctx) {
			couldSeeAll = false
			break
		}

		hres, res, err := SearchCourseAction(ctx, student.Agent, nil, nextPathParam)
		if err != nil {
			step.AddError(errValidation(err))
			return
		}
		actuals = append(actuals, res...)

		hresSample = hres
		_, nextPathParam, err = parseLinkHeader(hres)
		if err != nil {
			step.AddError(fails.ErrorCritical(err))
			return
		}

		if nextPathParam == "" {
			couldSeeAll = true
			break
		}
	}

	if couldSeeAll {
		if !AssertEqual("course count", len(expectCourses), len(actuals)) {
			step.AddError(errNotMatchCount(hresSample))
			return
		}
	}

	for _, actual := range actuals {
		expect, ok := expectCourses[actual.ID]
		if !ok {
			step.AddError(errNotMatch(hresSample))
			return
		}

		if err := AssertEqualCourse(expect, actual, hresSample, true); err != nil {
			AdminLogger.Printf("name: %v", expect.Name)
			step.AddError(errNotMatch(hresSample))
			return
		}
	}
}

// 10 秒 + ベンチの処理
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

			hres, res, err := GetGradeAction(ctx, user.Agent)
			if err != nil {
				step.AddError(errValidation(err))
				return
			}

			err = AssertEqualGrade(&expected, &res, hres)
			if err != nil {
				step.AddError(errValidation(err))
				return
			}
		})
		if err != nil {
			AdminLogger.Println("info: cannot start parallel: %w", err)
		}
	}

	p.Wait()
}

func calculateGradeRes(student *model.Student, students map[string]*model.Student) model.GradeRes {
	courses := student.Courses()
	courseResults := make(map[string]*model.CourseResult, len(courses))
	for _, course := range courses {
		result := course.CalcCourseResultByStudentCode(student.Code)
		if result == nil {
			// course は student が履修している科目なので、result が nil になることはないはず
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
		// ここに来ることはない気がするが webapp 的には偏差値のみ 50 でそれ以外は 0
		// summary の対象となる学生がいないというのは、科目がまだ一つも終わっていないことに等しい
		return model.Summary{
			Credits:   0,
			GPA:       0,
			GpaTScore: 50,
			GpaAvg:    0,
			GpaMax:    0,
			GpaMin:    0,
		}
	}

	if _, ok := students[userCode]; !ok {
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
