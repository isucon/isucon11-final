package scenario

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/isucon/isucon11-final/benchmarker/api"
	"github.com/isucon/isucon11-final/benchmarker/fails"
	"github.com/isucon/isucon11-final/benchmarker/model"

	"github.com/isucon/isucandar/failure"
)

// verify.go
// apiパッケージのレスポンス検証を行うもの
// http.Responseと検証に必要なデータを受け取ってerrorを返す
// param: http.Response, 検証用modelオブジェクト
// return: error

func errInvalidStatusCode(res *http.Response, expected []int) error {
	str := ""
	for _, v := range expected {
		str += strconv.Itoa(v) + ","
	}
	str = str[:len(str)-1]
	return failure.NewError(fails.ErrInvalidStatus, fmt.Errorf("期待するHTTPステータスコード以外が返却されました. %s: %s, expected: %s, actual: %d", res.Request.Method, res.Request.URL.Path,
		str, res.StatusCode))
}

func errInvalidResponse(message string, args ...interface{}) error {
	return failure.NewError(fails.ErrApplication, fmt.Errorf(message, args...))
}

func verifyStatusCode(res *http.Response, allowedStatusCodes []int) error {
	for _, code := range allowedStatusCodes {
		if res.StatusCode == code {
			return nil
		}
	}
	return errInvalidStatusCode(res, allowedStatusCodes)
}

func verifyGrades(res *api.GetGradeResponse) error {
	// TODO: modelとして何を渡すか
	// TODO: 成績のverify
	return nil
}

func verifySearchCourseResult(res []*api.GetCourseDetailResponse) error {
	// TODO: modelとして何を渡すか
	// TODO: 科目検索結果のverify
	return nil
}

func verifyAnnouncement(res *api.AnnouncementResponse, announcementStatus model.AnnouncementStatus) error {
	if res.ID != announcementStatus.Announcement.ID {
		return errInvalidResponse("お知らせIDが期待する値と一致しません")
	}

	if res.CourseID != announcementStatus.Announcement.CourseID {
		return errInvalidResponse("お知らせの講義IDが期待する値と一致しません")
	}

	if res.CourseName != announcementStatus.Announcement.CourseName {
		return errInvalidResponse("お知らせの講義名が期待する値と一致しません")
	}

	if res.Title != announcementStatus.Announcement.Title {
		return errInvalidResponse("お知らせのタイトルが期待する値と一致しません")
	}

	if res.Message != announcementStatus.Announcement.Message {
		return errInvalidResponse("お知らせのメッセージが期待する値と一致しません")
	}

	if res.Unread != announcementStatus.Unread {
		return errInvalidResponse("お知らせの未読/既読状態が期待する値と一致しません")
	}

	if res.CreatedAt != announcementStatus.Announcement.CreatedAt {
		return errInvalidResponse("お知らせの生成時刻が期待する値と一致しません")
	}

	return nil
}

func verifyAnnouncements(res *api.GetAnnouncementsResponse) error {
	// TODO: modelとして何を渡すか
	// TODO: お知らせ一覧のverify
	return nil
}

func verifyClass(res *api.GetClassResponse, class *model.Class) error {
	if res.ID != class.ID {
		return errInvalidResponse("講義IDが期待する値と一致しません")
	}

	if res.Title != class.Title {
		return errInvalidResponse("講義のタイトルが期待する値と一致しません")
	}

	if res.Description != class.Desc {
		return errInvalidResponse("講義の説明文が期待する値と一致しません")
	}

	if res.Part != class.Part {
		return errInvalidResponse("講義のパートが期待する値と一致しません")
	}

	// TODO: SubmissionClosedAtの検証

	return nil
}

func verifyClasses(res []*api.GetClassResponse, classes []*model.Class) error {
	if len(res) != len(classes) {
		return errInvalidResponse("講義数が期待する数と一致しません")
	}

	if len(res) > 0 {
		// 最後に追加された講義だけ中身を検証する
		return verifyClass(res[len(res)-1], classes[len(classes)-1])
	}

	return nil
}

func verifyAssignments(assignmentsData []byte) error {
	// TODO: modelとして何を渡すか
	// TODO: ダウンロードした課題データの検証
	return nil
}
