package scenario

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/isucon/isucon11-final/benchmarker/fails"

	"github.com/isucon/isucandar/failure"

	"github.com/isucon/isucon11-final/benchmarker/api"

	"github.com/isucon/isucon11-final/benchmarker/model"

	"github.com/isucon/isucandar/agent"
)

// action.go
// apiパッケージでリクエストを行う関数群
// param: modelオブジェクト
// POSTとか return: http.Response, error
// GETとか return: apiパッケージのオブジェクト, http.Response, error

const (
	RequestDuration = 100
)

func InitializeAction(ctx context.Context, agent *agent.Agent) (*http.Response, api.InitializeResponse, error) {
	res := api.InitializeResponse{}
	hres, err := api.Initialize(ctx, agent)
	if err != nil {
		return hres, res, err
	}
	defer hres.Body.Close()

	err = verifyStatusCode(hres, []int{http.StatusOK})
	if err != nil {
		return hres, res, err
	}

	err = json.NewDecoder(hres.Body).Decode(&res)
	if err != nil {
		return hres, res, failure.NewError(fails.ErrHTTP, err)
	}

	return hres, res, nil
}

func LoginAction(ctx context.Context, agent *agent.Agent, useraccount *model.UserAccount) (*http.Response, error) {
	req := api.LoginRequest{
		Code:     useraccount.Code,
		Password: useraccount.RawPassword,
	}
	hres, err := api.Login(ctx, agent, req)
	if err != nil {
		return hres, failure.NewError(fails.ErrHTTP, err)
	}
	defer hres.Body.Close()

	err = verifyStatusCode(hres, []int{http.StatusOK})
	if err != nil {
		return hres, err
	}

	return hres, nil
}

func GetMeAction(ctx context.Context, agent *agent.Agent) (*http.Response, api.GetMeResponse, error) {
	res := api.GetMeResponse{}
	hres, err := api.GetMe(ctx, agent)
	if err != nil {
		return hres, res, failure.NewError(fails.ErrHTTP, err)
	}
	defer hres.Body.Close()

	err = verifyStatusCode(hres, []int{http.StatusOK, http.StatusNotModified})
	if err != nil {
		return hres, res, err
	}

	err = json.NewDecoder(hres.Body).Decode(&res)
	if err != nil {
		return hres, res, failure.NewError(fails.ErrHTTP, err)
	}

	return hres, res, nil
}

func GetGradeAction(ctx context.Context, agent *agent.Agent) (*http.Response, api.GetGradeResponse, error) {
	res := api.GetGradeResponse{}
	hres, err := api.GetGrades(ctx, agent)
	if err != nil {
		return hres, res, err
	}
	defer hres.Body.Close()

	err = verifyStatusCode(hres, []int{http.StatusOK, http.StatusNotModified})
	if err != nil {
		return hres, res, err
	}

	err = json.NewDecoder(hres.Body).Decode(&res)
	if err != nil {
		return hres, res, failure.NewError(fails.ErrHTTP, err)
	}

	return hres, res, nil
}

func GetRegisteredCoursesAction(ctx context.Context, agent *agent.Agent) (*http.Response, []*api.GetRegisteredCourseResponseContent, error) {
	hres, err := api.GetRegisteredCourses(ctx, agent)
	if err != nil {
		return hres, nil, failure.NewError(fails.ErrHTTP, err)
	}
	defer hres.Body.Close()

	err = verifyStatusCode(hres, []int{http.StatusOK, http.StatusNotModified})
	if err != nil {
		return hres, nil, err
	}

	res := make([]*api.GetRegisteredCourseResponseContent, 0)
	err = json.NewDecoder(hres.Body).Decode(&res)
	if err != nil {
		return hres, res, failure.NewError(fails.ErrHTTP, err)
	}

	return hres, res, nil
}

