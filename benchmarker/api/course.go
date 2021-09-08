package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"

	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"

	"github.com/isucon/isucon11-final/benchmarker/fails"
)

type CourseType string

const (
	CourseTypeLiberalArts   CourseType = "liberal-arts"
	CourseTypeMajorSubjects CourseType = "major-subjects"
)

type DayOfWeek string

var DayOfWeekTable = []DayOfWeek{
	"monday",
	"tuesday",
	"wednesday",
	"thursday",
	"friday",
}

type CourseStatus string

const (
	StatusRegistration CourseStatus = "registration"
	StatusInProgress   CourseStatus = "in-progress"
	StatusClosed       CourseStatus = "closed"
)

type SearchCourseRequest struct {
	Type      CourseType
	Credit    uint8
	Teacher   string
	Period    uint8
	DayOfWeek DayOfWeek
	Keywords  string
	Status    CourseStatus
}

type GetCourseDetailResponse struct {
	ID          string       `json:"id"`
	Code        string       `json:"code"`
	Type        CourseType   `json:"type"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Credit      uint8        `json:"credit"`
	Period      uint8        `json:"period"`
	DayOfWeek   DayOfWeek    `json:"day_of_week"`
	Teacher     string       `json:"teacher"`
	Status      CourseStatus `json:"status"`
	Keywords    string       `json:"keywords"`
}

func SearchCourse(ctx context.Context, a *agent.Agent, param *SearchCourseRequest) (*http.Response, error) {
	req, err := a.GET("/api/courses")
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}
	query := req.URL.Query()
	if param.Type != "" {
		query.Add("type", string(param.Type))
	}
	if param.Credit != 0 {
		query.Add("credit", strconv.Itoa((int(param.Credit))))
	}
	if param.Teacher != "" {
		query.Add("teacher", param.Teacher)
	}
	if param.Period != 0 {
		query.Add("period", strconv.Itoa((int(param.Period))))
	}
	if param.DayOfWeek != "" {
		query.Add("day_of_week", string(param.DayOfWeek))
	}
	if param.Keywords != "" {
		query.Add("keywords", param.Keywords)
	}
	if param.Status != "" {
		query.Add("status", string(param.Status))
	}
	req.URL.RawQuery = query.Encode()

	return a.Do(ctx, req)
}

func SearchCourseWithNext(ctx context.Context, a *agent.Agent, pathParam string) (*http.Response, error) {
	req, err := a.GET(pathParam)
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	return a.Do(ctx, req)
}

func GetCourseDetail(ctx context.Context, a *agent.Agent, courseID string) (*http.Response, error) {
	path := fmt.Sprintf("/api/courses/%s", courseID)

	req, err := a.GET(path)
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	return a.Do(ctx, req)
}

type AddCourseRequest struct {
	Code        string     `json:"code"`
	Type        CourseType `json:"type"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Credit      int        `json:"credit"`
	Period      int        `json:"period"`
	DayOfWeek   DayOfWeek  `json:"day_of_week"`
	Keywords    string     `json:"keywords"`
}

type AddCourseResponse struct {
	ID string `json:"id"`
}

func AddCourse(ctx context.Context, a *agent.Agent, courseRequest AddCourseRequest) (*http.Response, error) {
	body, err := json.Marshal(courseRequest)
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	req, err := a.POST("/api/courses", bytes.NewReader(body))
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}
	req.Header.Set("Content-Type", "application/json")

	return a.Do(ctx, req)
}

type SetCourseStatusRequest struct {
	Status CourseStatus `json:"status"`
}

func SetCourseStatus(ctx context.Context, a *agent.Agent, courseID string, status CourseStatus) (*http.Response, error) {
	body, err := json.Marshal(SetCourseStatusRequest{
		Status: status,
	})
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	req, err := a.PUT(fmt.Sprintf("/api/courses/%s/status", courseID), bytes.NewReader(body))
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}
	req.Header.Set("Content-Type", "application/json")
	return a.Do(ctx, req)
}

type AddClassRequest struct {
	Part        uint8  `json:"part"`
	Title       string `json:"title"`
	Description string `json:"description"`
}
type AddClassResponse struct {
	ClassID string `json:"class_id"`
}

func AddClass(ctx context.Context, a *agent.Agent, courseID string, classRequest AddClassRequest) (*http.Response, error) {
	body, err := json.Marshal(classRequest)
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	req, err := a.POST(fmt.Sprintf("/api/courses/%s/classes", courseID), bytes.NewReader(body))
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}
	req.Header.Set("Content-Type", "application/json")

	return a.Do(ctx, req)
}

type GetClassResponse struct {
	ID               string `json:"id"`
	Part             uint8  `json:"part"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	SubmissionClosed bool   `json:"submission_closed"`
	Submitted        bool   `json:"submitted"`
}

func GetClasses(ctx context.Context, a *agent.Agent, courseID string) (*http.Response, error) {
	path := fmt.Sprintf("/api/courses/%s/classes", courseID)

	req, err := a.GET(path)
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	return a.Do(ctx, req)
}

func SubmitAssignment(ctx context.Context, a *agent.Agent, courseID, classID, fileName string, data []byte) (*http.Response, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	header := textproto.MIMEHeader{}
	header.Set("Content-Type", http.DetectContentType(data))
	header.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`, "file", fileName))

	part, err := w.CreatePart(header)
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}
	_, err = io.Copy(part, bytes.NewBuffer(data))
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	contentType := w.FormDataContentType()

	err = w.Close()
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	req, err := a.POST(fmt.Sprintf("/api/courses/%s/classes/%s/assignments", courseID, classID), &body)
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	req.Header.Set("Content-Type", contentType)

	return a.Do(ctx, req)
}

type RegisterScoreRequestContent struct {
	UserCode string `json:"user_code"`
	Score    int    `json:"score"`
}

func RegisterScores(ctx context.Context, a *agent.Agent, courseID, classID string, scores []RegisterScoreRequestContent) (*http.Response, error) {
	body, err := json.Marshal(scores)
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}
	path := fmt.Sprintf("/api/courses/%s/classes/%s/assignments/scores", courseID, classID)

	req, err := a.PUT(path, bytes.NewReader(body))
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	req.Header.Set("Content-Type", "application/json")
	return a.Do(ctx, req)
}

func DownloadSubmittedAssignments(ctx context.Context, a *agent.Agent, courseID, classID string) (*http.Response, error) {
	path := fmt.Sprintf("/api/courses/%s/classes/%s/assignments/export", courseID, classID)

	req, err := a.GET(path)
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}

	return a.Do(ctx, req)
}
