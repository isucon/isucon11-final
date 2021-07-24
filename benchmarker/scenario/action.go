package scenario

import (
	"context"
	"net/http"
	"time"

	"github.com/isucon/isucon11-final/benchmarker/api"

	"github.com/isucon/isucon11-final/benchmarker/model"

	"github.com/isucon/isucandar/agent"
)

// action.go
// apiパッケージでリクエストを行う関数群
// param: modelオブジェクト
// POSTとか return: http.Response, error
// GETとか return: modelオブジェクト, http.Response, error
// TODO: 返すのはmodelじゃないほうがいい気がした

const (
	RequestDuration = 100
)

// FIXME: 何返すか決めてない
func GetGradeAction(ctx context.Context, agent *agent.Agent) (*http.Response, error) {
	<-time.After(RequestDuration * time.Millisecond) // FIXME: for debug
	return nil, nil
}

func SearchCourseAction(ctx context.Context, agent *agent.Agent) (*http.Response, []*api.GetCourseDetailResponse, error) {
	<-time.After(RequestDuration * time.Millisecond) // FIXME: for debug
	return nil, nil, nil
}

func TakeCourseAction(ctx context.Context, agent *agent.Agent, course *model.Course) (*http.Response, error) {
	<-time.After(RequestDuration * time.Millisecond) // FIXME: for debug
	return nil, nil
}

func GetAnnouncementListAction(ctx context.Context, agent *agent.Agent, next string) (*http.Response, api.AnnouncementList, error) {
	// nextが存在する場合そちらにアクセスする。空文字なら1ページ目にアクセス。
	<-time.After(RequestDuration * time.Millisecond) // FIXME: for debug
	return nil, nil, nil
}

func GetAnnouncementDetailAction(ctx context.Context, agent *agent.Agent, id string) (*http.Response, *api.Announcement, error) {
	<-time.After(RequestDuration * time.Millisecond) // FIXME: for debug
	return nil, nil, nil
}

func AddCourseAction(ctx context.Context, faculty *model.Faculty, course *model.Course) (*http.Response, error) {
	<-time.After(RequestDuration * time.Millisecond) // FIXME: for debug
	return nil, nil
}

func AddClassAction(ctx context.Context, agent *agent.Agent, course *model.Course) (*http.Response, *model.Class, *model.Announcement, error) {
	<-time.After(RequestDuration * time.Millisecond) // FIXME: for debug
	return nil, nil, nil, nil
}

func SubmitAssignmentAction(ctx context.Context, agent *agent.Agent, classID string, submission *model.Submission) (*http.Response, error) {
	<-time.After(RequestDuration * time.Millisecond) // FIXME: for debug
	return nil, nil
}

func DownloadSubmissionsAction(ctx context.Context, agent *agent.Agent, classID string) (*http.Response, []byte, error) {
	<-time.After(RequestDuration * time.Millisecond) // FIXME: for debug
	return nil, nil, nil
}

func PostGradeAction(ctx context.Context, agent *agent.Agent, classID string, score int, code string) (*http.Response, error) {
	<-time.After(RequestDuration * time.Millisecond) // FIXME: for debug
	return nil, nil
}
