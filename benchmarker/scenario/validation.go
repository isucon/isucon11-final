package scenario

import (
	"context"
	"math"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/failure"
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

	return
}

func (s *Scenario) validateGrades(ctx context.Context, step *isucandar.BenchmarkStep) {

	return
}
