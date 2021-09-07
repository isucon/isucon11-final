package scenario

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"

	"github.com/isucon/isucon11-final/benchmarker/api"
	"github.com/isucon/isucon11-final/benchmarker/fails"
	"github.com/isucon/isucon11-final/benchmarker/model"
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

func verifyInitialize(res api.InitializeResponse) error {
	if res.Language == "" {
		return errInvalidResponse("initialize のレスポンスに利用言語が設定されていません")
	}
	return nil
}

func verifyMe(res *api.GetMeResponse, expectedUserAccount *model.UserAccount, expectedAdminFlag bool) error {
	if res.Code != expectedUserAccount.Code {
		return errInvalidResponse("学籍番号が期待する値と一致しません")
	}

	if res.IsAdmin != expectedAdminFlag {
		return errInvalidResponse("管理者フラグが期待する値と一致しません")
	}

	return nil
}

// この返り値の map の value の interfaceは
// すでにclosedなコースについてはCourseResult に、
// そうでない場合は、SimpleCourseResult になる
func collectVerifyGradesData(student *model.Student) map[string]interface{} {
	courses := student.Courses()
	simpleCourseResults := make(map[string]interface{}, len(courses))
	for _, course := range courses {
		if course.Status() == api.StatusClosed {
			simpleCourseResults[course.Code] = course.CalcCourseResultByStudentCode(student.Code)
		} else {
			classScore := course.CollectSimpleClassScores(student.Code)
			simpleCourseResults[course.Code] = model.NewSimpleCourseResult(course.Name, course.Code, classScore)
		}
	}

	return simpleCourseResults
}

func verifyGrades(expected map[string]interface{}, res *api.GetGradeResponse) error {
	// summaryはcreditが検証できそうな気がするけどめんどくさいのでしてない
	if len(expected) != len(res.CourseResults) {
		return errInvalidResponse("成績確認でのコース結果の数が一致しません")
	}

	for _, resCourseResult := range res.CourseResults {
		if _, ok := expected[resCourseResult.Code]; !ok {
			return errInvalidResponse("成績確認で期待しないコースが含まれています")
		}

		switch v := expected[resCourseResult.Code].(type) {
		case *model.SimpleCourseResult:
			err := verifySimpleCourseResult(v, &resCourseResult)
			if err != nil {
				return err
			}
		case *model.CourseResult:
			err := validateCourseResult(v, &resCourseResult)
			if err != nil {
				return err
			}
		default:
			panic(fmt.Sprintf("expect %T or %T, actual %T", &model.SimpleCourseResult{}, &model.CourseResult{}, v))
		}

	}

	return nil
}

func verifySimpleCourseResult(expected *model.SimpleCourseResult, res *api.CourseResult) error {
	if expected.Name != res.Name {
		AdminLogger.Println(fmt.Printf("expected: %s, actual: %s", expected.Name, res.Name))
		return errInvalidResponse("成績確認結果のコース名が違います")
	}

	if expected.Code != res.Code {
		AdminLogger.Println(fmt.Printf("expected: %s, actual: %s", expected.Code, res.Code))
		return errInvalidResponse("成績確認の生徒のCodeが一致しません")
	}

	// リクエスト前の時点で登録成功しているクラスの成績は、成績レスポンスに必ず含まれている
	// そのため、追加済みクラスのスコアの数よりレスポンス内クラスのスコアの数が少ない場合はエラーとなる
	if len(expected.SimpleClassScores) > len(res.ClassScores) {
		AdminLogger.Println(fmt.Printf("expected: %d, actual: %d", len(expected.SimpleClassScores), len(res.ClassScores)))
		return errInvalidResponse("成績確認のクラスのスコアの数が正しくありません")
	}

	// 最新のクラスの成績はまだ更新されているか判断できない
	// 一つ前のクラスの処理が終わらないと次のクラスの処理は始まらないので、
	// 一つ前のクラスまでの成績は正しくなっているはず
	// https://github.com/isucon/isucon11-final/pull/293#discussion_r690946334
	for i := 0; i < len(expected.SimpleClassScores)-1; i++ {
		// webapp 側は新しい(partが大きい)classから順番に帰ってくるので古いクラスから見るようにしている
		err := verifyClassScores(expected.SimpleClassScores[i], &res.ClassScores[len(res.ClassScores)-i-1])
		if err != nil {
			return err
		}
	}

	return nil
}

