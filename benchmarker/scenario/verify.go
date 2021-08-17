package scenario

import (
	"archive/zip"
	"bytes"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

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

// TODO: 決め打ちではなく外から指定できるようにする
const (
	searchCourseVerifyRate = 0.2
	assignmentsVerifyRate  = 0.2
)

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

func verifyGrades(res *api.GetGradeResponse, courses []*model.Course, userCode string) []error {
	// summaryはcreditが検証できそうな気がするけどめんどくさいのでしてない

	errs := make([]error, 0)

	simpleCourseResults := make(map[string]*model.SimpleCourseResult, len(courses))
	for _, course := range courses {
		classScore := course.CollectUserScores(userCode)
		simpleCourseResults[course.Code] = model.NewSimpleCourseResult(course.Name, course.Code, model.CalculateTotalScore(classScore), classScore)
	}

	if len(simpleCourseResults) != len(res.CourseResults) {
		errs = append(errs, errInvalidResponse("成績確認でのコース結果の数が一致しません"))
		return errs
	}
	for _, resCourseResult := range res.CourseResults {
		if _, ok := simpleCourseResults[resCourseResult.Code]; !ok {
			// ここには来ないはず
			panic("cannot find course")
		}

		courseResultErrs := verifySimpleCourseResult(simpleCourseResults[resCourseResult.Code], &resCourseResult)
		if len(courseResultErrs) > 0 {
			errs = append(errs, courseResultErrs...)
			return errs
		}

	}

	return nil
}

func verifySimpleCourseResult(expected *model.SimpleCourseResult, res *api.CourseResult) []error {
	errs := make([]error, 0)
	if expected.Name != res.Name {
		errs = append(errs, errInvalidResponse("成績確認結果のコース名が違います"))
		AdminLogger.Println(fmt.Printf("expected: %s, actual: %s", expected.Name, res.Name))
		return errs
	}

	if expected.Code != res.Code {
		errs = append(errs, errInvalidResponse("成績確認の生徒のCodeが一致しません"))
		AdminLogger.Println(fmt.Printf("expected: %s, actual: %s", expected.Code, res.Code))
		return errs
	}

	if expected.TotalScore != res.TotalScore {
		errs = append(errs, errInvalidResponse("成績確認のコースのトータルスコアが一致しません"))
		AdminLogger.Println(fmt.Printf("expected: %d, actual: %d", expected.TotalScore, res.TotalScore))
		return errs
	}

	if len(expected.ClassScores) != len(res.ClassScores) {
		errs = append(errs, errInvalidResponse("成績確認でのクラスの数が一致しません"))
		AdminLogger.Println(fmt.Printf("expected: %d, actual: %d\n", len(expected.ClassScores), len(res.ClassScores)))
		return errs
	}

	for i := 0; i < len(res.ClassScores); i++ {
		scoreErrs := verifyClassScores(expected.ClassScores[i], &res.ClassScores[len(res.ClassScores)-i-1])
		if len(scoreErrs) > 0 {
			errs = append(errs, scoreErrs...)
			return errs
		}
	}

	return errs
}

func verifyClassScores(expected *model.ClassScore, res *api.ClassScore) []error {
	errs := make([]error, 0)

	if expected.ClassID != res.ClassID {
		errs = append(errs, errInvalidResponse("成績確認でのクラスのIDが一致しません"))
		AdminLogger.Println("expected: ", expected.ClassID, "actual: ", res.ClassID)
		return errs
	}
	if expected.Part != res.Part {
		errs = append(errs, errInvalidResponse("成績確認でのクラスのpartが一致しません"))
		AdminLogger.Println("expected: ", expected.Part, "actual: ", res.Part)
		return errs
	}

	if expected.Score != res.Score {
		errs = append(errs, errInvalidResponse("成績確認でのクラスのスコアが一致しません"))
	}

	if expected.Title != res.Title {
		errs = append(errs, errInvalidResponse("成績確認でのクラスのスコアのタイトルが一致しません"))
	}

	return errs
}

func verifySearchCourseResult(res *api.GetCourseDetailResponse, param *model.SearchCourseParam) error {
	if param.Type != "" && res.Type != api.CourseType(param.Type) {
		return errInvalidResponse("科目検索結果に検索条件のタイプと一致しない科目が含まれています")
	}

	if param.Credit != 0 && res.Credit != uint8(param.Credit) {
		return errInvalidResponse("科目検索結果に検索条件の単位数と一致しない科目が含まれています")
	}

	if param.Teacher != "" && res.Teacher != param.Teacher {
		return errInvalidResponse("科目検索結果に検索条件の講師と一致しない科目が含まれています")
	}

	// resは1-6, paramは0-5
	if param.Period != -1 && res.Period != uint8(param.Period+1) {
		return errInvalidResponse("科目検索結果に検索条件の時限と一致しない科目が含まれています")
	}

	if param.DayOfWeek != -1 && res.DayOfWeek != api.DayOfWeekTable[param.DayOfWeek] {
		return errInvalidResponse("科目検索結果に検索条件の曜日と一致しない科目が含まれています")
	}

	// 以下の条件のいずれかを満たしたものがヒットする
	// - Nameに指定キーワードがすべて含まれている
	// - Keywordに指定キーワードがすべて含まれている
	isNameHit := true
	isKeywordsHit := true
	for _, keyword := range param.Keywords {
		if !strings.Contains(res.Name, keyword) {
			isNameHit = false
		}
		if !strings.Contains(res.Keywords, keyword) {
			isKeywordsHit = false
		}
	}

	if !isNameHit && !isKeywordsHit {
		return errInvalidResponse("科目検索結果に検索条件のキーワードにヒットしない科目が含まれています")
	}

	return nil
}

func verifySearchCourseResults(res []*api.GetCourseDetailResponse, param *model.SearchCourseParam) []error {
	errs := make([]error, 0)
	for _, course := range res {
		if rand.Float64() < searchCourseVerifyRate {
			if err := verifySearchCourseResult(course, param); err != nil {
				errs = append(errs, err)
			}
		}
	}

	// CreatedAtの降順でソートされているか
	for i := 0; i < len(res)-1; i++ {
		if res[i].Code > res[i+1].Code {
			errs = append(errs, errInvalidResponse("科目検索結果の順序が不正です"))
			break
		}
	}

	return errs
}

func verifyAnnouncement(res *api.AnnouncementResponse, announcementStatus *model.AnnouncementStatus) error {
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

// お知らせ一覧の中身の検証
// TODO: ヘルパ関数作ってverifyAnnouncementとまとめても良いかも
func verifyAnnouncementsContent(res *api.AnnouncementResponse, announcementStatus *model.AnnouncementStatus) error {
	if res.CourseID != announcementStatus.Announcement.CourseID {
		return errInvalidResponse("お知らせの講義IDが期待する値と一致しません")
	}

	if res.CourseName != announcementStatus.Announcement.CourseName {
		return errInvalidResponse("お知らせの講義名が期待する値と一致しません")
	}

	if res.Title != announcementStatus.Announcement.Title {
		return errInvalidResponse("お知らせのタイトルが期待する値と一致しません")
	}

	if res.Unread != announcementStatus.Unread {
		return errInvalidResponse("お知らせの未読/既読状態が期待する値と一致しません")
	}

	if res.CreatedAt != announcementStatus.Announcement.CreatedAt {
		return errInvalidResponse("お知らせの生成時刻が期待する値と一致しません")
	}

	return nil
}

func verifyAnnouncements(res *api.GetAnnouncementsResponse, student *model.Student) []error {
	errs := make([]error, 0)

	// リストの中身の検証
	// MEMO: ランダムで数件チェックにしてもいいかも
	// MEMO: unreadだけ返すとハックできそう
	for _, announcement := range res.Announcements {
		announcementStatus := student.GetAnnouncement(announcement.ID)
		if announcementStatus == nil {
			// webappでは認識されているが、ベンチではまだ認識されていないお知らせ
			// load中には検証できないのでskip
			continue
		}

		if err := verifyAnnouncementsContent(&announcement, announcementStatus); err != nil {
			errs = append(errs, err)
		}
	}

	// CreatedAtの降順でソートされているか
	for i := 0; i < len(res.Announcements)-1; i++ {
		if res.Announcements[i].CreatedAt < res.Announcements[i+1].CreatedAt {
			errs = append(errs, errInvalidResponse("お知らせの順序が不正です"))
			break
		}
	}

	// MEMO: res.UnreadCountはload中には検証できなさそう

	return errs
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
	// TODO: Submittedの検証

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

func verifyAssignments(assignmentsData []byte, class *model.Class) error {
	if rand.Float64() < assignmentsVerifyRate {
		r, err := zip.NewReader(bytes.NewReader(assignmentsData), int64(len(assignmentsData)))
		if err != nil {
			return errInvalidResponse("課題zipの展開に失敗しました")
		}

		downloadedAssignments := make(map[string]uint32)
		for _, f := range r.File {
			rc, err := f.Open()
			if err != nil {
				return errInvalidResponse("課題zipのデータ読み込みに失敗しました")
			}
			assignmentData, err := ioutil.ReadAll(rc)
			rc.Close()
			if err != nil {
				return errInvalidResponse("課題zipのデータ読み込みに失敗しました")
			}
			downloadedAssignments[f.Name] = crc32.ChecksumIEEE(assignmentData)
		}

		// mapのサイズが等しく、ダウンロードされた課題がすべて実際に提出した課題ならば、ダウンロードされた課題と提出した課題は集合として等しい
		if len(downloadedAssignments) != class.GetSubmittedCount() {
			return errInvalidResponse("課題zipに含まれるファイルの数が期待する値と一致しません")
		}

		for name, checksumDownloaded := range downloadedAssignments {
			checksumSubmitted, exists := class.GetAssignmentChecksum(name)
			if !exists {
				return errInvalidResponse("課題を提出していない学生のファイルが課題zipに含まれています")
			} else if checksumDownloaded != checksumSubmitted {
				return errInvalidResponse("ダウンロードされた課題が提出された課題と一致しません")
			}
		}
	}

	return nil
}
