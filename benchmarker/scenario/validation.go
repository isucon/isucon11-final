package scenario

import (
	"context"
	"fmt"
	"math"
	"time"

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
	errTimeout := failure.NewError(fails.ErrCritical, fmt.Errorf("時間内に Announcemet の検証が完了しませんでした"))
	errNotMatchUnreadCount := failure.NewError(fails.ErrCritical, fmt.Errorf("/api/announcements の unread_count の値が不正です"))
	errNotSorted := failure.NewError(fails.ErrCritical, fmt.Errorf("/api/announcements の順序が不正です"))
	errNotMatchOver := failure.NewError(fails.ErrCritical, fmt.Errorf("最終検証にて存在しないはずの Announcement が見つかりました"))
	errNotMatchUnder := failure.NewError(fails.ErrCritical, fmt.Errorf("最終検証にて存在するはずの Announcement が見つかりませんでした"))

	// TODO: 並列化
	sampleCount := s.ActiveStudentCount() * validateAnnouncementsRate
	sampleIndices := generate.ShuffledInts(s.ActiveStudentCount())[:sampleCount]
	for sampleIndex := range sampleIndices {
		student := s.activeStudents[sampleIndex]
		var responseUnreadCount int // responseに含まれるunread_count
		actualAnnouncements := map[string]api.AnnouncementResponse{}
		lastCreatedAt := int64(math.MaxInt64)

		timer := time.After(10 * time.Second)
		var next string
		for {
			hres, res, err := GetAnnouncementListAction(ctx, student.Agent, next)
			if err != nil {
				step.AddError(errValidation(err))
				return
			}

			// UnreadCount は各ページのレスポンスですべて同じ値が返ってくるはず
			if next == "" {
				responseUnreadCount = res.UnreadCount
			} else if responseUnreadCount != res.UnreadCount {
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
		if !AssertEqual("response unread count", actualUnreadCount, responseUnreadCount) {
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
			}

			// ベンチ内データが既読の場合のみUnreadの検証を行う
			// 既読化RequestがTimeoutで中断された際、ベンチには既読が反映しないがwebapp側が既読化される可能性があるため。
			if !expectStatus.Unread {
				if !AssertEqual("announcement Unread", expectStatus.Unread, actual.Unread) {
					AdminLogger.Printf("extra announcements ->name: %v, title:  %v", actual.CourseName, actual.Title)
					step.AddError(errNotMatchOver)
					return
				}
			}

			if !AssertEqual("announcement ID", expect.ID, actual.ID) ||
				!AssertEqual("announcement Code", expect.CourseID, actual.CourseID) ||
				!AssertEqual("announcement Title", expect.Title, actual.Title) ||
				!AssertEqual("announcement CourseName", expect.CourseName, actual.CourseName) ||
				!AssertEqual("announcement CreatedAt", expect.CreatedAt, actual.CreatedAt) {
				AdminLogger.Printf("extra announcements ->name: %v, title:  %v", actual.CourseName, actual.Title)
				step.AddError(errNotMatchOver)
				return
			}
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