func verifyClassScores(expected *model.SimpleClassScore, res *api.ClassScore) error {
	if expected.ClassID != res.ClassID {
		AdminLogger.Println("expected: ", expected.ClassID, "actual: ", res.ClassID)
		return errInvalidResponse("成績確認でのクラスのIDが一致しません")
	}
	if expected.Part != res.Part {
		AdminLogger.Println("expected: ", expected.Part, "actual: ", res.Part)
		return errInvalidResponse("成績確認でのクラスのpartが一致しません")
	}

	if expected.Title != res.Title {
		AdminLogger.Println("expected: ", expected.Title, "actual: ", res.Title)
		return errInvalidResponse("成績確認でのクラスのタイトルが一致しません")
	}

	if !((expected.Score == nil && res.Score == nil) ||
		((expected.Score != nil && res.Score != nil) && (*expected.Score == *res.Score))) {
		AdminLogger.Println("expected: ", expected.Score, "actual: ", res.Score)
		return errInvalidResponse("成績確認でのクラスのスコアが一致しません")
	}

	return nil
}

func verifyRegisteredCourse(actual *api.GetRegisteredCourseResponseContent, expected *model.Course) error {
	if actual.ID.String() != expected.ID {
		return errInvalidResponse("コースのIDが期待する値と一致しません")
	}

	if actual.Name != expected.Name {
		return errInvalidResponse("コース名が期待する値と一致しません")
	}

	if actual.Teacher != expected.Teacher().Name {
		return errInvalidResponse("コースの講師が期待する値と一致しません")
	}

	// DayOfWeekとPeriodは、ベンチのスケジュールと突き合わせている時点で一致しているはずなのでここでは検証しない

	return nil
}

func verifyRegisteredCourses(res []*api.GetRegisteredCourseResponseContent, expectedSchedule [5][6]*model.Course) error {
	// DayOfWeekの逆引きテーブル（string -> int）
	dayOfWeekIndexTable := map[api.DayOfWeek]int{
		"monday":    0,
		"tuesday":   1,
		"wednesday": 2,
		"thursday":  3,
		"friday":    4,
	}

	actualSchedule := [5][6]*api.GetRegisteredCourseResponseContent{}
	for _, resContent := range res {
		dayOfWeekIndex, ok := dayOfWeekIndexTable[resContent.DayOfWeek]
		if !ok {
			return errInvalidResponse("科目の開講曜日が不正です")
		}
		periodIndex := int(resContent.Period) - 1
		if periodIndex < 0 || periodIndex >= 6 {
			return errInvalidResponse("科目の開講時限が不正です")
		}
		if actualSchedule[dayOfWeekIndex][periodIndex] != nil {
			return errInvalidResponse("履修済み科目のリストに時限の重複が存在します")
		}

		actualSchedule[dayOfWeekIndex][periodIndex] = resContent
	}

	// 科目の終了処理は履修済み科目取得のリクエストと並列で走るため、ベンチに存在する科目(registeredSchedule)がレスポンスに存在しないことは許容する。
	// ただし、registeredScheduleは履修済み科目取得のリクエスト直前に取得してそれ以降削除されず、また履修登録は直列であるため、レスポンスに存在する科目は必ずベンチにも存在することを期待する。
	// したがって、レスポンスに含まれる科目はベンチにある科目(registeredSchedule)の部分集合であることを確認すれば十分である。
	for d := 0; d < 5; d++ {
		for p := 0; p < 6; p++ {
			if actualSchedule[d][p] != nil {
				if expectedSchedule[d][p] != nil {
					if err := verifyRegisteredCourse(actualSchedule[d][p], expectedSchedule[d][p]); err != nil {
						return err
					}
				} else {
					return errInvalidResponse("履修済み科目のリストに期待しない科目が含まれています")
				}
			}
		}
	}

	return nil
}

