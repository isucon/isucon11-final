package scenario

import (
	"context"
	"fmt"

	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/api"
	"github.com/isucon/isucon11-final/benchmarker/fails"
	"github.com/isucon/isucon11-final/benchmarker/model"
)

func InitializeAction(ctx context.Context, agent *agent.Agent) (string, error) {
	language, err := api.Initialize(ctx, agent)
	if err != nil {
		return "", err
	}
	if language == "" {
		return "", failure.NewError(fails.ErrCritical, fmt.Errorf("実装言語が返却されていません"))
	}

	return language, nil
}

func LoginAction(ctx context.Context, agent *agent.Agent, u *model.UserData) []error {
	errs := api.AccessLoginPage(ctx, agent)
	if len(errs) > 0 {
		return errs
	}

	err := api.Login(ctx, agent, u.Number, u.RawPassword)
	if err != nil {
		return []error{err}
	}

	return nil
}

func SearchCoursesAction(ctx context.Context, agent *agent.Agent, course *model.Course) []error {
	syllabusIDs, err := api.SearchSyllabus(ctx, agent, course.Keyword[0])
	if err != nil {
		return []error{err}
	}
	// FIXME: pagingが実装されてないので修正

	var isContain bool
	for _, id := range syllabusIDs {
		if id == course.ID {
			isContain = true
		}
	}

	if !isContain {
		err := failure.NewError(fails.ErrApplication, fmt.Errorf(
			"検索結果に期待する講義が含まれませんでした: 講義(%s), 検索キーワード(%s)",
			course.Name, course.Keyword[0]),
		)
		return []error{err}
	}

	if errs := api.AccessSyllabusPage(ctx, agent, course.ID); len(errs) > 0 {
		return errs
	}
	return nil
}

func RegisterCoursesAction(ctx context.Context, student *model.Student, courses []*model.Course) error {
	var coursesID []string
	for _, c := range courses {
		coursesID = append(coursesID, c.ID)
	}

	registeredCoursesID, err := api.RegisterCourses(ctx, student.Agent, student.UserData.Number, coursesID)
	if err != nil {
		return err
	}
	// nolint:staticcheck
	if len(registeredCoursesID) == 0 {
		// FIXME:登録失敗した講義を除いて再登録したい
	}

	student.AddCourses(coursesID)
	for _, c := range courses {
		c.AddStudent(student)
	}

	return nil
}

func FetchRegisteredCoursesAction(ctx context.Context, student *model.Student) ([]string, error) {
	registeredCoursesID, err := api.FetchRegisteredCourses(ctx, student.Agent, student.UserData.Number)
	if err != nil {
		return nil, err
	}

	return registeredCoursesID, nil
}

func RegisterGradeAction(ctx context.Context, faculty *model.Faculty, student *model.Student, courseID string) error {
	var grade uint32 = 1
	err := api.RegisterGrades(ctx, faculty.Agent, courseID, student.UserData.Number, grade)
	if err != nil {
		return err
	}
	student.SetGradesUnchecked(courseID, grade)
	return nil

}

// 他のアクションに付随しないページアクセス
func AccessMyPageAction(ctx context.Context, agent *agent.Agent) []error {
	return api.AccessMyPage(ctx, agent)
}
func AccessRegPageAction(ctx context.Context, agent *agent.Agent) []error {
	return api.AccessCourseRegPage(ctx, agent)
}
