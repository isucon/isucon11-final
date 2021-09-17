package scenario

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"math/rand"
	"mime"
	"net"
	"net/http"
	"net/url"

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

func verifyStatusCode(hres *http.Response, allowedStatusCodes []int) error {
	for _, code := range allowedStatusCodes {
		if hres.StatusCode == code {
			return nil
		}
	}
	return fails.ErrorInvalidStatusCode(hres, allowedStatusCodes)
}

func verifyContentType(hres *http.Response, allowedMediaType string) error {
	mediaType, _, err := mime.ParseMediaType(hres.Header.Get("Content-Type"))
	if err != nil {
		return fails.ErrorInvalidContentType(fmt.Errorf("Content-Type の取得に失敗しました (%w)", err), hres)
	}

	if !AssertEqual("content type", allowedMediaType, mediaType) {
		return fails.ErrorInvalidContentType(errors.New("Content-Type が不正です"), hres)
	}

	return nil
}

func verifyInitialize(res api.InitializeResponse, hres *http.Response) error {
	if res.Language == "" {
		return fails.ErrorInvalidResponse(errors.New("initialize のレスポンスに利用言語が設定されていません"), hres)
	}

	return nil
}

func verifyMe(expected *model.UserAccount, res *api.GetMeResponse, hres *http.Response) error {
	if err := AssertEqualUserAccount(expected, res); err != nil {
		return fails.ErrorInvalidResponse(err, hres)
	}

	return nil
}

// この返り値の map の value の interfaceは
// すでにclosedな科目についてはCourseResult に、
// そうでない場合は、SimpleCourseResult になる
func collectVerifyGradesData(student *model.Student) map[string]interface{} {
	courses := student.Courses()
	courseResults := make(map[string]interface{}, len(courses))
	for _, course := range courses {
		if course.Status() == api.StatusClosed {
			courseResults[course.Code] = course.CalcCourseResultByStudentCode(student.Code)
		} else {
			classScore := course.CollectSimpleClassScores(student.Code)
			courseResults[course.Code] = model.NewSimpleCourseResult(course.Name, course.Code, classScore)
		}
	}

	return courseResults
}

func verifyGrades(expected map[string]interface{}, res *api.GetGradeResponse, hres *http.Response) error {
	// summaryはcreditが検証できそうな気がするけどめんどくさいのでしてない
	if !AssertEqual("grade courses length", len(expected), len(res.CourseResults)) {
		return fails.ErrorInvalidResponse(errors.New("成績取得での科目結果の数が一致しません"), hres)
	}

	for _, resCourseResult := range res.CourseResults {
		if _, ok := expected[resCourseResult.Code]; !ok {
			return fails.ErrorInvalidResponse(errors.New("成績取得で期待しない科目が含まれています"), hres)
		}

		switch v := expected[resCourseResult.Code].(type) {
		case *model.SimpleCourseResult:
			err := verifySimpleCourseResult(v, &resCourseResult, hres)
			if err != nil {
				return err
			}
		case *model.CourseResult:
			err := AssertEqualCourseResult(v, &resCourseResult)
			if err != nil {
				return fails.ErrorInvalidResponse(err, hres)
			}
		default:
			// 上の2種類の型しか来ないはず + 別のが来たら検証もできないしベンチがおかしいのでpanicで良い
			panic(fmt.Sprintf("expect %T or %T, actual %T", &model.SimpleCourseResult{}, &model.CourseResult{}, v))
		}
	}

	return nil
}

func verifySimpleCourseResult(expected *model.SimpleCourseResult, res *api.CourseResult, hres *http.Response) error {
	if !AssertEqual("grade courses name", expected.Name, res.Name) {
		return fails.ErrorInvalidResponse(errors.New("成績取得結果の科目名が違います"), hres)
	}

	if !AssertEqual("grade courses code", expected.Code, res.Code) {
		return fails.ErrorInvalidResponse(errors.New("成績取得の科目コードが一致しません"), hres)
	}

	// リクエスト前の時点で登録成功している講義の成績は、成績レスポンスに必ず含まれている
	// そのため、追加済み講義の採点結果の数よりレスポンス内講義の採点結果の数が少ない場合はエラーとなる
	if !AssertGreaterOrEqual("grade courses class_scores length", len(expected.SimpleClassScores), len(res.ClassScores)) {
		return fails.ErrorInvalidResponse(errors.New("成績取得の講義の採点結果の数が正しくありません"), hres)
	}

	// 最新の講義の成績はまだ更新されているか判断できない
	// 一つ前の講義の処理が終わらないと次の講義の処理は始まらないので、
	// 一つ前の講義までの成績は正しくなっているはず
	// https://github.com/isucon/isucon11-final/pull/293#discussion_r690946334
	for i := 0; i < len(expected.SimpleClassScores)-1; i++ {
		// webapp 側は新しい(partが大きい)classから順番に帰ってくるので古い講義から見るようにしている
		err := AssertEqualSimpleClassScore(expected.SimpleClassScores[i], &res.ClassScores[len(res.ClassScores)-i-1])
		if err != nil {
			return fails.ErrorInvalidResponse(err, hres)
		}
	}

	return nil
}

