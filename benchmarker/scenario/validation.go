package scenario

import (
	"context"
	"fmt"
	"math"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/api"
	"github.com/isucon/isucon11-final/benchmarker/fails"
	"github.com/isucon/isucon11-final/benchmarker/generate"
)

const (
	validateAnnouncementsRate = 1.0
)

// validationフェーズでのエラーはすべてCriticalにする
func errValidation(err error) error {
	return failure.NewError(fails.ErrCritical, err)
}

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
	sampleCount := s.ActiveStudentCount() * validateAnnouncementsRate
	sampleIndices := generate.ShuffledInts(s.ActiveStudentCount())[:sampleCount]
	for sampleIndex := range sampleIndices {
		student := s.activeStudents[sampleIndex]
		var actualCount int
		var autualUnreadCount int
		var expectedUnreadCount int
		lastCreatedAt := int64(math.MaxInt64)
		var next string
		for {
			hres, res, err := GetAnnouncementListAction(ctx, student.Agent, next)
			if err != nil {
				step.AddError(errValidation(err))
				return
			}

			// UnreadCount は各ページのレスポンスですべて同じ値が返ってくるはず
			if next == "" {
				expectedUnreadCount = res.UnreadCount
			} else if expectedUnreadCount != res.UnreadCount {
				step.AddError(errValidation(errInvalidResponse("お知らせの未読数が不正です")))
				return
			}

			for _, actual := range res.Announcements {
				expected := student.GetAnnouncement(actual.ID)
				if expected == nil {
					step.AddError(errValidation(errInvalidResponse("存在しないはずのお知らせがレスポンスに含まれています")))
					return
				}

				// 内容の検証
				// Message の検証はお知らせ詳細を取得しないとできないので省略
				if err := verifyAnnouncementsContent(&actual, expected); err != nil {
					step.AddError(errValidation(err))
					return
				}

				actualCount++
				if actual.Unread {
					autualUnreadCount++
				}

				// 順序の検証
				if lastCreatedAt < actual.CreatedAt {
					step.AddError(errValidation(errInvalidResponse("お知らせの順序が不正です")))
					return
				}
				lastCreatedAt = actual.CreatedAt
			}

			_, next = parseLinkHeader(hres)
			if next == "" {
				break
			}
		}

		// お知らせ総数の検証
		// レスポンスに含まれるお知らせがすべてベンチ側に含まれていることは検証済みのため、総数が一致すれば両者は集合として一致する
		if actualCount != student.AnnouncementCount() {
			step.AddError(errValidation(errInvalidResponse("お知らせの総数が期待する値と一致しません")))
			return
		}

		// 未読数の検証
		if autualUnreadCount != expectedUnreadCount {
			step.AddError(errValidation(errInvalidResponse("お知らせの未読数が期待する値と一致しません")))
			return
		}
	}
}

func (s *Scenario) validateCourses(ctx context.Context, step *isucandar.BenchmarkStep) {
	errNotMatchCount := failure.NewError(fails.ErrCritical, fmt.Errorf("最終検証にて登録されている Course の個数が一致しませんでした"))
	errNotMatch := failure.NewError(fails.ErrCritical, fmt.Errorf("最終検証にて存在しないはずの Course が見つかりました"))

	students := s.ActiveStudents()
	expectCourses := s.Courses()

	if len(students) == 0 || len(expectCourses) == 0 {
		return
	}

	// searchAPIを叩くユーザ
	student := students[0]

	_, err := LoginAction(ctx, student.Agent, student.UserAccount)
	if err != nil {
		step.AddError(failure.NewError(fails.ErrCritical, err))
		return
	}

	var actuals []*api.GetCourseDetailResponse
	// 空検索パラメータで全部ページング → コースをすべて集める
	nextPathParam := "/api/syllabus"
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
			// webappはMonday..., benchは0-6
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

	return
}
