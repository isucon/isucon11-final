package scenario

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/isucon/isucon11-final/benchmarker/score"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/parallel"

	"github.com/isucon/isucon11-final/benchmarker/api"
	"github.com/isucon/isucon11-final/benchmarker/fails"
	"github.com/isucon/isucon11-final/benchmarker/model"
	"github.com/isucon/isucon11-final/benchmarker/util"
)

const (
	validationRequestTime = 10 * time.Second
)

func (s *Scenario) Validation(ctx context.Context, step *isucandar.BenchmarkStep) error {
	if s.NoLoad {
		return nil
	}
	ContestantLogger.Printf("===> VALIDATION")

	AdminLogger.Printf("no validation timeout flag: %v", s.NoValidationTimeout)

	// これは 10 秒 + ベンチの計算分しかかからないので先にやる
	s.validateGrades(ctx, step)
	AdminLogger.Println("finished grade validation")

	// それぞれ 10 秒以上かかったらリクエストをやめて通す
	s.validateAnnouncements(ctx, step)
	AdminLogger.Println("finished announcement validation")
	s.validateCourses(ctx, step)
	AdminLogger.Println("finished courses validation")

	return nil
}

func errValidation(err error) error {
	return fails.ErrorCritical(fmt.Errorf("整合性チェックに失敗しました: %w", err))
}

func splitArr(students []*model.Student, n int) []*model.Student {
	if len(students) < n {
		return students
	}

	res := make([]*model.Student, n)
	for i := 0; i < n; i++ {
		idx := int(float64(len(students)) * float64(i) / float64(n))
		if idx < 0 {
			idx = 0
		}
		if idx > len(students)-1 {
			idx = len(students) - 1
		}
		res[i] = students[idx]
	}

	return res
}

func (s *Scenario) validateAnnouncements(ctx context.Context, step *isucandar.BenchmarkStep) {
	errNotMatchUnreadCountAmongPages := func(hres *http.Response) error {
		return errValidation(fails.ErrorInvalidResponse(errors.New("お知らせ一覧の各ページの unread_count の値が一致しません"), hres))
	}
	errNotMatchUnreadCount := func(hres *http.Response) error {
		return errValidation(fails.ErrorInvalidResponse(errors.New("お知らせ一覧の unread_count の値が実際に返却された未読お知らせの総数と一致しません"), hres))
	}
	errNotSorted := func(hres *http.Response) error {
		return errValidation(fails.ErrorInvalidResponse(errors.New("お知らせ一覧の順序が id の降順になっていません"), hres))
	}
	errNotMatchOver := func(hres *http.Response) error {
		return errValidation(fails.ErrorInvalidResponse(errors.New("お知らせ一覧に存在しないはずのお知らせが見つかりました"), hres))
	}
	errNotMatchUnder := func(hres *http.Response) error {
		return errValidation(fails.ErrorInvalidResponse(errors.New("お知らせ一覧に存在するはずのお知らせが見つかりませんでした"), hres))
	}
	errDuplicated := func(hres *http.Response) error {
		return errValidation(fails.ErrorInvalidResponse(errors.New("お知らせ一覧に id が重複したお知らせが見つかりました"), hres))
	}
	errUnnecessaryNext := func(hres *http.Response) error {
		return errValidation(fails.ErrorInvalidResponse(errors.New("お知らせ一覧の最後のページの link header に next が設定されていました"), hres))
	}
	errMissingNext := func(hres *http.Response) error {
		return errValidation(fails.ErrorInvalidResponse(errors.New("お知らせ一覧の最後以外のページの link header に next が設定されていませんでした"), hres))
	}

	sampleStudents := splitArr(s.ActiveStudents(), validateAnnouncementSampleStudentCount)
	wg := sync.WaitGroup{}
	wg.Add(len(sampleStudents))
	for _ = range sampleStudents {
		AdminLogger.Printf("add wait for check announcement")
	}

	for _, student := range sampleStudents {
		student := student
		go func() {
			defer func() {
				AdminLogger.Printf("done announcement wg")
				wg.Done()
			}()

			// 1〜5秒ランダムに待つ
			time.Sleep(time.Duration(rand.Int63n(5)+1) * time.Second)

			// responseに含まれるunread_count
			responseUnreadCounts := make([]int, 0)
			actualAnnouncements := make([]api.AnnouncementResponse, 0)

			timer := time.After(validationRequestTime)
			var hresFirst *http.Response
			var next string
			couldSeeAll := false
			maxPage := (student.AnnouncementCount()-1)/AnnouncementCountPerPage + 1
		fetchLoop:
			for i := 1; i <= maxPage; i++ {
				hres, res, err := GetAnnouncementListAction(ctx, student.Agent, next, "")
				if err != nil {
					step.AddError(errValidation(err))
					return
				}
				if i == 1 {
					hresFirst = hres
				}

				responseUnreadCounts = append(responseUnreadCounts, res.UnreadCount)
				actualAnnouncements = append(actualAnnouncements, res.Announcements...)

				_, next, err = parseLinkHeader(hres)
				if err != nil {
					step.AddError(fails.ErrorCritical(err))
				}

				if i == maxPage && next != "" {
					step.AddError(errUnnecessaryNext(hres))
					return
				}
				if i != maxPage && next == "" {
					step.AddError(errMissingNext(hres))
					return
				}
				if i == maxPage {
					couldSeeAll = true
					break
				}

				if !s.NoValidationTimeout {
					select {
					case <-timer:
						step.AddScore(score.ValidateTimeout)
						break fetchLoop
					default:
						AdminLogger.Printf("check next announcement page")
					}
				}
			}

			AdminLogger.Printf("finish correct announcements")

			// UnreadCount は各ページのレスポンスですべて同じ値が返ってくることを検証
			for _, unreadCount := range responseUnreadCounts {
				if responseUnreadCounts[0] != unreadCount {
					step.AddError(errNotMatchUnreadCountAmongPages(hresFirst))
					return
				}
			}

			// 順序の検証
			for i := 0; i < len(actualAnnouncements)-1; i++ {
				if actualAnnouncements[i].ID < actualAnnouncements[i+1].ID {
					step.AddError(errNotSorted(hresFirst))
					return
				}
			}

			// actual の重複確認
			existingID := make(map[string]struct{}, len(actualAnnouncements))
			for _, a := range actualAnnouncements {
				if _, ok := existingID[a.ID]; ok {
					step.AddError(errDuplicated(hresFirst))
					return
				}
				existingID[a.ID] = struct{}{}
			}

			expectAnnouncementsMap := student.AnnouncementsMap()

			for _, actual := range actualAnnouncements {
				expectStatus, ok := expectAnnouncementsMap[actual.ID]
				if !ok {
					AdminLogger.Printf("extra announcements -> name: %v, title:  %v", actual.CourseName, actual.Title)
					step.AddError(errNotMatchOver(hresFirst))
					return
				}

				// Dirtyフラグが立っていない場合のみ、Unreadの検証を行う
				// 既読化RequestがTimeoutで中断された際、ベンチには既読が反映しないがwebapp側が既読化される可能性があるため。
				if err := AssertEqualAnnouncementListContent(expectStatus, &actual, !expectStatus.Dirty); err != nil {
					step.AddError(errValidation(fails.ErrorInvalidResponse(err, hresFirst)))
					return
				}
			}

			if couldSeeAll {
				// レスポンスのunread_countの検証
				var actualUnreadCount int
				for _, a := range actualAnnouncements {
					if a.Unread {
						actualUnreadCount++
					}
				}
				if !AssertEqual("response unread count", actualUnreadCount, responseUnreadCounts[0]) {
					step.AddError(errNotMatchUnreadCount(hresFirst))
					return
				}

				if !AssertEqual("announcement len", len(expectAnnouncementsMap), len(actualAnnouncements)) {
					// 上で actual が expect の部分集合であることを確認しているので、ここで数が合わない場合は expect の方が多い
					AdminLogger.Printf("announcement len mismatch -> code: %v", student.Code)
					step.AddError(errNotMatchUnder(hresFirst))
					return
				}
			}
		}()
	}
	wg.Wait()
}