func SearchCourseAction(ctx context.Context, agent *agent.Agent, param *model.SearchCourseParam, nextPathParam string) (*http.Response, []*api.GetCourseDetailResponse, error) {
	var hres *http.Response
	if nextPathParam != "" {
		var err error
		hres, err = api.SearchCourseWithNext(ctx, agent, nextPathParam)
		if err != nil {
			return hres, nil, failure.NewError(fails.ErrHTTP, err)
		}
		defer hres.Body.Close()
	} else {
		req := api.SearchCourseRequest{
			Type:     api.CourseType(param.Type),
			Credit:   uint8(param.Credit),
			Teacher:  param.Teacher,
			Period:   uint8(param.Period + 1),
			Keywords: strings.Join(param.Keywords, " "),
		}
		if param.DayOfWeek != -1 {
			req.DayOfWeek = api.DayOfWeekTable[param.DayOfWeek]
		}
		var err error
		hres, err = api.SearchCourse(ctx, agent, &req)
		if err != nil {
			return hres, nil, failure.NewError(fails.ErrHTTP, err)
		}
		defer hres.Body.Close()
	}

	err := verifyStatusCode(hres, []int{http.StatusOK, http.StatusNotModified})
	if err != nil {
		return hres, nil, err
	}

	res := make([]*api.GetCourseDetailResponse, 0)
	err = json.NewDecoder(hres.Body).Decode(&res)
	if err != nil {
		return hres, res, failure.NewError(fails.ErrHTTP, err)
	}

	return hres, res, nil
}

func GetCourseDetailAction(ctx context.Context, agent *agent.Agent, id string) (*http.Response, api.GetCourseDetailResponse, error) {
	res := api.GetCourseDetailResponse{}
	hres, err := api.GetCourseDetail(ctx, agent, id)
	if err != nil {
		return hres, res, failure.NewError(fails.ErrHTTP, err)
	}
	defer hres.Body.Close()

	err = verifyStatusCode(hres, []int{http.StatusOK, http.StatusNotModified})
	if err != nil {
		return hres, res, err
	}

	err = json.NewDecoder(hres.Body).Decode(&res)
	if err != nil {
		return hres, res, failure.NewError(fails.ErrHTTP, err)
	}

	return hres, res, nil
}

func TakeCoursesAction(ctx context.Context, agent *agent.Agent, courses []*model.Course) (*http.Response, api.RegisterCoursesErrorResponse, error) {
	req := make([]api.RegisterCourseRequestContent, 0, len(courses))
	for _, c := range courses {
		req = append(req, api.RegisterCourseRequestContent{ID: c.ID})
	}

	eres := api.RegisterCoursesErrorResponse{}
	hres, err := api.RegisterCourses(ctx, agent, req)
	if err != nil {
		return hres, eres, failure.NewError(fails.ErrHTTP, err)
	}
	defer hres.Body.Close()

	err = verifyStatusCode(hres, []int{http.StatusOK})
	if err != nil {
		// 400のときはエラー内容が返ってくるのでレスポンスをデコードする
		if hres.StatusCode == http.StatusBadRequest {
			decodeErr := json.NewDecoder(hres.Body).Decode(&eres)
			if decodeErr != nil {
				return hres, eres, failure.NewError(fails.ErrHTTP, decodeErr)
			}

			return hres, eres, err
		}

		return hres, eres, err
	}

	return hres, eres, nil
}

func GetAnnouncementListAction(ctx context.Context, agent *agent.Agent, next string) (*http.Response, api.GetAnnouncementsResponse, error) {
	res := api.GetAnnouncementsResponse{}
	if next == "" {
		next = "/api/announcements"
	}
	hres, err := api.GetAnnouncementList(ctx, agent, next, nil)
	if err != nil {
		return hres, res, failure.NewError(fails.ErrHTTP, err)
	}
	defer hres.Body.Close()

	err = verifyStatusCode(hres, []int{http.StatusOK, http.StatusNotModified})
	if err != nil {
		return hres, res, err
	}

	err = json.NewDecoder(hres.Body).Decode(&res)
	if err != nil {
		return hres, res, failure.NewError(fails.ErrHTTP, err)
	}

	return hres, res, nil
}