func verifySearchCourseResult(res *api.GetCourseDetailResponse, param *model.SearchCourseParam) error {
	if param.Type != "" && !AssertEqual("course type", api.CourseType(param.Type), res.Type) {
		return errInvalidResponse("科目検索結果に検索条件のタイプと一致しない科目が含まれています")
	}

	if param.Credit != 0 && !AssertEqual("course credit", uint8(param.Credit), res.Credit) {
		return errInvalidResponse("科目検索結果に検索条件の単位数と一致しない科目が含まれています")
	}

	if param.Teacher != "" && !AssertEqual("course teacher", param.Teacher, res.Teacher) {
		return errInvalidResponse("科目検索結果に検索条件の講師と一致しない科目が含まれています")
	}

	// resは1-6, paramは0-5
	if param.Period != -1 && !AssertEqual("course period", uint8(param.Period+1), res.Period) {
		return errInvalidResponse("科目検索結果に検索条件の時限と一致しない科目が含まれています")
	}

	if param.DayOfWeek != -1 && !AssertEqual("course DoW", api.DayOfWeekTable[param.DayOfWeek], res.DayOfWeek) {
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
		AdminLogger.Printf("name / keyword not match: expect: %v, acutual: %s", param.Keywords, res.Keywords)
		return errInvalidResponse("科目検索結果に検索条件のキーワードにヒットしない科目が含まれています")
	}

	return nil
}

func verifySearchCourseResults(res []*api.GetCourseDetailResponse, param *model.SearchCourseParam) error {
	for _, course := range res {
		if rand.Float64() < searchCourseVerifyRate {
			if err := verifySearchCourseResult(course, param); err != nil {
				return err
			}
		}
	}

	// CreatedAtの降順でソートされているか
	for i := 0; i < len(res)-1; i++ {
		if res[i].Code > res[i+1].Code {
			return errInvalidResponse("科目検索結果の順序が不正です")
		}
	}

	return nil
}

func verifyCourseDetail(actual *api.GetCourseDetailResponse, expected *model.Course) error {
	if actual.Code != expected.Code {
		return errInvalidResponse("科目のコードが期待する値と一致しません")
	}

	if actual.Type != api.CourseType(expected.Type) {
		return errInvalidResponse("科目のタイプが期待する値と一致しません")
	}

	if actual.Name != expected.Name {
		return errInvalidResponse("科目名が期待する値と一致しません")
	}

	if actual.Description != expected.Description {
		return errInvalidResponse("科目の詳細が期待する値と一致しません")
	}

	if actual.Credit != uint8(expected.Credit) {
		return errInvalidResponse("科目の単位数が期待する値と一致しません")
	}

	if actual.Period != uint8(expected.Period+1) {
		return errInvalidResponse("科目の開講時限が期待する値と一致しません")
	}

	if actual.DayOfWeek != api.DayOfWeekTable[expected.DayOfWeek] {
		return errInvalidResponse("科目の開講曜日が期待する値と一致しません")
	}

	if actual.Teacher != expected.Teacher().Name {
		return errInvalidResponse("科目の講師が期待する値と一致しません")
	}

	if actual.Keywords != expected.Keywords {
		return errInvalidResponse("科目のキーワードが期待する値と一致しません")
	}

	return nil
}

func verifyAnnouncementDetail(res *api.GetAnnouncementDetailResponse, announcementStatus *model.AnnouncementStatus) error {
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

	if res.CreatedAt != announcementStatus.Announcement.CreatedAt {
		return errInvalidResponse("お知らせの生成時刻が期待する値と一致しません")
	}

	// Dirtyフラグが立っていない場合のみ、Unreadの検証を行う
	// 既読化RequestがTimeoutで中断された際、ベンチには既読が反映しないがwebapp側が既読化される可能性があるため。
	if !announcementStatus.Dirty {
		if !AssertEqual("announce unread", announcementStatus.Unread, res.Unread) {
			return errInvalidResponse("お知らせの未読/既読状態が期待する値と一致しません")
		}
	}

	return nil
}