func (s *Scenario) validateCourses(ctx context.Context, step *isucandar.BenchmarkStep) {
	errNotMatchUnder := func(hres *http.Response) error {
		return errValidation(fails.ErrorInvalidResponse(errors.New("科目検索で登録されているはずの科目が見つかりませんでした"), hres))
	}
	errNotMatchOver := func(hres *http.Response) error {
		return errValidation(fails.ErrorInvalidResponse(errors.New("科目検索で登録されてないはずの科目が見つかりました"), hres))
	}
	errUnnecessaryNext := func(hres *http.Response) error {
		return errValidation(fails.ErrorInvalidResponse(errors.New("科目検索の最後のページの link header に next が設定されていました"), hres))
	}
	errMissingNext := func(hres *http.Response) error {
		return errValidation(fails.ErrorInvalidResponse(errors.New("科目検索の最後以外のページの link header に next が設定されていませんでした"), hres))
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
	timer := time.After(validationRequestTime)
	var actuals []*api.GetCourseDetailResponse
	// 空検索パラメータで全部ページング → 科目をすべて集める
	var hresFirst *http.Response
	nextPathParam := "/api/courses"
	maxPage := (s.CourseManager.GetCourseCount()-1)/SearchCourseCountPerPage + 1
fetchLoop:
	for i := 1; i <= maxPage; i++ {
		hres, res, err := SearchCourseAction(ctx, student.Agent, nil, nextPathParam)
		if err != nil {
			step.AddError(errValidation(err))
			return
		}
		if i == 1 {
			hresFirst = hres
		}

		actuals = append(actuals, res...)

		_, nextPathParam, err = parseLinkHeader(hres)
		if err != nil {
			step.AddError(fails.ErrorCritical(err))
			return
		}

		if i == maxPage && nextPathParam != "" {
			step.AddError(errUnnecessaryNext(hres))
			return
		}

		if i != maxPage && nextPathParam == "" {
			step.AddError(errMissingNext(hres))
			return
		}

		if i == maxPage {
			couldSeeAll = true
			break
		}

		if !s.NoValidationTimeout {
			select {
			case <-timer:
				step.AddScore(score.ValidateTimeout)
				break fetchLoop
			default:
			}
		}
	}

	for _, actual := range actuals {
		expect, ok := expectCourses[actual.ID]
		if !ok {
			step.AddError(errNotMatchOver(hresFirst))
			return
		}

		if err := AssertEqualCourse(expect, actual, true); err != nil {
			AdminLogger.Printf("name: %v", expect.Name)
			step.AddError(errValidation(fails.ErrorInvalidResponse(err, hresFirst)))
			return
		}
	}

	if couldSeeAll {
		// 上で actual が expect の部分集合であることを確認しているので、ここで数が合わない場合は expect の方が多い
		if !AssertEqual("course count", len(expectCourses), len(actuals)) {
			step.AddError(errNotMatchUnder(hresFirst))
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
	if len(users) == 0 {
		AdminLogger.Printf("HasFinishedCourse Student is 0")
		return
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

			err = AssertEqualGrade(&expected, &res)
			if err != nil {
				step.AddError(errValidation(fails.ErrorInvalidResponse(err, hres)))
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
