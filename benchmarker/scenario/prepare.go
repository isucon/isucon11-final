package scenario

import (
	"context"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/api"
	"github.com/isucon/isucon11-final/benchmarker/model"
)

func (s *Scenario) Prepare(ctx context.Context, step *isucandar.BenchmarkStep) error {
	ContestantLogger.Printf("===> PREPARE")

	if err := s.prepareCheck(ctx, step); err != nil {
		return Cancel
	}

	step.Result().Score.Reset()

	if s.NoLoad {
		return nil
	}

	f := model.StaticFacultyData
	errs := InitializeAction(ctx, f.Agent, step)
	for _, err := range errs {
		step.AddError(failure.NewError(ErrCritical, err))
	}
	if len(errs) > 0 {
		return Cancel
	}

	return nil
}

// prepareFailCheck は全Phaseの簡易チェックを行う
func (s *Scenario) prepareCheck(ctx context.Context, step *isucandar.BenchmarkStep) error {
	sd := model.StaticStudentsData[0]
	// StaticStudentsDataのAgentを利用しないように新たにAgentを載せたStudentを作成する
	initializeStudent := model.NewStudent(sd.Name, sd.Number, sd.RawPassword)
	initializeStudent.Agent.BaseURL = s.BaseURL
	initializeStudent.Agent.Name = "student_user_agent"

	f := model.StaticFacultyData
	initializeFaculty := model.NewUser(f.Name, f.Number, f.RawPassword)
	initializeFaculty.Agent.BaseURL = s.BaseURL
	initializeFaculty.Agent.Name = "faculty_user_agent"

	errors := step.Result().Errors
	hasErrors := func() bool {
		errors.Wait()

		return len(errors.All()) > 0
	}

	errs := InitializeAction(ctx, f.Agent, step)
	for _, err := range errs {
		step.AddError(failure.NewError(ErrCritical, err))
	}
	if len(errs) > 0 {
		return Cancel
	}

	err := LoginAction(ctx, initializeFaculty, step)
	if err != nil {
		step.AddError(failure.NewError(ErrCritical, err))
	}

	// 履修登録期間
	if status, err := api.ChangePhaseToRegister(ctx, initializeFaculty.Agent); err != nil && status == 200 {
		step.AddError(err)
		return Cancel
	}
	// AccessCheckは並列でリクエストを行うのでerrorはstep.Results().Errorsを確認
	s.prepareAccessCheckInRegister(ctx, initializeStudent, initializeFaculty, step)
	if hasErrors() {
		return Cancel
	}
	if err := s.prepareFastCheckInRegister(ctx, initializeStudent, initializeFaculty); err != nil {
		step.AddError(err)
		return Cancel
	}

	// 講義期間
	if status, err := api.ChangePhaseToClasses(ctx, initializeFaculty.Agent); err != nil && status == 200 {
		step.AddError(err)
		return Cancel
	}
	s.prepareAccessCheckInClass(ctx, initializeStudent, initializeFaculty, step)
	if hasErrors() {
		return Cancel
	}
	if err := s.prepareFastCheckInClass(ctx, initializeStudent, initializeFaculty); err != nil {
		step.AddError(err)
		return Cancel
	}

	// 成績開示期間
	if status, err := api.ChangePhaseToResult(ctx, initializeFaculty.Agent); err != nil && status == 200 {
		step.AddError(err)
		return Cancel
	}

	s.prepareAccessCheckInResult(ctx, initializeStudent, initializeFaculty, step)
	if hasErrors() {
		return Cancel
	}

	if err := s.prepareFastCheckInResult(ctx, initializeStudent, initializeFaculty); err != nil {
		step.AddError(err)
		return Cancel
	}

	return nil
}

// TODO: 以下のTODOをaction.goあたりにまとめる
func (s *Scenario) prepareAccessCheckInRegister(ctx context.Context, student *model.Student, faculty *model.User, step *isucandar.BenchmarkStep) {
	// 履修登録期間でのアクセス制御チェック
	// TODO: goroutineで各エンドポイントへアクセス確認. エラーはstep.AddError()で追加
	return
}
func (s *Scenario) prepareAccessCheckInClass(ctx context.Context, student *model.Student, faculty *model.User, step *isucandar.BenchmarkStep) {
	// 講義期間でのアクセス制御チェック
	// TODO: goroutineで各エンドポイントへアクセス確認. エラーはstep.AddError()で追加
	return
}
func (s *Scenario) prepareAccessCheckInResult(ctx context.Context, student *model.Student, faculty *model.User, step *isucandar.BenchmarkStep) {
	// 履修登録期間でのアクセス制御チェック
	// TODO: goroutineで各エンドポイントへアクセス確認. エラーはstep.AddError()で追加
	return
}

func (s *Scenario) prepareFastCheckInRegister(ctx context.Context, student *model.Student, faculty *model.User) error {
	// 履修登録期間での動作確認
	// TODO: initializeStudentによる講義Aへの履修登録.
	// TODO: (MEMO)直列なのでエラーはそのまま返す
	return nil
}
func (s *Scenario) prepareFastCheckInClass(ctx context.Context, student *model.Student, faculty *model.User) error {
	// 講義期間での動作確認
	// TODO: Userによる資料追加
	// TODO: Userによるお知らせ追加
	// TODO: Studentによるお知らせ確認
	// TODO: Studentによる資料DL
	// TODO: Userによる出席コード追加
	// TODO: Studentによる出席コード入力
	// TODO: Userによる課題追加
	// TODO: Studentによる課題提出
	return nil
}
func (s *Scenario) prepareFastCheckInResult(ctx context.Context, student *model.Student, faculty *model.User) error {
	// 成績開示期間での動作確認
	// TODO: Userによる講義Aの課題確認 & 成績登録
	// TODO: Studentによる成績確認
	return nil
}