// お知らせ一覧の中身の検証
func verifyAnnouncementsList(res *api.GetAnnouncementsResponse, expectList map[string]*model.AnnouncementStatus, verifyUnread bool) error {
	// リストの中身の検証
	for _, actual := range res.Announcements {
		expectStatus, ok := expectList[actual.ID]
		if !ok {
			// webappでは認識されているが、ベンチではまだ認識されていないお知らせ（お知らせ登録がリトライ中の場合発生）
			// load中には検証できないのでskip
			continue
		}

		if !AssertEqual("announcement list course id", expectStatus.Announcement.CourseID, actual.CourseID) {
			return errInvalidResponse("お知らせの科目IDが期待する値と一致しません")
		}
		if !AssertEqual("announcement list course name", expectStatus.Announcement.CourseName, actual.CourseName) {
			return errInvalidResponse("お知らせの科目名が期待する値と一致しません")
		}
		if !AssertEqual("announcement list title", expectStatus.Announcement.Title, actual.Title) {
			return errInvalidResponse("お知らせのタイトルが期待する値と一致しません")
		}
		if !AssertEqual("announcement create_at", expectStatus.Announcement.CreatedAt, actual.CreatedAt) {
			return errInvalidResponse("お知らせの生成時刻が期待する値と一致しません")
		}

		// Dirtyフラグが立っていない場合のみ、Unreadの検証を行う
		// 既読化RequestがTimeoutで中断された際、ベンチには既読が反映しないがwebapp側が既読化される可能性があるため。
		if verifyUnread && !expectStatus.Dirty {
			if !AssertEqual("announce unread", expectStatus.Unread, actual.Unread) {
				return errInvalidResponse("お知らせの未読/既読状態が期待する値と一致しません")
			}
		}
	}

	// CreatedAtの降順でソートされているか
	for i := 0; i < len(res.Announcements)-1; i++ {
		if res.Announcements[i].CreatedAt < res.Announcements[i+1].CreatedAt {
			return errInvalidResponse("お知らせの順序が不正です")
		}
	}

	// MEMO: res.UnreadCountはload中には検証できない

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

		// mapのサイズが等しく、提出した課題がすべてダウンロードされた課題に含まれていれば、提出した課題とダウンロードされた課題は集合として等しい
		if len(downloadedAssignments) != class.GetSubmittedCount() {
			return errInvalidResponse("課題zipに含まれるファイルの数が期待する値と一致しません")
		}

		for studentCode, submission := range class.Submissions() {
			expectedFileName := studentCode + "-" + submission.Title
			checksumDownloaded, ok := downloadedAssignments[expectedFileName]
			if !ok {
				return errInvalidResponse("提出した課題が課題zipに含まれていないか、ファイル名が間違っています")
			}
			if submission.Checksum != checksumDownloaded {
				return errInvalidResponse("提出した課題とダウンロードされた課題の内容が一致しません")
			}
		}
	}

	return nil
}

func joinURL(base *url.URL, target string) string {
	b := *base
	t, _ := url.Parse(target)
	u := b.ResolveReference(t).String()
	return u
}

func verifyPageResource(res *http.Response, resources agent.Resources) []error {
	if resources == nil && res.StatusCode != http.StatusOK {
		// 期待するリソースはstatus:200のページのみなのでそれ以外は無視する
		return []error{}
	}

	checks := []error{
		verifyResource(resources[joinURL(res.Request.URL, "/_nuxt/app.js")], "/_nuxt/app.js"),
		verifyResource(resources[joinURL(res.Request.URL, "/_nuxt/runtime.js")], "/_nuxt/runtime.js"),
		verifyResource(resources[joinURL(res.Request.URL, "/_nuxt/css/app.css")], "/_nuxt/css/app.css"),
	}

	var errs []error
	for _, err := range checks {
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func verifyResource(resource *agent.Resource, expectPath string) error {
	if resource == nil || resource.Response == nil {
		return failure.NewError(fails.ErrStaticResource, fmt.Errorf("期待するリソースが読み込まれませんでした(%s)", expectPath))
	}

	if resource.Error != nil {
		var nerr net.Error
		if failure.As(resource.Error, &nerr) {
			if nerr.Timeout() || nerr.Temporary() {
				return nerr
			}
		}
		return failure.NewError(fails.ErrStaticResource, fmt.Errorf("リソースの取得に失敗しました: %s: %v", expectPath, resource.Error))
	}

	return verifyChecksum(resource.Response, expectPath)
}

func verifyChecksum(res *http.Response, expectPath string) error {
	defer res.Body.Close()

	expected, ok := resourcesHash[expectPath]
	if !ok {
		AdminLogger.Printf("意図していないリソース(%s)への検証が発生しました。verify.goとassets.goを確認してください。", expectPath)
		return nil
	}

	err := verifyStatusCode(res, []int{http.StatusOK, http.StatusNotModified})
	if err != nil {
		return err
	}
	if res.StatusCode == http.StatusNotModified {
		return nil
	}

	hash := md5.New()
	io.Copy(hash, res.Body)
	actual := fmt.Sprintf("%x", hash.Sum(nil))

	if expected != actual {
		return failure.NewError(fails.ErrStaticResource, fmt.Errorf("期待するチェックサムと一致しません(%s)", expectPath))
	}
	return nil
}