func verifyRegisteredCourses(expectedSchedule [5][6]*model.Course, res []*api.GetRegisteredCourseResponseContent, hres *http.Response) error {
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
			return fails.ErrorInvalidResponse(errors.New("科目の開講曜日が不正です"), hres)
		}
		periodIndex := int(resContent.Period) - 1
		if periodIndex < 0 || periodIndex >= 6 {
			return fails.ErrorInvalidResponse(errors.New("科目の開講時限が不正です"), hres)
		}
		if actualSchedule[dayOfWeekIndex][periodIndex] != nil {
			return fails.ErrorInvalidResponse(errors.New("履修済み科目のリストに時限の重複が存在します"), hres)
		}

		actualSchedule[dayOfWeekIndex][periodIndex] = resContent
	}

	// 科目の終了処理は履修済み科目取得のリクエストと並列で走るため、ベンチに存在する科目(expectedSchedule)がレスポンスに存在しないことは許容する。
	// ただし、expectedScheduleは履修済み科目取得のリクエスト直前に取得してそれ以降削除されず、また履修登録は直列であるため、レスポンスに存在する科目は必ずベンチにも存在することを期待する。
	// したがって、レスポンスに含まれる科目はベンチにある科目(expectedSchedule)の部分集合であることを確認すれば十分である。
	for d := 0; d < 5; d++ {
		for p := 0; p < 6; p++ {
			if actualSchedule[d][p] != nil {
				if expectedSchedule[d][p] != nil {
					if err := AssertEqualRegisteredCourse(expectedSchedule[d][p], actualSchedule[d][p]); err != nil {
						return fails.ErrorInvalidResponse(err, hres)
					}
				} else {
					return fails.ErrorInvalidResponse(errors.New("履修済み科目のリストに期待しない科目が含まれています"), hres)
				}
			}
		}
	}

	return nil
}

func verifyMatchCourse(res *api.GetCourseDetailResponse, param *model.SearchCourseParam, hres *http.Response) error {
	if param.Type != "" && !AssertEqual("search type", api.CourseType(param.Type), res.Type) {
		return fails.ErrorInvalidResponse(errors.New("科目検索結果に検索条件のタイプと一致しない科目が含まれています"), hres)
	}

	if param.Credit != 0 && !AssertEqual("search credit", uint8(param.Credit), res.Credit) {
		return fails.ErrorInvalidResponse(errors.New("科目検索結果に検索条件の単位数と一致しない科目が含まれています"), hres)
	}

	if param.Teacher != "" && !AssertEqual("search teacher", param.Teacher, res.Teacher) {
		return fails.ErrorInvalidResponse(errors.New("科目検索結果に検索条件の教員名と一致しない科目が含まれています"), hres)
	}

	// resは1-6, paramは0-5
	if param.Period != -1 && !AssertEqual("search period", uint8(param.Period+1), res.Period) {
		return fails.ErrorInvalidResponse(errors.New("科目検索結果に検索条件の時限と一致しない科目が含まれています"), hres)
	}

	if param.DayOfWeek != -1 && !AssertEqual("search day_of_week", api.DayOfWeekTable[param.DayOfWeek], res.DayOfWeek) {
		return fails.ErrorInvalidResponse(errors.New("科目検索結果に検索条件の曜日と一致しない科目が含まれています"), hres)
	}

	// 以下の条件のいずれかを満たしたものがヒットする
	// - Nameに指定キーワードがすべて含まれている
	// - Keywordsに指定キーワードがすべて含まれている
	if !containsAll(res.Name, param.Keywords) && !containsAll(res.Keywords, param.Keywords) {
		AdminLogger.Printf("search keywords: keywords: %v / actual name: %s, actual keywords: %s", param.Keywords, res.Name, res.Keywords)
		return fails.ErrorInvalidResponse(errors.New("科目検索結果に検索条件のキーワードにヒットしない科目が含まれています"), hres)
	}

	if param.Status != "" && !AssertEqual("search status", param.Status, res.Status) {
		return fails.ErrorInvalidResponse(errors.New("科目検索結果に検索条件の科目ステータスと一致しない科目が含まれています"), hres)
	}

	return nil
}

func verifySearchCourseResults(res []*api.GetCourseDetailResponse, param *model.SearchCourseParam, hres *http.Response) error {
	// Code の昇順でソートされているか
	for i := 0; i < len(res)-1; i++ {
		if res[i].Code > res[i+1].Code {
			return fails.ErrorInvalidResponse(errors.New("科目検索結果の順序が不正です"), hres)
		}
	}

	// 取得されたものが検索条件にヒットするか
	for _, course := range res {
		if err := verifyMatchCourse(course, param, hres); err != nil {
			return err
		}
	}

	return nil
}

func verifyCourseDetail(expected *model.Course, actual *api.GetCourseDetailResponse, hres *http.Response) error {
	// load中ではstatusが並列で更新されるので検証を行わない
	if err := AssertEqualCourse(expected, actual, false); err != nil {
		return fails.ErrorInvalidResponse(err, hres)
	}

	return nil
}

