package scenario

import (
	"context"
	"fmt"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/fails"
)

func (s *Scenario) Validation(ctx context.Context, step *isucandar.BenchmarkStep) error {
	if s.NoLoad {
		return nil
	}
	ContestantLogger.Printf("===> VALIDATION")

	s.validateAnnouncements(ctx, step)
	s.validateCourses(ctx, step)
	s.validateEmptyCourses(ctx, step)
	s.validateGrades(ctx, step)

	return nil
}

func (s *Scenario) validateAnnouncements(ctx context.Context, step *isucandar.BenchmarkStep) {
	return
}

func (s *Scenario) validateCourses(ctx context.Context, step *isucandar.BenchmarkStep) {

	return
}

func (s *Scenario) validateEmptyCourses(ctx context.Context, step *isucandar.BenchmarkStep) {
	errNotMatchCount := failure.NewError(fails.ErrCritical, fmt.Errorf("最終検証にて登録されている Course の個数が一致しませんでした"))
	errNotMatch := failure.NewError(fails.ErrCritical, fmt.Errorf("最終検証にて存在しないはずの Course が見つかりました"))

	students := s.ActiveStudent()
	expectCourses := s.RegistrableCourses()

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

	var actualCourseIDs []string
	// 空検索パラメータで全部ページング → 履修可能なコースをすべて集める
	nextPathParam := "/api/syllabus"
	for nextPathParam != "" {
		hres, res, err := SearchCourseAction(ctx, student.Agent, nil, nextPathParam)
		if err != nil {
			step.AddError(failure.NewError(fails.ErrCritical, err))
			return
		}
		for _, c := range res {
			actualCourseIDs = append(actualCourseIDs, c.ID.String())
		}

		_, nextPathParam = parseLinkHeader(hres)
	}

	if len(expectCourses) != len(actualCourseIDs) {
		step.AddError(errNotMatchCount)
		return
	}

	// IDによる存在チェックのみ
	// 中身の検証はLoadでしているので省略
	for _, id := range actualCourseIDs {
		if expectCourses[id] != nil {
			step.AddError(errNotMatch)
			return
		}
	}
	return
}

func (s *Scenario) validateGrades(ctx context.Context, step *isucandar.BenchmarkStep) {

	return
}
