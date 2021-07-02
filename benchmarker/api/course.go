package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"

	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/fails"
)

type addClassRequest struct {
	Title       string `json:"id"`
	Description string `json:"description"`
}
type addClassResponse struct {
	ID string `json:"id"`
}

func AddClass(ctx context.Context, a *agent.Agent, courseID, title, desc string) (string, error) {
	reqBody, err := json.Marshal(&addClassRequest{
		Title:       title,
		Description: desc,
	})
	if err != nil {
		return "", failure.NewError(fails.ErrCritical, err)
	}

	req, err := a.POST(fmt.Sprintf("/api/courses/%s/classes", courseID), bytes.NewBuffer(reqBody))
	if err != nil {
		return "", failure.NewError(fails.ErrCritical, err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := a.Do(ctx, req)
	if err != nil {
		return "", failure.NewError(fails.ErrHTTP, err)
	}
	defer res.Body.Close()

	if err := assertStatusCode(res, http.StatusOK); err != nil {
		return "", err
	}

	var respObj addClassResponse
	err = json.NewDecoder(res.Body).Decode(&respObj)
	if err != nil {
		return "", failure.NewError(fails.ErrHTTP, fmt.Errorf(
			"JSONのパースに失敗しました (%s: %s)", res.Request.Method, res.Request.URL.Path,
		))
	}
	return respObj.ID, nil
}

type addDocResponse struct {
	ID string `json:"id"`
}

func AddDocument(ctx context.Context, a *agent.Agent, courseID, classID, docName string, docData []byte) (string, error) {
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)

	mh := textproto.MIMEHeader{}
	mh.Set("Content-Type", http.DetectContentType(docData))
	mh.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`, "files", docName))
	fw, err := mw.CreatePart(mh)
	if err != nil {
		return "", failure.NewError(fails.ErrCritical, err)
	}
	_, err = io.Copy(fw, bytes.NewBuffer(docData))
	if err != nil {
		return "", failure.NewError(fails.ErrCritical, err)
	}

	req, err := a.POST(fmt.Sprintf("/api/courses/%s/classes/%s/documents", courseID, classID), body)
	if err != nil {
		return "", failure.NewError(fails.ErrCritical, err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res, err := a.Do(ctx, req)
	if err != nil {
		return "", failure.NewError(fails.ErrHTTP, err)
	}
	defer res.Body.Close()

	if err := assertStatusCode(res, http.StatusOK); err != nil {
		return "", err
	}

	var respObj addDocResponse
	err = json.NewDecoder(res.Body).Decode(&respObj)
	if err != nil {
		return "", failure.NewError(fails.ErrHTTP, fmt.Errorf(
			"JSONのパースに失敗しました (%s: %s)", res.Request.Method, res.Request.URL.Path,
		))
	}
	return respObj.ID, nil
}

type docIDListResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func FetchDocumentIDList(ctx context.Context, a *agent.Agent, courseID, classID string) ([]string, error) {
	req, err := a.GET(fmt.Sprintf("/api/courses/%s/classes/%s/documents", courseID, classID))
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}
	res, err := a.Do(ctx, req)
	if err != nil {
		return nil, failure.NewError(fails.ErrHTTP, err)
	}
	defer res.Body.Close()

	if err := assertStatusCode(res, http.StatusOK); err != nil {
		return nil, err
	}

	var idListResponse []docIDListResponse
	err = json.NewDecoder(res.Body).Decode(&idListResponse)
	if err != nil {
		return nil, failure.NewError(fails.ErrHTTP, fmt.Errorf(
			"JSONのパースに失敗しました (%s: %s)", res.Request.Method, res.Request.URL.Path,
		))
	}
	idList := make([]string, len(idListResponse))
	for _, resp := range idListResponse {
		idList = append(idList, resp.ID)
	}
	return idList, nil
}

func FetchDocument(ctx context.Context, a *agent.Agent, courseID, classID, docID string) ([]byte, error) {
	req, err := a.GET(fmt.Sprintf("/api/courses/%s/classes/%s/documents/%s", courseID, classID, docID))
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}
	res, err := a.Do(ctx, req)
	if err != nil {
		return nil, failure.NewError(fails.ErrHTTP, err)
	}
	defer res.Body.Close()

	if err := assertStatusCode(res, http.StatusOK); err != nil {
		return nil, err
	}
	// 基本的にレスポンスはapplication/jsonだろうからチェックしてないけどpdfは特別なので確認する
	if err := assertContentType(res, "application/pdf"); err != nil {
		return nil, failure.NewError(fails.ErrHTTP, err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, failure.NewError(fails.ErrHTTP, err)
	}
	return body, nil
}

type attendCodeResponse struct {
	Code string `json:"code"`
}

func GetAttendanceCode(ctx context.Context, a *agent.Agent, courseID, classID string) (string, error) {
	req, err := a.GET(fmt.Sprintf("/api/courses/%s/classes/%s/attendance_code", courseID, classID))
	if err != nil {
		return "", failure.NewError(fails.ErrCritical, err)
	}
	res, err := a.Do(ctx, req)
	if err != nil {
		return "", failure.NewError(fails.ErrHTTP, err)
	}
	defer res.Body.Close()

	if err := assertStatusCode(res, http.StatusOK); err != nil {
		return "", err
	}

	var codeResponse attendCodeResponse
	err = json.NewDecoder(res.Body).Decode(&codeResponse)
	if err != nil {
		return "", failure.NewError(fails.ErrHTTP, fmt.Errorf(
			"JSONのパースに失敗しました (%s: %s)", res.Request.Method, res.Request.URL.Path,
		))
	}
	return codeResponse.Code, nil
}

type attendStudentResponse struct {
	ID         string `json:"user_id"`
	AttendedAt int64  `json:"attended_at"`
}

func GetAttendanceStudentIDs(ctx context.Context, a *agent.Agent, courseID, classID string) ([]string, error) {
	req, err := a.GET(fmt.Sprintf("/api/courses/%s/classes/%s/attendances", courseID, classID))
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}
	res, err := a.Do(ctx, req)
	if err != nil {
		return nil, failure.NewError(fails.ErrHTTP, err)
	}
	defer res.Body.Close()

	if err := assertStatusCode(res, http.StatusOK); err != nil {
		return nil, err
	}

	var resObj []attendStudentResponse
	err = json.NewDecoder(res.Body).Decode(&resObj)
	if err != nil {
		return nil, failure.NewError(fails.ErrHTTP, fmt.Errorf(
			"JSONのパースに失敗しました (%s: %s)", res.Request.Method, res.Request.URL.Path,
		))
	}
	r := make([]string, 0)
	for _, resp := range resObj {
		r = append(r, resp.ID)
	}
	return r, nil
}

type addAssignmentRequest struct {
	Name     string `json:"name"`
	Desc     string `json:"description"`
	Deadline int64  `json:"deadline"`
}
type addAssignmentResponse struct {
	ID string `json:"id"`
}

func AddAssignments(ctx context.Context, a *agent.Agent, courseID, classID, name, desc string, deadline int64) (string, error) {
	reqBody, err := json.Marshal(&addAssignmentRequest{
		Name:     name,
		Desc:     desc,
		Deadline: deadline,
	})
	if err != nil {
		return "", failure.NewError(fails.ErrCritical, err)
	}
	req, err := a.POST(fmt.Sprintf("/api/courses/%s/classes/%s/assignments", courseID, classID), bytes.NewBuffer(reqBody))
	if err != nil {
		return "", failure.NewError(fails.ErrCritical, err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := a.Do(ctx, req)
	if err != nil {
		return "", failure.NewError(fails.ErrHTTP, err)
	}
	defer res.Body.Close()

	if err := assertStatusCode(res, http.StatusOK); err != nil {
		return "", err
	}

	var resp addAssignmentResponse
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		return "", failure.NewError(fails.ErrHTTP, fmt.Errorf(
			"JSONのパースに失敗しました (%s: %s)", res.Request.Method, res.Request.URL.Path,
		))
	}
	return resp.ID, nil
}

func SubmitAssignment(ctx context.Context, a *agent.Agent, courseID, assignmentID, fileName string, data []byte) error {
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)

	mh := textproto.MIMEHeader{}
	mh.Set("Content-Type", http.DetectContentType(data))
	mh.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`, "file", fileName))
	fw, err := mw.CreatePart(mh)
	if err != nil {
		return failure.NewError(fails.ErrCritical, err)
	}
	_, err = io.Copy(fw, bytes.NewBuffer(data))
	if err != nil {
		return failure.NewError(fails.ErrCritical, err)
	}

	req, err := a.POST(fmt.Sprintf("/api/courses/%s/assignments/%s", courseID, assignmentID), body)
	if err != nil {
		return failure.NewError(fails.ErrCritical, err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res, err := a.Do(ctx, req)
	if err != nil {
		return failure.NewError(fails.ErrHTTP, err)
	}
	defer res.Body.Close()

	if err := assertStatusCode(res, http.StatusOK); err != nil {
		return err
	}
	return nil
}

func ExportSubmissions(ctx context.Context, a *agent.Agent, courseID, assignmentID string) ([]byte, error) {
	req, err := a.GET(fmt.Sprintf("/api/courses/%s/assignments/%s/export", courseID, assignmentID))
	if err != nil {
		return nil, failure.NewError(fails.ErrCritical, err)
	}
	res, err := a.Do(ctx, req)
	if err != nil {
		return nil, failure.NewError(fails.ErrHTTP, err)
	}
	defer res.Body.Close()

	if err := assertStatusCode(res, http.StatusOK); err != nil {
		return nil, err
	}
	if err := assertContentType(res, "application/zip"); err != nil {
		return nil, failure.NewError(fails.ErrHTTP, err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, failure.NewError(fails.ErrHTTP, err)
	}
	return body, nil
}