func GetAnnouncementDetailAction(ctx context.Context, agent *agent.Agent, id string) (*http.Response, api.GetAnnouncementDetailResponse, error) {
	res := api.GetAnnouncementDetailResponse{}
	hres, err := api.GetAnnouncementDetail(ctx, agent, id)
	if err != nil {
		return hres, res, failure.NewError(fails.ErrHTTP, err)
	}
	defer hres.Body.Close()

	err = verifyStatusCode(hres, []int{http.StatusOK, http.StatusNotModified})
	if err != nil {
		return hres, res, err
	}

	err = json.NewDecoder(hres.Body).Decode(&res)
	if err != nil {
		return hres, res, failure.NewError(fails.ErrHTTP, err)
	}

	return hres, res, nil
}

func SendAnnouncementAction(ctx context.Context, agent *agent.Agent, announcement *model.Announcement) (*http.Response, error) {
	req := &api.AddAnnouncementRequest{
		ID:        announcement.ID,
		CourseID:  announcement.CourseID,
		Title:     announcement.Title,
		Message:   announcement.Message,
		CreatedAt: announcement.CreatedAt,
	}

	hres, err := api.AddAnnouncement(ctx, agent, *req)
	if err != nil {
		return hres, failure.NewError(fails.ErrHTTP, err)
	}
	defer hres.Body.Close()

	err = verifyStatusCode(hres, []int{http.StatusCreated})
	if err != nil {
		return hres, err
	}

	return hres, nil
}

func GetClassesAction(ctx context.Context, agent *agent.Agent, courseID string) (*http.Response, []*api.GetClassResponse, error) {
	res := make([]*api.GetClassResponse, 0)
	hres, err := api.GetClasses(ctx, agent, courseID)
	if err != nil {
		return hres, res, failure.NewError(fails.ErrHTTP, err)
	}
	defer hres.Body.Close()

	err = verifyStatusCode(hres, []int{http.StatusOK, http.StatusNotModified})
	if err != nil {
		return hres, res, err
	}

	err = json.NewDecoder(hres.Body).Decode(&res)
	if err != nil {
		return hres, res, failure.NewError(fails.ErrHTTP, err)
	}

	return hres, res, nil
}

func AddClassAction(ctx context.Context, agent *agent.Agent, course *model.Course, param *model.ClassParam) (*http.Response, api.AddClassResponse, error) {
	req := api.AddClassRequest{
		Part:        uint8(param.Part),
		Title:       param.Title,
		Description: param.Desc,
	}

	res := api.AddClassResponse{}
	hres, err := api.AddClass(ctx, agent, course.ID, req)
	if err != nil {
		return hres, res, failure.NewError(fails.ErrHTTP, err)
	}
	defer hres.Body.Close()

	err = verifyStatusCode(hres, []int{http.StatusCreated})
	if err != nil {
		return hres, res, err
	}

	err = json.NewDecoder(hres.Body).Decode(&res)
	if err != nil {
		return hres, res, failure.NewError(fails.ErrHTTP, err)
	}

	return hres, res, nil
}

func AddCourseAction(ctx context.Context, agent *agent.Agent, param *model.CourseParam) (*http.Response, api.AddCourseResponse, error) {
	// 不正な param.DayOfWeek は空文字列として送信してprepareの異常系チェックに使用する
	var dayOfWeek api.DayOfWeek
	if 0 <= param.DayOfWeek && param.DayOfWeek < len(api.DayOfWeekTable) {
		dayOfWeek = api.DayOfWeekTable[param.DayOfWeek]
	}

	req := api.AddCourseRequest{
		Code:        param.Code,
		Type:        api.CourseType(param.Type),
		Name:        param.Name,
		Description: param.Description,
		Credit:      param.Credit,
		Period:      param.Period + 1,
		DayOfWeek:   dayOfWeek,
		Keywords:    param.Keywords,
	}
	res := api.AddCourseResponse{}
	hres, err := api.AddCourse(ctx, agent, req)
	if err != nil {
		return hres, res, failure.NewError(fails.ErrHTTP, err)
	}
	defer hres.Body.Close()

	err = verifyStatusCode(hres, []int{http.StatusCreated})
	if err != nil {
		return hres, res, err
	}

	err = json.NewDecoder(hres.Body).Decode(&res)
	if err != nil {
		return hres, res, failure.NewError(fails.ErrHTTP, err)
	}

	return hres, res, nil
}

