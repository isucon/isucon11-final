package scenario

import (
	"context"
	"net/http"
	"time"

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

func SearchCourseAction(ctx context.Context, agent *agent.Agent) (*http.Response, []*model.Course, error) {
	<-time.After(RequestDuration * time.Millisecond) // FIXME: for debug
	return nil, nil, nil
}

func TakeCourseAction(ctx context.Context, agent *agent.Agent, course *model.Course) (*http.Response, error) {
	<-time.After(RequestDuration * time.Millisecond) // FIXME: for debug
	return nil, nil
}

func GetAnnouncementAction(ctx context.Context, agent *agent.Agent) (*http.Response, []*model.Announcement, error) {
	<-time.After(RequestDuration * time.Millisecond) // FIXME: for debug
	return nil, nil, nil
}

func GetAnnouncementDetailAction(ctx context.Context, agent *agent.Agent, id string) (*http.Response, *model.Announcement, error) {
	<-time.After(RequestDuration * time.Millisecond) // FIXME: for debug
	return nil, nil, nil
}