func verifyAnnouncementDetail(expected *model.AnnouncementStatus, res *api.GetAnnouncementDetailResponse, hres *http.Response) error {
	// Dirtyフラグが立っていない場合のみ、Unreadの検証を行う
	// 既読化RequestがTimeoutで中断された際、ベンチには既読が反映しないがwebapp側が既読化される可能性があるため。
	if err := AssertEqualAnnouncementDetail(expected, res, !expected.Dirty); err != nil {
		return fails.ErrorInvalidResponse(err, hres)
	}

	return nil
}

// お知らせ一覧の中身の検証
func verifyAnnouncementsList(expectedMap map[string]*model.AnnouncementStatus, res *api.GetAnnouncementsResponse, hres *http.Response, verifyUnread bool) error {
	// id の降順でソートされているか
	for i := 0; i < len(res.Announcements)-1; i++ {
		if res.Announcements[i].ID < res.Announcements[i+1].ID {
			return fails.ErrorInvalidResponse(errors.New("お知らせの順序が不正です"), hres)
		}
	}

	// リストの中身の検証
	for _, actual := range res.Announcements {
		expectStatus, ok := expectedMap[actual.ID]
		if !ok {
			// webappでは認識されているが、ベンチではまだ認識されていないお知らせ（お知らせ登録がリトライ中の場合発生）
			// load中には検証できないのでskip
			continue
		}

		// Dirtyフラグが立っていない場合のみ、Unreadの検証を行う
		// 既読化RequestがTimeoutで中断された際、ベンチには既読が反映しないがwebapp側が既読化される可能性があるため。
		if err := AssertEqualAnnouncementListContent(expectStatus, &actual, verifyUnread && !expectStatus.Dirty); err != nil {
			return fails.ErrorInvalidResponse(err, hres)
		}
	}

	// unread_count はload中には検証できない

	return nil
}

func verifyClasses(expected []*model.Class, res []*api.GetClassResponse, student *model.Student, hres *http.Response) error {
	if !AssertEqual("class_list length", len(expected), len(res)) {
		return fails.ErrorInvalidResponse(errors.New("講義数が期待する数と一致しません"), hres)
	}

	for i, expectedClass := range expected {
		actualClass := res[i]
		if err := AssertEqualClass(expectedClass, actualClass, student); err != nil {
			return fails.ErrorInvalidResponse(err, hres)
		}
	}

	return nil
}

func verifyAssignments(assignmentsData []byte, class *model.Class, mustVerify bool, hres *http.Response) error {
	if mustVerify || rand.Float64() < assignmentsVerifyRate {
		r, err := zip.NewReader(bytes.NewReader(assignmentsData), int64(len(assignmentsData)))
		if err != nil {
			return fails.ErrorInvalidResponse(errors.New("課題zipの展開に失敗しました"), hres)
		}

		downloadedAssignments := make(map[string]uint32)
		for _, f := range r.File {
			rc, err := f.Open()
			if err != nil {
				return fails.ErrorInvalidResponse(errors.New("課題zipのデータ読み込みに失敗しました"), hres)
			}
			assignmentData, err := ioutil.ReadAll(rc)
			rc.Close()
			if err != nil {
				return fails.ErrorInvalidResponse(errors.New("課題zipのデータ読み込みに失敗しました"), hres)
			}
			downloadedAssignments[f.Name] = crc32.ChecksumIEEE(assignmentData)
		}

		expectedSubmissions := class.Submissions()

		// mapのサイズが等しく、提出した課題がすべてダウンロードされた課題に含まれていれば、提出した課題とダウンロードされた課題は集合として等しい
		if !AssertEqual("assignment length", len(expectedSubmissions), len(downloadedAssignments)) {
			return fails.ErrorInvalidResponse(errors.New("課題zipに含まれるファイルの数が期待する値と一致しません"), hres)
		}

		for studentCode, expectedSubmission := range expectedSubmissions {
			expectedFileName := studentCode + "-" + expectedSubmission.Title
			actualChecksum, ok := downloadedAssignments[expectedFileName]
			if !ok {
				return fails.ErrorInvalidResponse(errors.New("提出課題が課題zipに含まれていないか、ファイル名が間違っています"), hres)
			}
			if !AssertEqual("assignment checksum", expectedSubmission.Checksum, actualChecksum) {
				return fails.ErrorInvalidResponse(errors.New("提出課題とダウンロードされた課題の内容が一致しません"), hres)
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
		return fails.ErrorStaticResource(fmt.Errorf("期待するリソースが読み込まれませんでした (%s)", expectPath))
	}

	if resource.Error != nil {
		var nerr net.Error
		if failure.As(resource.Error, &nerr) {
			if nerr.Timeout() || nerr.Temporary() {
				return nerr
			}
		}
		return fails.ErrorStaticResource(fmt.Errorf("リソースの取得に失敗しました (%s) %w", expectPath, resource.Error))
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

	if !AssertEqual("resource checksum", expected, actual) {
		return fails.ErrorStaticResource(fmt.Errorf("期待するチェックサムと一致しません (%s)", expectPath))
	}
	return nil
}