func SubmitAssignmentAction(ctx context.Context, agent *agent.Agent, courseID, classID string, title string, data []byte) (*http.Response, error) {
	hres, err := api.SubmitAssignment(ctx, agent, courseID, classID, title, data)
	if err != nil {
		return hres, failure.NewError(fails.ErrHTTP, err)
	}
	defer hres.Body.Close()

	err = verifyStatusCode(hres, []int{http.StatusNoContent})
	if err != nil {
		return hres, err
	}

	return hres, nil
}

func DownloadSubmissionsAction(ctx context.Context, agent *agent.Agent, courseID, classID string) (*http.Response, []byte, error) {
	hres, err := api.DownloadSubmittedAssignments(ctx, agent, courseID, classID)
	if err != nil {
		return hres, nil, failure.NewError(fails.ErrHTTP, err)
	}
	defer hres.Body.Close()

	err = verifyStatusCode(hres, []int{http.StatusOK, http.StatusNotModified})
	if err != nil {
		return hres, nil, err
	}

	data, err := io.ReadAll(hres.Body)
	if err != nil {
		return nil, nil, failure.NewError(fails.ErrHTTP, err)
	}

	return hres, data, nil
}

func PostGradeAction(ctx context.Context, agent *agent.Agent, courseID, classID string, scores []StudentScore) (*http.Response, error) {
	req := make([]api.RegisterScoreRequestContent, 0, len(scores))
	for _, v := range scores {
		req = append(req, api.RegisterScoreRequestContent{
			UserCode: v.code,
			Score:    v.score,
		})
	}
	hres, err := api.RegisterScores(ctx, agent, courseID, classID, req)
	if err != nil {
		return hres, failure.NewError(fails.ErrHTTP, err)
	}
	defer hres.Body.Close()

	err = verifyStatusCode(hres, []int{http.StatusNoContent})
	if err != nil {
		return hres, err
	}

	return hres, nil
}

func SetCourseStatusInProgressAction(ctx context.Context, agent *agent.Agent, courseID string) (*http.Response, error) {
	return setCourseStatusAction(ctx, agent, courseID, api.StatusInProgress)
}

func SetCourseStatusClosedAction(ctx context.Context, agent *agent.Agent, courseID string) (*http.Response, error) {
	return setCourseStatusAction(ctx, agent, courseID, api.StatusClosed)
}

func setCourseStatusAction(ctx context.Context, agent *agent.Agent, courseID string, status api.CourseStatus) (*http.Response, error) {
	hres, err := api.SetCourseStatus(ctx, agent, courseID, status)
	if err != nil {
		return hres, failure.NewError(fails.ErrHTTP, err)
	}
	defer hres.Body.Close()

	err = verifyStatusCode(hres, []int{http.StatusOK})
	if err != nil {
		return hres, err
	}

	return hres, nil
}

func AccessTopPageAction(ctx context.Context, agent *agent.Agent) (*http.Response, agent.Resources, error) {
	hres, resources, err := api.BrowserAccess(ctx, agent, "")
	if err != nil {
		return nil, nil, failure.NewError(fails.ErrHTTP, err)
	}

	err = verifyStatusCode(hres, []int{http.StatusOK, http.StatusNotModified})
	if err != nil {
		return hres, nil, err
	}

	return hres, resources, nil
}

func AccessTopPageActionWithoutCache(ctx context.Context, agent *agent.Agent) (*http.Response, agent.Resources, error) {
	hres, resources, err := api.BrowserAccess(ctx, agent, "")
	if err != nil {
		return nil, nil, failure.NewError(fails.ErrHTTP, err)
	}

	// 検証用として200のみ許可
	err = verifyStatusCode(hres, []int{http.StatusOK})
	if err != nil {
		return hres, nil, err
	}

	return hres, resources, nil
}
