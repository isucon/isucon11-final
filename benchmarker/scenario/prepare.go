package scenario

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/parallel"
	"github.com/isucon/isucandar/random/useragent"
	"github.com/isucon/isucandar/worker"

	"github.com/isucon/isucon11-final/benchmarker/api"
	"github.com/isucon/isucon11-final/benchmarker/fails"
	"github.com/isucon/isucon11-final/benchmarker/generate"
	"github.com/isucon/isucon11-final/benchmarker/model"
)

const (
	prepareTimeout           = 20
	SearchCourseCountPerPage = 20
	AnnouncementCountPerPage = 20
	prepareCourseCapacity    = 50
)

func (s *Scenario) Prepare(ctx context.Context, step *isucandar.BenchmarkStep) error {
	ContestantLogger.Printf("===> PREPARE")

	a, err := agent.NewAgent(
		agent.WithNoCache(),
		agent.WithNoCookie(),
		agent.WithTimeout(20*time.Second),
		agent.WithBaseURL(s.BaseURL.String()),
		agent.WithCloneTransport(agent.DefaultTransport),
	)
	if err != nil {
		return fails.ErrorCritical(err)
	}

	a.Name = "benchmarker-initializer"

	ContestantLogger.Printf("start Initialize")
	hres, res, err := InitializeAction(ctx, a)
	if err != nil {
		ContestantLogger.Printf("initializeが失敗しました")
		return fails.ErrorCritical(err)
	}
	err = verifyInitialize(res, hres)
	if err != nil {
		return fails.ErrorCritical(err)
	}
	s.language = res.Language

	if s.NoPrepare {
		return nil
	}

	// 検索の検証
	// 初期科目を対象に検索したいので最初に検証する
	err = s.prepareSearchCourse(ctx)
	if err != nil {
		return fails.ErrorCritical(err)
	}

	err = s.prepareNormal(ctx, step)
	if err != nil {
		return fails.ErrorCritical(err)
	}

	err = s.prepareAnnouncementsList(ctx, step)
	if err != nil {
		return fails.ErrorCritical(err)
	}

	err = s.prepareAbnormal(ctx)
	if err != nil {
		return fails.ErrorCritical(err)
	}

	_, _, err = InitializeAction(ctx, a)
	if err != nil {
		ContestantLogger.Printf("initializeが失敗しました")
		return fails.ErrorCritical(err)
	}

	AdminLogger.Printf("Language: %s", s.Language())

	step.Result().Score.Reset()
	s.Reset()
	return nil
}

func (s *Scenario) prepareNormal(ctx context.Context, step *isucandar.BenchmarkStep) error {
	const (
		prepareTeacherCount        = 2
		prepareStudentCount        = 2
		prepareCourseCount         = 20
		prepareClassCountPerCourse = 5
		prepareCourseCapacity      = 50
	)
	errors := step.Result().Errors
	hasErrors := func() bool {
		errors.Wait()
		return len(errors.All()) > 0
	}

	teachers := make([]*model.Teacher, 0, prepareTeacherCount)
	// TODO: ランダムなので同じ教師が入る可能性がある
	for i := 0; i < prepareTeacherCount; i++ {
		teachers = append(teachers, s.userPool.randomTeacher())
	}

	students := make([]*model.Student, 0, prepareStudentCount)
	for i := 0; i < prepareStudentCount; i++ {
		student, err := s.userPool.newStudent()
		if err != nil {
			return err
		}
		students = append(students, student)
	}

	courses := make([]*model.Course, 0, prepareCourseCount)
	mu := sync.Mutex{}
	// 教師のログインとコース登録をするワーカー
	w, err := worker.NewWorker(func(ctx context.Context, i int) {
		teacher := teachers[i%len(teachers)]
		isLoggedIn := teacher.LoginOnce(func(teacher *model.Teacher) {
			_, err := LoginAction(ctx, teacher.Agent, teacher.UserAccount)
			if err != nil {
				AdminLogger.Printf("teacherのログインに失敗しました")
				step.AddError(fails.ErrorCritical(err))
				return
			}
			teacher.IsLoggedIn = true
		})
		if !isLoggedIn {
			return
		}

		hres, getMeRes, err := GetMeAction(ctx, teacher.Agent)
		if err != nil {
			AdminLogger.Printf("teacherのユーザ情報取得に失敗しました")
			step.AddError(err)
			return
		}
		if err := verifyMe(teacher.UserAccount, &getMeRes, hres); err != nil {
			step.AddError(err)
			return
		}

		param := generate.CourseParam(i/6, i%6, teacher)
		_, res, err := AddCourseAction(ctx, teacher.Agent, param)
		if err != nil {
			step.AddError(err)
			return
		}
		course := model.NewCourse(param, res.ID, teacher, prepareCourseCapacity, model.NewCapacityCounter())
		mu.Lock()
		courses = append(courses, course)
		mu.Unlock()
	}, worker.WithLoopCount(prepareCourseCount))

	if err != nil {
		AdminLogger.Println("info: cannot start worker: %w", err)
	}

	w.Process(ctx)
	w.Wait()

	if hasErrors() {
		return fmt.Errorf("アプリケーション互換性チェックに失敗しました")
	}

	// 生徒のログインとコース登録
	w, err = worker.NewWorker(func(ctx context.Context, i int) {
		student := students[i]
		_, err := LoginAction(ctx, student.Agent, student.UserAccount)
		if err != nil {
			AdminLogger.Printf("studentのログインに失敗しました")
			step.AddError(fails.ErrorCritical(err))
			return
		}

		hres, getMeRes, err := GetMeAction(ctx, student.Agent)
		if err != nil {
			AdminLogger.Printf("studentのユーザ情報取得に失敗しました")
			step.AddError(err)
			return
		}
		if err := verifyMe(student.UserAccount, &getMeRes, hres); err != nil {
			step.AddError(err)
			return
		}

		_, _, err = TakeCoursesAction(ctx, student.Agent, courses)
		if err != nil {
			step.AddError(err)
			return
		}
		for _, course := range courses {
			student.AddCourse(course)
			course.AddStudent(student)
		}
	}, worker.WithLoopCount(prepareStudentCount))
	if err != nil {
		AdminLogger.Println("info: cannot start worker: %w", err)
	}
	w.Process(ctx)
	w.Wait()
	if hasErrors() {
		return fmt.Errorf("アプリケーション互換性チェックに失敗しました")
	}

	// コースのステータスの変更
	w, err = worker.NewWorker(func(ctx context.Context, i int) {
		course := courses[i]
		teacher := course.Teacher()
		_, err = SetCourseStatusInProgressAction(ctx, teacher.Agent, course.ID)
		if err != nil {
			step.AddError(err)
			return
		}
		course.SetStatusToInProgress()
	}, worker.WithLoopCount(prepareCourseCount))
	if err != nil {
		AdminLogger.Println("info: cannot start worker: %w", err)
	}
	w.Process(ctx)
	w.Wait()

	if hasErrors() {
		return fmt.Errorf("アプリケーション互換性チェックに失敗しました")
	}

	studentByCode := make(map[string]*model.Student)
	for _, student := range students {
		studentByCode[student.Code] = student
	}
	// クラス追加、おしらせ追加、（一部）お知らせ確認
	// 課題提出、ダウンロード、採点、成績確認
	// workerはコースごと
	checkAnnouncementDetailPart := rand.Intn(prepareClassCountPerCourse)
	for classPart := 0; classPart < prepareClassCountPerCourse; classPart++ {
		w, err = worker.NewWorker(func(ctx context.Context, i int) {
			course := courses[i]
			teacher := course.Teacher()
			// クラス追加
			classParam := generate.ClassParam(course, uint8(classPart+1))
			_, classRes, err := AddClassAction(ctx, teacher.Agent, course, classParam)
			if err != nil {
				step.AddError(err)
				return
			}
			class := model.NewClass(classRes.ClassID, classParam)
			course.AddClass(class)

			// お知らせ追加
			announcement := generate.Announcement(course, class)
			_, err = SendAnnouncementAction(ctx, teacher.Agent, announcement)
			if err != nil {
				step.AddError(err)
				return
			}
			course.BroadCastAnnouncement(announcement)

			courseStudents := course.Students()

			// 課題提出, ランダムでお知らせを読む
			// 生徒ごとのループ
			p := parallel.NewParallel(ctx, prepareStudentCount)
			for _, student := range courseStudents {
				student := student
				err := p.Do(func(ctx context.Context) {
					if classPart == checkAnnouncementDetailPart {
						hres, res, err := GetAnnouncementDetailAction(ctx, student.Agent, announcement.ID)
						if err != nil {
							step.AddError(err)
							return
						}
						expected := student.GetAnnouncement(announcement.ID)
						// announcement は course に追加されていて、student は course を履修しているので nil になることはないはず
						if expected == nil {
							panic("unreachable! announcementID" + announcement.ID)
						}
						err = AssertEqualAnnouncementDetail(expected, &res, hres, true)
						if err != nil {
							AdminLogger.Printf("extra announcements ->name: %v, title:  %v", res.CourseName, res.Title)
							step.AddError(err)
							return
						}
						student.ReadAnnouncement(announcement.ID)
					}
					submissionData, fileName := generate.SubmissionData(course, class, student.UserAccount)
					_, err := SubmitAssignmentAction(ctx, student.Agent, course.ID, class.ID, fileName, submissionData)
					if err != nil {
						step.AddError(err)
						return
					}
					submissionSummary := model.NewSubmission(fileName, submissionData)
					class.AddSubmission(student.Code, submissionSummary)
				})
				if err != nil {
					AdminLogger.Println("info: cannot start parallel: %w", err)
				}
			}
			p.Wait()

			// 課題ダウンロード
			hres, assignmentsData, err := DownloadSubmissionsAction(ctx, teacher.Agent, course.ID, class.ID)
			if err != nil {
				step.AddError(err)
				return
			}
			if err := verifyAssignments(assignmentsData, class, hres); err != nil {
				step.AddError(err)
				return
			}

			// 採点
			scores := make([]StudentScore, 0, len(students))
			for _, student := range students {
				sub := class.GetSubmissionByStudentCode(student.Code)
				if sub == nil {
					step.AddError(fails.ErrorCritical(fmt.Errorf("cannot find submission")))
					return
				}
				score := rand.Intn(101)
				sub.SetScore(score)
				scores = append(scores, StudentScore{
					score: score,
					code:  student.Code,
				})
			}

			_, err = PostGradeAction(ctx, teacher.Agent, course.ID, class.ID, scores)
			if err != nil {
				step.AddError(err)
				return
			}
		}, worker.WithLoopCount(prepareCourseCount))
		if err != nil {
			AdminLogger.Println("info: cannot start worker: %w", err)
		}
		w.Process(ctx)
		w.Wait()

		w, err = worker.NewWorker(func(ctx context.Context, i int) {
			student := students[i]
			expected := calculateGradeRes(student, studentByCode)
			hres, res, err := GetGradeAction(ctx, student.Agent)
			if err != nil {
				step.AddError(fails.ErrorCritical(err))
				return
			}

			err = AssertEqualGrade(&expected, &res, hres)
			if err != nil {
				step.AddError(err)
				return
			}
		}, worker.WithLoopCount(prepareStudentCount))
		if err != nil {
			AdminLogger.Println("info: cannot start worker: %w", err)
		}
		w.Process(ctx)
		w.Wait()
	}

	w, err = worker.NewWorker(func(ctx context.Context, i int) {
		course := courses[i]
		teacher := course.Teacher()
		_, err = SetCourseStatusClosedAction(ctx, teacher.Agent, course.ID)
		if err != nil {
			step.AddError(err)
			return
		}
		course.SetStatusToClosed()
	}, worker.WithLoopCount(prepareCourseCount))
	if err != nil {
		AdminLogger.Println("info: cannot start worker: %w", err)
	}
	w.Process(ctx)
	w.Wait()

	w, err = worker.NewWorker(func(ctx context.Context, i int) {
		student := students[i]
		expected := calculateGradeRes(student, studentByCode)
		hres, res, err := GetGradeAction(ctx, student.Agent)
		if err != nil {
			step.AddError(fails.ErrorCritical(err))
			return
		}

		err = AssertEqualGrade(&expected, &res, hres)
		if err != nil {
			step.AddError(err)
			return
		}
	}, worker.WithLoopCount(prepareStudentCount))
	if err != nil {
		AdminLogger.Println("info: cannot start worker: %w", err)
	}
	w.Process(ctx)
	w.Wait()

	if hasErrors() {
		return fmt.Errorf("アプリケーション互換性チェックに失敗しました")
	}

	// お知らせの検証
	w, err = worker.NewWorker(func(ctx context.Context, i int) {
		student := students[i]
		expected := student.Announcements()

		// id が新しい方が先頭に来るようにソート
		sort.Slice(expected, func(i, j int) bool {
			return expected[i].Announcement.ID > expected[j].Announcement.ID
		})
		expectedUnreadCount := 0
		for _, announcement := range expected {
			if announcement.Unread {
				expectedUnreadCount++
			}
		}
		_, err := prepareCheckAnnouncementsList(ctx, student.Agent, "", "", expected, expectedUnreadCount, student.Code)
		if err != nil {
			step.AddError(err)
			return
		}

	}, worker.WithLoopCount(prepareStudentCount))
	if err != nil {
		AdminLogger.Println("info: cannot start worker: %w", err)
	}
	w.Process(ctx)
	w.Wait()
	if hasErrors() {
		return fmt.Errorf("アプリケーション互換性チェックに失敗しました")
	}

	return nil
}

func (s *Scenario) prepareAnnouncementsList(ctx context.Context, step *isucandar.BenchmarkStep) error {
	// 4 回目のクラスのおしらせが追加された時点で CourseCount(5) * 4 = 20
	// となりおしらせが 20 個になるので、次のページの next がないかを検証できるはず
	const (
		prepareCheckAnnouncementListStudentCount        = 2
		prepareCheckAnnouncementListTeacherCount        = 2
		prepareCheckAnnouncementListCourseCount         = 5
		prepareCheckAnnouncementListClassCountPerCourse = 5
	)
	errors := step.Result().Errors
	hasErrors := func() bool {
		errors.Wait()
		return len(errors.All()) > 0
	}

	// 生徒の用意
	students := make([]*model.Student, prepareCheckAnnouncementListStudentCount)
	for i := 0; i < prepareCheckAnnouncementListStudentCount; i++ {
		student, err := s.getLoggedInStudent(ctx)
		if err != nil {
			return err
		}
		students[i] = student
	}

	if hasErrors() {
		return fmt.Errorf("アプリケーション互換性チェックに失敗しました")
	}

	// 教師の用意
	teachers := make([]*model.Teacher, prepareCheckAnnouncementListTeacherCount)
	for i := 0; i < prepareCheckAnnouncementListTeacherCount; i++ {
		teacher, err := s.getLoggedInTeacher(ctx)
		if err != nil {
			return err
		}
		teachers[i] = teacher
	}

	if hasErrors() {
		return fmt.Errorf("アプリケーション互換性チェックに失敗しました")
	}

	// コース登録
	var mu sync.Mutex
	courses := make([]*model.Course, 0, prepareCheckAnnouncementListCourseCount)
	w, err := worker.NewWorker(func(ctx context.Context, i int) {
		teacher := teachers[i%len(teachers)]
		param := generate.CourseParam(i/6, i%6, teacher)
		_, res, err := AddCourseAction(ctx, teacher.Agent, param)
		if err != nil {
			step.AddError(err)
			return
		}
		course := model.NewCourse(param, res.ID, teacher, prepareCourseCapacity, model.NewCapacityCounter())
		mu.Lock()
		courses = append(courses, course)
		mu.Unlock()
	}, worker.WithLoopCount(prepareCheckAnnouncementListCourseCount))
	if err != nil {
		AdminLogger.Println("info: cannot start worker: %w", err)
	}
	w.Process(ctx)
	w.Wait()

	if hasErrors() {
		return fmt.Errorf("アプリケーション互換性チェックに失敗しました")
	}

	// コース登録
	w, err = worker.NewWorker(func(ctx context.Context, i int) {
		student := students[i]
		_, _, err := TakeCoursesAction(ctx, student.Agent, courses)
		if err != nil {
			step.AddError(err)
			return
		}
		for _, course := range courses {
			student.AddCourse(course)
			course.AddStudent(student)
		}
	}, worker.WithLoopCount(prepareCheckAnnouncementListStudentCount))
	if err != nil {
		AdminLogger.Println("info: cannot start worker: %w", err)
	}
	w.Process(ctx)
	w.Wait()

	if hasErrors() {
		return fmt.Errorf("アプリケーション互換性チェックに失敗しました")
	}

	// コースのステータスを更新する
	for _, course := range courses {
		_, err = SetCourseStatusInProgressAction(ctx, course.Teacher().Agent, course.ID)
		if err != nil {
			return err
		}
		course.SetStatusToInProgress()
	}

	// クラス追加、おしらせ追加をする
	// そのたびにおしらせリストを確認する
	// 既読にはしない
	for classPart := 0; classPart < prepareCheckAnnouncementListClassCountPerCourse; classPart++ {
		for j := 0; j < prepareCheckAnnouncementListCourseCount; j++ {
			course := courses[j]

			// 最初のクラスが追加される前にも確認(おしらせが0のときに next がないことを保証する)
			if classPart == 0 {
				prepareCheckCourseAnnouncementList(ctx, step, course)
			}

			teacher := course.Teacher()
			// クラス追加
			classParam := generate.ClassParam(course, uint8(classPart+1))
			_, classRes, err := AddClassAction(ctx, teacher.Agent, course, classParam)
			if err != nil {
				step.AddError(err)
				return err
			}
			class := model.NewClass(classRes.ClassID, classParam)
			course.AddClass(class)

			// お知らせ追加
			announcement := generate.Announcement(course, class)
			_, err = SendAnnouncementAction(ctx, teacher.Agent, announcement)
			if err != nil {
				step.AddError(err)
				return err
			}
			course.BroadCastAnnouncement(announcement)

			// コースごとに、そのコースに登録している生徒ごとにおしらせリストを確認する
			prepareCheckCourseAnnouncementList(ctx, step, course)
		}
	}

	if hasErrors() {
		return fmt.Errorf("アプリケーション互換性チェックに失敗しました")
	}

	return nil
}

// コースに登録している生徒全員について、すべてのおしらせリストとコースで絞り込んだおしらせリストを検証する
func prepareCheckCourseAnnouncementList(ctx context.Context, step *isucandar.BenchmarkStep, course *model.Course) {
	courseStudents := course.Students()
	p := parallel.NewParallel(ctx, int32(len(courseStudents)))
	for _, student := range courseStudents {
		student := student
		err := p.Do(func(ctx context.Context) {
			expected := student.Announcements()

			// id が新しい方が先頭に来るようにソート
			sort.Slice(expected, func(i, j int) bool {
				return expected[i].Announcement.ID > expected[j].Announcement.ID
			})
			_, err := prepareCheckAnnouncementsList(ctx, student.Agent, "", "", expected, len(expected), student.Code)
			if err != nil {
				step.AddError(err)
				return
			}

			courseAnnouncementStatus := make([]*model.AnnouncementStatus, 0, 5)
			for _, status := range expected {
				if status.Announcement.CourseID == course.ID {
					courseAnnouncementStatus = append(courseAnnouncementStatus, status)
				}
			}

			_, err = prepareCheckAnnouncementsList(ctx, student.Agent, "", course.ID, courseAnnouncementStatus, len(expected), student.Code)
			if err != nil {
				step.AddError(err)
				return
			}
		})
		if err != nil {
			AdminLogger.Println("info: cannot start parallel: %w", err)
		}
	}
	p.Wait()
}

func prepareCheckAnnouncementsList(ctx context.Context, a *agent.Agent, path, courseID string, expected []*model.AnnouncementStatus, expectedUnreadCount int, userCode string) (prev string, err error) {
	errInvalidNext := fails.ErrorCritical(fmt.Errorf("link header の next によってページングできる回数が不正です"))

	hres, res, err := GetAnnouncementListAction(ctx, a, path, courseID)
	if err != nil {
		return "", err
	}
	prev, next, err := parseLinkHeader(hres)
	if err != nil {
		return "", err
	}

	if (len(expected) <= AnnouncementCountPerPage && next != "") || (len(expected) > AnnouncementCountPerPage && next == "") {
		return "", errInvalidNext
	}
	// 次のページが存在しない
	if next == "" {
		err = prepareCheckAnnouncementContent(expected, res, expectedUnreadCount, userCode, hres)
		if err != nil {
			return "", err
		}
		return prev, nil
	}

	err = prepareCheckAnnouncementContent(expected[:AnnouncementCountPerPage], res, expectedUnreadCount, userCode, hres)
	if err != nil {
		return "", err
	}

	// この_prevはpathと同じところを指すはず
	// _prevとpathが同じ文字列であるとは限らない（pathが"" で_prevが/api/announcements?page=1とか）
	_prev, err := prepareCheckAnnouncementsList(ctx, a, next, courseID, expected[AnnouncementCountPerPage:], expectedUnreadCount, userCode)
	if err != nil {
		return "", err
	}

	hres, res, err = GetAnnouncementListAction(ctx, a, _prev, courseID)
	if err != nil {
		return "", err
	}

	err = prepareCheckAnnouncementContent(expected[:AnnouncementCountPerPage], res, expectedUnreadCount, userCode, hres)
	if err != nil {
		return "", err
	}

	return prev, nil
}

func prepareCheckAnnouncementContent(expected []*model.AnnouncementStatus, actual api.GetAnnouncementsResponse, expectedUnreadCount int, userCode string, hres *http.Response) error {
	errWithUserCode := func(err error, hres *http.Response) error {
		return fails.ErrorCritical(fails.ErrorInvalidResponse(fmt.Errorf("%w (検証対象学生の学内コード: %s)", err, userCode), hres))
	}
	errWithUserCodeAndAnnouncementID := func(err error, announcementID string, hres *http.Response) error {
		return fails.ErrorCritical(fails.ErrorInvalidResponse(fmt.Errorf("%w (検証対象学生の学内コード: %s, お知らせID: %s)", err, userCode, announcementID), hres))
	}

	reasonNotSorted := errors.New("お知らせの順序が不正です")
	reasonNotMatch := errors.New("お知らせの内容が不正です")
	reasonNoCount := errors.New("お知らせの数が期待したものと一致しませんでした")
	reasonNoMatchUnreadCount := errors.New("お知らせの unread_count が期待したものと一致しませんでした")

	if actual.UnreadCount != expectedUnreadCount {
		return errWithUserCode(reasonNoMatchUnreadCount, hres)
	}

	if len(expected) != len(actual.Announcements) {
		return errWithUserCode(reasonNoCount, hres)
	}

	// 順序の検証
	for i := 0; i < len(actual.Announcements)-1; i++ {
		if actual.Announcements[i].ID < actual.Announcements[i+1].ID {
			return errWithUserCode(reasonNotSorted, hres)
		}
	}

	for i := 0; i < len(actual.Announcements); i++ {
		if err := AssertEqualAnnouncementListContent(expected[i], &actual.Announcements[i], hres, true); err != nil {
			AdminLogger.Printf("extra announcements ->name: %v, title:  %v", actual.Announcements[i].CourseName, actual.Announcements[i].Title)
			return errWithUserCodeAndAnnouncementID(reasonNotMatch, actual.Announcements[i].ID, hres)
		}
	}

	return nil
}

func (s *Scenario) prepareSearchCourse(ctx context.Context) error {
	// 検証で使用する学生ユーザ
	student, err := s.getLoggedInStudent(ctx)
	if err != nil {
		return err
	}

	courses := s.initCourses
	// code の昇順にソート
	sort.Slice(courses, func(i, j int) bool {
		return courses[i].Code < courses[j].Code
	})

	// 全検索の検証
	param := model.NewCourseParam()
	expected := searchCourseLocal(courses, param)
	if err := prepareCheckSearchCourse(ctx, student.Agent, param, expected); err != nil {
		return err
	}

	// 単体条件クエリの検証
	param = model.NewCourseParam()
	param.Type = "major-subjects"
	expected = searchCourseLocal(courses, param)
	if err := prepareCheckSearchCourse(ctx, student.Agent, param, expected); err != nil {
		return err
	}

	param = model.NewCourseParam()
	param.Credit = 1
	expected = searchCourseLocal(courses, param)
	if err := prepareCheckSearchCourse(ctx, student.Agent, param, expected); err != nil {
		return err
	}

	param = model.NewCourseParam()
	param.Teacher = courses[rand.Intn(len(courses))].Teacher().Name
	expected = searchCourseLocal(courses, param)
	if err := prepareCheckSearchCourse(ctx, student.Agent, param, expected); err != nil {
		return err
	}

	param = model.NewCourseParam()
	param.Period = 0
	expected = searchCourseLocal(courses, param)
	if err := prepareCheckSearchCourse(ctx, student.Agent, param, expected); err != nil {
		return err
	}

	param = model.NewCourseParam()
	param.DayOfWeek = 0
	expected = searchCourseLocal(courses, param)
	if err := prepareCheckSearchCourse(ctx, student.Agent, param, expected); err != nil {
		return err
	}

	param = model.NewCourseParam()
	param.Keywords = strings.Split(courses[rand.Intn(len(courses))].Keywords, " ")[:1]
	expected = searchCourseLocal(courses, param)
	if err := prepareCheckSearchCourse(ctx, student.Agent, param, expected); err != nil {
		return err
	}

	//  キーワード検索の簡単化の防止 https://github.com/isucon/isucon11-final/issues/691
	param = model.NewCourseParam()
	param.Keywords = []string{"SpeedUP"}
	expected = searchCourseLocal(courses, param)
	if err := prepareCheckSearchCourse(ctx, student.Agent, param, expected); err != nil {
		return err
	}

	param = model.NewCourseParam()
	param.Status = "closed"
	expected = searchCourseLocal(courses, param)
	if err := prepareCheckSearchCourse(ctx, student.Agent, param, expected); err != nil {
		return err
	}

	// 複合条件クエリの検証
	target := courses[rand.Intn(len(courses))]
	param = model.NewCourseParam()
	param.Type = target.Type
	param.Credit = target.Credit
	param.Teacher = target.Teacher().Name
	param.Period = target.Period
	param.DayOfWeek = target.DayOfWeek
	param.Keywords = strings.Split(target.Keywords, " ")[:1]
	param.Status = string(target.Status())
	expected = searchCourseLocal(courses, param)
	if err := prepareCheckSearchCourse(ctx, student.Agent, param, expected); err != nil {
		return err
	}

	return nil
}

func prepareCheckSearchCourse(ctx context.Context, a *agent.Agent, param *model.SearchCourseParam, expected []*model.Course) error {
	errWithParamInfo := func(err error, hres *http.Response) error {
		return fails.ErrorInvalidResponse(fmt.Errorf("%w (検索条件: %s)", err, param.GetParamString()), hres)
	}

	reasonEmpty := errors.New("科目検索の最初以外のページで空の検索結果が返却されました")
	reasonDuplicated := errors.New("科目検索結果に重複した id の科目が存在します")
	reasonLack := errors.New("科目検索で条件にヒットするはずの科目が見つかりませんでした")
	reasonExcess := errors.New("科目検索で条件にヒットしない科目が見つかりました")
	reasonInvalidContent := errors.New("科目検索結果に含まれる科目の内容が不正です")
	reasonNotSorted := errors.New("科目検索結果の順序が code の昇順になっていません")
	reasonNotMatchCountPerPage := errors.New("科目検索のページごとの件数が不正です")
	reasonExistPrevFirstPage := errors.New("科目検索の最初のページの link header に prev が存在しました")
	reasonNotExistPrevOtherThanFirstPage := errors.New("科目検索の最初以外のページの link header に prev が存在しませんでした")
	reasonInvalidPrev := errors.New("科目検索の link header の prev で前のページに正しく戻ることができませんでした")

	var hresSample *http.Response
	var path string
	actual := make([]*api.GetCourseDetailResponse, 0)
	actualByID := make(map[string]*api.GetCourseDetailResponse)
	actualResCountList := make([]int, 0)
	prevList := make([]string, 0)
	for {
		hres, res, err := SearchCourseAction(ctx, a, param, path)
		if err != nil {
			return errWithParamInfo(err, hres)
		}

		// 空リストを返され続けると無限ループするので最初のページ以外で空リストが返ってきたらエラーにする
		if path != "" && len(res) == 0 {
			return errWithParamInfo(reasonEmpty, hres)
		}

		for _, course := range res {
			_, exists := actualByID[course.ID]
			// IDが重複していたらエラーにする
			if exists {
				return errWithParamInfo(reasonDuplicated, hres)
			}
			actualByID[course.ID] = course
			actual = append(actual, course)
		}
		actualResCountList = append(actualResCountList, len(res))

		// 期待する件数よりも多かったら少なくとも1件ヒットすべきでない科目がヒットしている
		if len(actual) > len(expected) {
			return errWithParamInfo(reasonExcess, hres)
		}

		hresSample = hres
		prev, next, err := parseLinkHeader(hres)
		if err != nil {
			return errWithParamInfo(err, hres)
		}

		prevList = append(prevList, prev)
		path = next

		if path == "" {
			break
		}
	}

	// 順序は無視して期待するコースがすべて検索結果に含まれていることを検証
	// len(actual) <= len(expected) であり、actual には重複がないことが保証されているので expected の科目がすべて actual に含まれていれば両者は順序を除いて等しい
	for _, expectCourse := range expected {
		actualCourse, exists := actualByID[expectCourse.ID]
		// 同じIDの科目がなかったらその科目は見つからなかった扱いにする
		if !exists {
			return errWithParamInfo(reasonLack, hresSample)
		}
		// 同じIDでも内容が違っていたら科目自体は見つかったが内容が不正という扱いにする
		if err := AssertEqualCourse(expectCourse, actualCourse, hresSample, true); err != nil {
			return errWithParamInfo(reasonInvalidContent, hresSample)
		}
	}

	// 順序の検証
	for i := 0; i < len(actual)-1; i++ {
		if actual[i].Code > actual[i+1].Code {
			return errWithParamInfo(reasonNotSorted, hresSample)
		}
	}

	// 各ページの件数の検証
	expectResCountList := make([]int, 0)
	rest := len(expected)
	for {
		if rest <= SearchCourseCountPerPage {
			expectResCountList = append(expectResCountList, rest)
			break
		} else {
			expectResCountList = append(expectResCountList, SearchCourseCountPerPage)
			rest -= SearchCourseCountPerPage
		}
	}
	if !AssertEqual("search count per page", expectResCountList, actualResCountList) {
		return errWithParamInfo(reasonNotMatchCountPerPage, hresSample)
	}

	// prev の存在検証
	for i := 0; i < len(prevList); i++ {
		if i == 0 && prevList[i] != "" {
			return errWithParamInfo(reasonExistPrevFirstPage, hresSample)
		}
		if i > 0 && prevList[i] == "" {
			return errWithParamInfo(reasonNotExistPrevOtherThanFirstPage, hresSample)
		}
	}

	// prev で前のページに正しく戻れることの検証（最終ページから戻るように見ていく）
	for page := len(prevList) - 1; page >= 1; page-- {
		hres, res, err := SearchCourseAction(ctx, a, param, prevList[page])
		if err != nil {
			return errWithParamInfo(err, hres)
		}

		// prev でのアクセスなので1ページあたりの最大件数が取れるはず
		if len(res) != SearchCourseCountPerPage {
			return errWithParamInfo(reasonInvalidPrev, hres)
		}

		// リストの内容の検証
		for i, course := range res {
			if err := AssertEqualCourse(expected[SearchCourseCountPerPage*(page-1)+i], course, hres, true); err != nil {
				return errWithParamInfo(reasonInvalidPrev, hres)
			}
		}
	}

	return nil
}

func searchCourseLocal(courses []*model.Course, param *model.SearchCourseParam) []*model.Course {
	matchCourses := make([]*model.Course, 0)

	for _, course := range courses {
		if (param.Type == "" || course.Type == param.Type) &&
			(param.Credit == 0 || course.Credit == param.Credit) &&
			(param.Teacher == "" || course.Teacher().Name == param.Teacher) &&
			(param.Period == -1 || course.Period == param.Period) &&
			(param.DayOfWeek == -1 || course.DayOfWeek == param.DayOfWeek) &&
			(containsAll(course.Name, param.Keywords) || containsAll(course.Keywords, param.Keywords)) &&
			(param.Status == "" || string(course.Status()) == param.Status) {
			matchCourses = append(matchCourses, course)
		}
	}

	return matchCourses
}

func (s *Scenario) prepareAbnormal(ctx context.Context) error {
	// TODO: 並列化

	// 認証チェック
	if err := s.prepareCheckAuthenticationAbnormal(ctx); err != nil {
		return err
	}

	// 講師用APIの認可チェック
	if err := s.prepareCheckAdminAuthorizationAbnormal(ctx); err != nil {
		return err
	}

	// POST /login
	if err := s.prepareCheckLoginAbnormal(ctx); err != nil {
		return err
	}

	// PUT /api/users/me/courses
	if err := s.prepareCheckRegisterCoursesAbnormal(ctx); err != nil {
		return err
	}

	// GET /api/courses/:courseID
	if err := s.prepareCheckGetCourseDetailAbnormal(ctx); err != nil {
		return err
	}

	// POST /api/courses
	if err := s.prepareCheckAddCourseAbnormal(ctx); err != nil {
		return err
	}

	// PUT /api/courses/:courseID/status
	if err := s.prepareCheckSetCourseStatusAbnormal(ctx); err != nil {
		return err
	}

	// GET /api/courses/:courseID/classes
	if err := s.prepareCheckGetClassesAbnormal(ctx); err != nil {
		return err
	}

	// POST /api/courses/:courseID/classes
	if err := s.prepareCheckAddClassAbnormal(ctx); err != nil {
		return err
	}

	// POST /api/courses/:courseID/classes/:classID/assignments
	if err := s.prepareCheckSubmitAssignmentAbnormal(ctx); err != nil {
		return err
	}

	// PUT /api/courses/:courseID/classes/:classID/assignments/scores
	if err := s.prepareCheckPostGradeAbnormal(ctx); err != nil {
		return err
	}

	// GET /api/courses/:courseID/classes/:classID/assignments/export
	if err := s.prepareCheckDownloadSubmissionsAbnormal(ctx); err != nil {
		return err
	}

	// POST /api/announcements
	if err := s.prepareCheckSendAnnouncementAbnormal(ctx); err != nil {
		return err
	}

	// GET /api/announcements/:announcementID
	if err := s.prepareCheckGetAnnouncementDetailAbnormal(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) prepareCheckAuthenticationAbnormal(ctx context.Context) error {
	errAuthentication := func(hres *http.Response) error {
		return fails.ErrorInvalidResponse(errors.New("未ログイン状態で認証が必要なAPIへのアクセスが成功しました"), hres)
	}
	checkAuthentication := func(hres *http.Response, err error) error {
		// リクエストが成功したらwebappの不具合
		if err == nil {
			return errAuthentication(hres)
		}

		// ステータスコードのチェック
		if err := verifyStatusCode(hres, []int{http.StatusUnauthorized}); err != nil {
			return err
		}

		return nil
	}

	// ======== 検証用データの準備 ========

	// 未ログインのagent
	agent, _ := agent.NewAgent(
		agent.WithUserAgent(useragent.UserAgent()),
		agent.WithBaseURL(s.BaseURL.String()),
		agent.WithCloneTransport(agent.DefaultTransport),
	)

	// 検証で使用する学生ユーザ
	student, err := s.getLoggedInStudent(ctx)
	if err != nil {
		return err
	}

	// 検証で使用する講師ユーザ
	teacher, err := s.getLoggedInTeacher(ctx)
	if err != nil {
		return err
	}

	// 適当な科目
	courseParam := generate.CourseParam(0, 0, teacher)
	_, addCourseRes, err := AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	course := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity, model.NewCapacityCounter())

	// 科目のステータス更新
	_, err = SetCourseStatusInProgressAction(ctx, teacher.Agent, course.ID)
	if err != nil {
		return err
	}
	course.SetStatusToInProgress()

	// 課題提出が締め切られた講義
	classParam := generate.ClassParam(course, 1)
	_, addClassRes, err := AddClassAction(ctx, teacher.Agent, course, classParam)
	if err != nil {
		return err
	}
	submissionClosedClass := model.NewClass(addClassRes.ClassID, classParam)
	_, _, err = DownloadSubmissionsAction(ctx, teacher.Agent, course.ID, submissionClosedClass.ID)
	if err != nil {
		return err
	}

	// 課題提出が締め切られていない講義
	classParam = generate.ClassParam(course, 2)
	_, addClassRes, err = AddClassAction(ctx, teacher.Agent, course, classParam)
	if err != nil {
		return err
	}
	submissionNotClosedClass := model.NewClass(addClassRes.ClassID, classParam)

	// course に紐づくお知らせ
	announcement1 := generate.Announcement(course, submissionNotClosedClass)
	_, err = SendAnnouncementAction(ctx, teacher.Agent, announcement1)
	if err != nil {
		return err
	}

	// ======== 検証 ========

	hres, _, err := GetMeAction(ctx, agent)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	hres, _, err = GetRegisteredCoursesAction(ctx, agent)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	hres, _, err = TakeCoursesAction(ctx, agent, []*model.Course{course})
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	hres, _, err = GetGradeAction(ctx, agent)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	param := generate.SearchCourseParam()
	hres, _, err = SearchCourseAction(ctx, agent, param, "")
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	hres, _, err = GetCourseDetailAction(ctx, agent, course.ID)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	courseParam = generate.CourseParam(0, 1, teacher)
	hres, _, err = AddCourseAction(ctx, agent, courseParam)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	hres, err = SetCourseStatusInProgressAction(ctx, agent, course.ID)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	hres, _, err = GetClassesAction(ctx, agent, course.ID)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	classParam = generate.ClassParam(course, 3)
	hres, _, err = AddClassAction(ctx, agent, course, classParam)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	submissionData, fileName := generate.SubmissionData(course, submissionNotClosedClass, student.UserAccount)
	hres, err = SubmitAssignmentAction(ctx, agent, course.ID, submissionNotClosedClass.ID, fileName, submissionData)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	scores := []StudentScore{
		{
			score: 90,
			code:  student.Code,
		},
	}
	hres, err = PostGradeAction(ctx, agent, course.ID, submissionClosedClass.ID, scores)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	hres, _, err = DownloadSubmissionsAction(ctx, agent, course.ID, submissionNotClosedClass.ID)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	hres, _, err = GetAnnouncementListAction(ctx, agent, "", "")
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	announcement2 := generate.Announcement(course, submissionNotClosedClass)
	hres, err = SendAnnouncementAction(ctx, agent, announcement2)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	hres, _, err = GetAnnouncementDetailAction(ctx, agent, announcement1.ID)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) prepareCheckAdminAuthorizationAbnormal(ctx context.Context) error {
	errAuthorization := func(hres *http.Response) error {
		return fails.ErrorInvalidResponse(errors.New("学生ユーザで講師用APIへのアクセスが成功しました"), hres)
	}
	checkAuthorization := func(hres *http.Response, err error) error {
		// リクエストが成功したらwebappの不具合
		if err == nil {
			return errAuthorization(hres)
		}

		// ステータスコードのチェック
		if err := verifyStatusCode(hres, []int{http.StatusForbidden}); err != nil {
			return err
		}

		return nil
	}

	// ======== 検証用データの準備 ========

	// 検証で使用する学生ユーザ
	student, err := s.getLoggedInStudent(ctx)
	if err != nil {
		return err
	}

	// 検証で使用する講師ユーザ
	teacher, err := s.getLoggedInTeacher(ctx)
	if err != nil {
		return err
	}

	// 適当な科目
	courseParam := generate.CourseParam(0, 0, teacher)
	_, addCourseRes, err := AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	course := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity, model.NewCapacityCounter())
	_, err = SetCourseStatusInProgressAction(ctx, teacher.Agent, course.ID)
	if err != nil {
		return err
	}
	course.SetStatusToInProgress()

	// 課題提出が締め切られた講義
	classParam := generate.ClassParam(course, 1)
	_, addClassRes, err := AddClassAction(ctx, teacher.Agent, course, classParam)
	if err != nil {
		return err
	}
	submissionClosedClass := model.NewClass(addClassRes.ClassID, classParam)
	_, _, err = DownloadSubmissionsAction(ctx, teacher.Agent, course.ID, submissionClosedClass.ID)
	if err != nil {
		return err
	}

	// 課題提出が締め切られていない講義
	classParam = generate.ClassParam(course, 2)
	_, addClassRes, err = AddClassAction(ctx, teacher.Agent, course, classParam)
	if err != nil {
		return err
	}
	submissionNotClosedClass := model.NewClass(addClassRes.ClassID, classParam)

	// ======== 検証 ========

	courseParam = generate.CourseParam(0, 1, teacher)
	hres, _, err := AddCourseAction(ctx, student.Agent, courseParam)
	if err := checkAuthorization(hres, err); err != nil {
		return err
	}

	hres, err = SetCourseStatusInProgressAction(ctx, student.Agent, course.ID)
	if err := checkAuthorization(hres, err); err != nil {
		return err
	}

	classParam = generate.ClassParam(course, 3)
	hres, _, err = AddClassAction(ctx, student.Agent, course, classParam)
	if err := checkAuthorization(hres, err); err != nil {
		return err
	}

	scores := []StudentScore{
		{
			score: 90,
			code:  student.Code,
		},
	}
	hres, err = PostGradeAction(ctx, student.Agent, course.ID, submissionClosedClass.ID, scores)
	if err := checkAuthorization(hres, err); err != nil {
		return err
	}

	hres, _, err = DownloadSubmissionsAction(ctx, student.Agent, course.ID, submissionNotClosedClass.ID)
	if err := checkAuthorization(hres, err); err != nil {
		return err
	}

	announcement := generate.Announcement(course, submissionNotClosedClass)
	hres, err = SendAnnouncementAction(ctx, student.Agent, announcement)
	if err := checkAuthorization(hres, err); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) prepareCheckLoginAbnormal(ctx context.Context) error {
	errInvalidLogin := func(hres *http.Response) error {
		return fails.ErrorInvalidResponse(errors.New("間違った認証情報でのログインに成功しました"), hres)
	}
	errRelogin := func(hres *http.Response) error {
		return fails.ErrorInvalidResponse(errors.New("ログイン状態での再ログインに成功しました"), hres)
	}

	// ======== 検証用データの準備 ========

	// 検証で使用する学生ユーザ（未ログイン状態）
	student, err := s.userPool.newStudent()
	if err != nil {
		panic("unreachable! studentPool is empty")
	}

	// ======== 検証 ========

	// 存在しないユーザでのログイン
	hres, err := LoginAction(ctx, student.Agent, &model.UserAccount{
		Code:        "X12345",
		RawPassword: "password",
		IsAdmin:     false,
	})
	if err == nil {
		return errInvalidLogin(hres)
	}
	if err := verifyStatusCode(hres, []int{http.StatusUnauthorized}); err != nil {
		return err
	}

	// 間違ったパスワードでのログイン
	hres, err = LoginAction(ctx, student.Agent, &model.UserAccount{
		Code:        student.Code,
		RawPassword: student.RawPassword + "abc",
		IsAdmin:     false,
	})
	if err == nil {
		return errInvalidLogin(hres)
	}
	if err := verifyStatusCode(hres, []int{http.StatusUnauthorized}); err != nil {
		return err
	}

	// 再ログインチェックのため一度ちゃんとログインする
	_, err = LoginAction(ctx, student.Agent, student.UserAccount)
	if err != nil {
		return err
	}

	// 再ログイン
	hres, err = LoginAction(ctx, student.Agent, student.UserAccount)
	if err == nil {
		return errRelogin(hres)
	}
	if err := verifyStatusCode(hres, []int{http.StatusBadRequest}); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) prepareCheckRegisterCoursesAbnormal(ctx context.Context) error {
	errInvalidRegistration := func(hres *http.Response) error {
		return fails.ErrorInvalidResponse(errors.New("履修登録できないはずの科目の履修に成功しました"), hres)
	}
	errInvalidErrorResponse := func(hres *http.Response) error {
		return fails.ErrorInvalidResponse(errors.New("履修登録失敗時のレスポンスが期待する内容と一致しません"), hres)
	}

	// ======== 検証用データの準備 ========

	// 検証で使用する学生ユーザ
	student, err := s.getLoggedInStudent(ctx)
	if err != nil {
		return err
	}

	// 検証で使用する講師ユーザ
	teacher, err := s.getLoggedInTeacher(ctx)
	if err != nil {
		return err
	}

	// ステータスが registration の科目
	courseParam := generate.CourseParam(0, 0, teacher)
	_, addCourseRes, err := AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	registrationCourse := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity, model.NewCapacityCounter())

	// ステータスが in-progress の科目
	courseParam = generate.CourseParam(0, 1, teacher)
	_, addCourseRes, err = AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	inProgressCourse := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity, model.NewCapacityCounter())
	_, err = SetCourseStatusInProgressAction(ctx, teacher.Agent, inProgressCourse.ID)
	if err != nil {
		return err
	}
	inProgressCourse.SetStatusToInProgress()

	// ステータスが closed の科目
	courseParam = generate.CourseParam(0, 2, teacher)
	_, addCourseRes, err = AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	closedCourse := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity, model.NewCapacityCounter())
	_, err = SetCourseStatusClosedAction(ctx, teacher.Agent, closedCourse.ID)
	if err != nil {
		return err
	}
	closedCourse.SetStatusToClosed()

	// student が履修登録済みの科目
	courseParam = generate.CourseParam(1, 0, teacher)
	_, addCourseRes, err = AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	alreadyRegisteredCourse := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity, model.NewCapacityCounter())
	_, _, err = TakeCoursesAction(ctx, student.Agent, []*model.Course{alreadyRegisteredCourse})
	if err != nil {
		return err
	}

	// alreadyRegisteredCourse と時間割がコンフリクトする科目
	courseParam = generate.CourseParam(1, 0, teacher)
	_, addCourseRes, err = AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	conflictedCourse1 := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity, model.NewCapacityCounter())

	// 時間割がコンフリクトする2つの科目
	courseParam = generate.CourseParam(1, 1, teacher)
	_, addCourseRes, err = AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	conflictedCourse2 := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity, model.NewCapacityCounter())

	courseParam = generate.CourseParam(1, 1, teacher)
	_, addCourseRes, err = AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	conflictedCourse3 := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity, model.NewCapacityCounter())

	// 存在しない科目
	courseParam = generate.CourseParam(2, 0, teacher)
	unknownCourse := model.NewCourse(courseParam, generate.GenULID(), teacher, prepareCourseCapacity, model.NewCapacityCounter())

	// ======== 検証 ========

	courses := []*model.Course{
		registrationCourse,
		inProgressCourse,
		closedCourse,
		alreadyRegisteredCourse,
		conflictedCourse1,
		conflictedCourse2,
		conflictedCourse3,
		unknownCourse,
	}
	hres, eres, err := TakeCoursesAction(ctx, student.Agent, courses)
	if err == nil {
		return errInvalidRegistration(hres)
	}
	err = verifyStatusCode(hres, []int{http.StatusBadRequest})
	if err != nil {
		return err
	}

	// 順序を無視して一致するならtrue
	isSameIgnoringOrder := func(s1, s2 []string) bool {
		if len(s1) != len(s2) {
			return false
		}

		sort.Slice(s1, func(i, j int) bool { return s1[i] < s1[j] })
		sort.Slice(s2, func(i, j int) bool { return s2[i] < s2[j] })

		for i := 0; i < len(s1); i++ {
			if s1[i] != s2[i] {
				return false
			}
		}

		return true
	}

	if !isSameIgnoringOrder(eres.CourseNotFound, []string{unknownCourse.ID}) ||
		!isSameIgnoringOrder(eres.NotRegistrableStatus, []string{inProgressCourse.ID, closedCourse.ID}) ||
		!isSameIgnoringOrder(eres.ScheduleConflict, []string{conflictedCourse1.ID, conflictedCourse2.ID, conflictedCourse3.ID}) {
		return errInvalidErrorResponse(hres)
	}

	return nil
}

func (s *Scenario) prepareCheckGetCourseDetailAbnormal(ctx context.Context) error {
	errGetUnknownCourseDetail := func(hres *http.Response) error {
		return fails.ErrorInvalidResponse(errors.New("存在しない科目の詳細取得に成功しました"), hres)
	}

	// ======== 検証用データの準備 ========

	// 検証で使用する学生ユーザ
	student, err := s.getLoggedInStudent(ctx)
	if err != nil {
		return err
	}

	// ======== 検証 ========

	// 存在しない科目IDでの科目詳細取得
	hres, _, err := GetCourseDetailAction(ctx, student.Agent, generate.GenULID())
	if err == nil {
		return errGetUnknownCourseDetail(hres)
	}
	if err := verifyStatusCode(hres, []int{http.StatusNotFound}); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) prepareCheckAddCourseAbnormal(ctx context.Context) error {
	errAddInvalidCourse := func(hres *http.Response) error {
		return fails.ErrorInvalidResponse(errors.New("不正な科目の追加に成功しました"), hres)
	}
	errAddConflictedCourse := func(hres *http.Response) error {
		return fails.ErrorInvalidResponse(errors.New("コードが重複した科目の追加に成功しました"), hres)
	}

	// ======== 検証用データの準備 ========

	// 検証で使用する講師ユーザ
	teacher, err := s.getLoggedInTeacher(ctx)
	if err != nil {
		return err
	}

	// 適当な科目
	courseParam := generate.CourseParam(0, 0, teacher)
	_, addCourseRes, err := AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	course := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity, model.NewCapacityCounter())

	// ======== 検証 ========

	// Type が不正な科目の追加
	courseParam = generate.CourseParam(0, 1, teacher)
	courseParam.Type = "invalid-type"
	hres, _, err := AddCourseAction(ctx, teacher.Agent, courseParam)
	if err == nil {
		return errAddInvalidCourse(hres)
	}
	if err := verifyStatusCode(hres, []int{http.StatusBadRequest}); err != nil {
		return err
	}

	// DayOfWeek が不正な科目の追加
	// courseParam.DayOfWeek を0-6以外にしておくとAction側の処理で空文字列（不正な入力）として送信される
	courseParam = generate.CourseParam(-1, 0, teacher)
	hres, _, err = AddCourseAction(ctx, teacher.Agent, courseParam)
	if err == nil {
		return errAddInvalidCourse(hres)
	}
	if err := verifyStatusCode(hres, []int{http.StatusBadRequest}); err != nil {
		return err
	}

	// コンフリクトする科目の追加
	// Code を同じにし、少なくとも Period を変えることで course とコンフリクトさせる
	courseParam = generate.CourseParam(0, 2, teacher)
	courseParam.Code = course.Code
	hres, _, err = AddCourseAction(ctx, teacher.Agent, courseParam)
	if err == nil {
		return errAddConflictedCourse(hres)
	}
	if err := verifyStatusCode(hres, []int{http.StatusConflict}); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) prepareCheckSetCourseStatusAbnormal(ctx context.Context) error {
	errSetStatusForUnknownCourse := func(hres *http.Response) error {
		return fails.ErrorInvalidResponse(errors.New("存在しない科目のステータス変更に成功しました"), hres)
	}

	// ======== 検証用データの準備 ========

	// 検証で使用する講師ユーザ
	teacher, err := s.getLoggedInTeacher(ctx)
	if err != nil {
		return err
	}

	// ======== 検証 ========

	// 存在しない科目IDでの科目ステータス変更
	hres, err := SetCourseStatusInProgressAction(ctx, teacher.Agent, generate.GenULID())
	if err == nil {
		return errSetStatusForUnknownCourse(hres)
	}
	if err := verifyStatusCode(hres, []int{http.StatusNotFound}); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) prepareCheckGetClassesAbnormal(ctx context.Context) error {
	errGetClassesForUnknownCourse := func(hres *http.Response) error {
		return fails.ErrorInvalidResponse(errors.New("存在しない科目の講義一覧取得に成功しました"), hres)
	}

	// ======== 検証用データの準備 ========

	// 検証で使用する学生ユーザ
	student, err := s.getLoggedInStudent(ctx)
	if err != nil {
		return err
	}

	// ======== 検証 ========

	// 存在しない科目IDでの講義一覧取得
	hres, _, err := GetClassesAction(ctx, student.Agent, generate.GenULID())
	if err == nil {
		return errGetClassesForUnknownCourse(hres)
	}
	if err := verifyStatusCode(hres, []int{http.StatusNotFound}); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) prepareCheckAddClassAbnormal(ctx context.Context) error {
	errAddClassInvalidStatus := func(hres *http.Response) error {
		return fails.ErrorInvalidResponse(errors.New("in-progress でない科目に講義の追加が成功しました"), hres)
	}
	errAddClassForUnknownCourse := func(hres *http.Response) error {
		return fails.ErrorInvalidResponse(errors.New("存在しない科目に対する講義の追加に成功しました"), hres)
	}
	errAddConflictedClass := func(hres *http.Response) error {
		return fails.ErrorInvalidResponse(errors.New("科目IDとパートが重複した講義の追加に成功しました"), hres)
	}

	// ======== 検証用データの準備 ========

	// 検証で使用する講師ユーザ
	teacher, err := s.getLoggedInTeacher(ctx)
	if err != nil {
		return err
	}

	// 適当な科目
	courseParam := generate.CourseParam(0, 0, teacher)
	_, addCourseRes, err := AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	course := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity, model.NewCapacityCounter())

	// 存在しない科目
	courseParam = generate.CourseParam(0, 1, teacher)
	unknownCourse := model.NewCourse(courseParam, generate.GenULID(), teacher, prepareCourseCapacity, model.NewCapacityCounter())

	// ======== 検証 ========

	// 科目ステータスが registration での講義追加
	classParam := generate.ClassParam(course, 1)
	hres, _, err := AddClassAction(ctx, teacher.Agent, course, classParam)
	if err == nil {
		return errAddClassInvalidStatus(hres)
	}
	if err := verifyStatusCode(hres, []int{http.StatusBadRequest}); err != nil {
		return err
	}

	// 存在しない科目IDでの講義追加
	classParam = generate.ClassParam(unknownCourse, 1)
	hres, _, err = AddClassAction(ctx, teacher.Agent, unknownCourse, classParam)
	if err == nil {
		return errAddClassForUnknownCourse(hres)
	}
	if err := verifyStatusCode(hres, []int{http.StatusNotFound}); err != nil {
		return err
	}

	// ======== 検証用データの準備(2) ========

	// 科目ステータスを in-progress に変更
	_, err = SetCourseStatusInProgressAction(ctx, teacher.Agent, course.ID)
	if err != nil {
		return err
	}
	course.SetStatusToInProgress()

	// course の講義を追加
	classParam = generate.ClassParam(course, 1)
	_, addClassRes, err := AddClassAction(ctx, teacher.Agent, course, classParam)
	if err != nil {
		return err
	}
	class := model.NewClass(addClassRes.ClassID, classParam)

	// ======== 検証 ========

	// コンフリクトする講義の追加
	// course と partを同じにし、少なくともタイトルを変えることでコンフリクトさせる。
	classParam = generate.ClassParam(course, 1)
	classParam.Title = class.Title + "追記：講義室が変更になりました。"
	hres, _, err = AddClassAction(ctx, teacher.Agent, course, classParam)
	if err == nil {
		return errAddConflictedClass(hres)
	}
	if err := verifyStatusCode(hres, []int{http.StatusConflict}); err != nil {
		return err
	}

	// ======== 検証用データの準備(3) ========

	// 科目ステータスを closed に変更
	_, err = SetCourseStatusClosedAction(ctx, teacher.Agent, course.ID)
	if err != nil {
		return err
	}
	course.SetStatusToClosed()

	// ======== 検証 ========

	// 科目ステータスが closed での講義追加
	classParam = generate.ClassParam(course, 2)
	hres, _, err = AddClassAction(ctx, teacher.Agent, course, classParam)
	if err == nil {
		return errAddClassInvalidStatus(hres)
	}
	if err := verifyStatusCode(hres, []int{http.StatusBadRequest}); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) prepareCheckSubmitAssignmentAbnormal(ctx context.Context) error {
	errSubmitAssignmentForUnknownClass := func(hres *http.Response) error {
		return fails.ErrorInvalidResponse(errors.New("存在しない講義に対する課題提出に成功しました"), hres)
	}
	errSubmitAssignmentForNotRegisteredCourse := func(hres *http.Response) error {
		return fails.ErrorInvalidResponse(errors.New("履修していない科目の講義に対する課題提出に成功しました"), hres)
	}
	errSubmitAssignmentForSubmissionClosedClass := func(hres *http.Response) error {
		return fails.ErrorInvalidResponse(errors.New("課題提出が締め切られた講義に対する課題提出に成功しました"), hres)
	}
	errSubmitAssignmentForNotInProgressClass := func(hres *http.Response) error {
		return fails.ErrorInvalidResponse(errors.New("ステータスがin-progressでない科目の講義に対する課題提出が成功しました"), hres)
	}

	// ======== 検証用データの準備 ========

	// 検証で使用する学生ユーザ
	student, err := s.getLoggedInStudent(ctx)
	if err != nil {
		return err
	}

	// 検証で使用する講師ユーザ
	teacher, err := s.getLoggedInTeacher(ctx)
	if err != nil {
		return err
	}

	// student が履修登録済みで、in-progressの科目
	courseParam := generate.CourseParam(0, 0, teacher)
	_, addCourseRes, err := AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	inProgressCourse := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity, model.NewCapacityCounter())
	_, _, err = TakeCoursesAction(ctx, student.Agent, []*model.Course{inProgressCourse})
	if err != nil {
		return err
	}
	_, err = SetCourseStatusInProgressAction(ctx, teacher.Agent, inProgressCourse.ID)
	if err != nil {
		return err
	}
	inProgressCourse.SetStatusToInProgress()

	// student が履修していない、in-progressの科目
	courseParam = generate.CourseParam(0, 2, teacher)
	_, addCourseRes, err = AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	notRegisteredCourse := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity, model.NewCapacityCounter())
	_, err = SetCourseStatusInProgressAction(ctx, teacher.Agent, notRegisteredCourse.ID)
	if err != nil {
		return err
	}
	notRegisteredCourse.SetStatusToInProgress()

	// inProgressCourse の課題提出が締め切られた講義
	classParam := generate.ClassParam(inProgressCourse, 1)
	_, addClassRes, err := AddClassAction(ctx, teacher.Agent, inProgressCourse, classParam)
	if err != nil {
		return err
	}
	submissionClosedClass := model.NewClass(addClassRes.ClassID, classParam)
	_, _, err = DownloadSubmissionsAction(ctx, teacher.Agent, inProgressCourse.ID, submissionClosedClass.ID)
	if err != nil {
		return err
	}

	// inProgressCourse の課題提出が締め切られていない講義
	classParam = generate.ClassParam(inProgressCourse, 2)
	_, addClassRes, err = AddClassAction(ctx, teacher.Agent, inProgressCourse, classParam)
	if err != nil {
		return err
	}
	submissionNotClosedClass := model.NewClass(addClassRes.ClassID, classParam)

	// notRegisteredCourse の課題提出が締め切られていない講義
	classParam = generate.ClassParam(notRegisteredCourse, 1)
	_, addClassRes, err = AddClassAction(ctx, teacher.Agent, notRegisteredCourse, classParam)
	if err != nil {
		return err
	}
	submissionNotClosedClassOfNotRegisteredCourse := model.NewClass(addClassRes.ClassID, classParam)

	// ======== 検証 ========

	submissionData, fileName := generate.SubmissionData(inProgressCourse, submissionNotClosedClass, student.UserAccount)

	// 存在しない科目IDでの課題提出
	hres, err := SubmitAssignmentAction(ctx, student.Agent, generate.GenULID(), submissionNotClosedClass.ID, fileName, submissionData)
	if err == nil {
		return errSubmitAssignmentForUnknownClass(hres)
	}
	if err := verifyStatusCode(hres, []int{http.StatusNotFound}); err != nil {
		return err
	}

	// 存在しない講義IDでの課題提出
	hres, err = SubmitAssignmentAction(ctx, student.Agent, inProgressCourse.ID, generate.GenULID(), fileName, submissionData)
	if err == nil {
		return errSubmitAssignmentForUnknownClass(hres)
	}
	if err := verifyStatusCode(hres, []int{http.StatusNotFound}); err != nil {
		return err
	}

	// 履修していない科目への課題提出
	hres, err = SubmitAssignmentAction(ctx, student.Agent, notRegisteredCourse.ID, submissionNotClosedClassOfNotRegisteredCourse.ID, fileName, submissionData)
	if err == nil {
		return errSubmitAssignmentForNotRegisteredCourse(hres)
	}
	if err := verifyStatusCode(hres, []int{http.StatusBadRequest}); err != nil {
		return err
	}

	// 課題提出が締め切られた講義への課題提出
	hres, err = SubmitAssignmentAction(ctx, student.Agent, inProgressCourse.ID, submissionClosedClass.ID, fileName, submissionData)
	if err == nil {
		return errSubmitAssignmentForSubmissionClosedClass(hres)
	}
	if err := verifyStatusCode(hres, []int{http.StatusBadRequest}); err != nil {
		return err
	}

	// TODO: 不正な課題ファイルの提出で弾かれることのチェック
	// やるなら専用Action作らないといけないかも

	// ======== 検証用データの準備(2) ========

	// inProgressCourse のステータスを closed にする
	_, err = SetCourseStatusClosedAction(ctx, teacher.Agent, inProgressCourse.ID)
	if err != nil {
		return err
	}
	inProgressCourse.SetStatusToClosed()

	// ======== 検証 ========

	// 課題提出が締め切られていないが closed な科目への課題提出
	hres, err = SubmitAssignmentAction(ctx, student.Agent, inProgressCourse.ID, submissionNotClosedClass.ID, fileName, submissionData)
	if err == nil {
		return errSubmitAssignmentForNotInProgressClass(hres)
	}
	if err := verifyStatusCode(hres, []int{http.StatusBadRequest}); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) prepareCheckPostGradeAbnormal(ctx context.Context) error {
	errPostGradeForUnknownClass := func(hres *http.Response) error {
		return fails.ErrorInvalidResponse(errors.New("存在しない講義に対する成績登録に成功しました"), hres)
	}
	errPostGradeForSubmissionNotClosedClass := func(hres *http.Response) error {
		return fails.ErrorInvalidResponse(errors.New("課題提出が締め切られていない講義に対する成績登録に成功しました"), hres)
	}

	// ======== 検証用データの準備 ========

	// 検証で使用する学生ユーザ
	student, err := s.getLoggedInStudent(ctx)
	if err != nil {
		return err
	}

	// 検証で使用する講師ユーザ
	teacher, err := s.getLoggedInTeacher(ctx)
	if err != nil {
		return err
	}

	// 適当な科目
	courseParam := generate.CourseParam(0, 0, teacher)
	_, addCourseRes, err := AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	course := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity, model.NewCapacityCounter())
	_, err = SetCourseStatusInProgressAction(ctx, teacher.Agent, course.ID)
	if err != nil {
		return err
	}
	course.SetStatusToInProgress()

	// 課題提出が締め切られていない講義
	classParam := generate.ClassParam(course, 1)
	_, addClassRes, err := AddClassAction(ctx, teacher.Agent, course, classParam)
	if err != nil {
		return err
	}
	submissionNotClosedClass := model.NewClass(addClassRes.ClassID, classParam)

	// ======== 検証 ========

	scores := []StudentScore{
		{
			score: 90,
			code:  student.Code,
		},
	}

	// 存在しない講義IDでの成績登録
	hres, err := PostGradeAction(ctx, teacher.Agent, course.ID, generate.GenULID(), scores)
	if err == nil {
		return errPostGradeForUnknownClass(hres)
	}
	if err := verifyStatusCode(hres, []int{http.StatusNotFound}); err != nil {
		return err
	}

	// 課題提出が締め切られていない講義の成績登録
	hres, err = PostGradeAction(ctx, teacher.Agent, course.ID, submissionNotClosedClass.ID, scores)
	if err == nil {
		return errPostGradeForSubmissionNotClosedClass(hres)
	}
	if err := verifyStatusCode(hres, []int{http.StatusBadRequest}); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) prepareCheckDownloadSubmissionsAbnormal(ctx context.Context) error {
	errDownloadSubmissionsForUnknownClass := func(hres *http.Response) error {
		return fails.ErrorInvalidResponse(errors.New("存在しない講義の課題ダウンロードに成功しました"), hres)
	}

	// ======== 検証用データの準備 ========

	// 検証で使用する講師ユーザ
	teacher, err := s.getLoggedInTeacher(ctx)
	if err != nil {
		return err
	}

	// 適当な科目
	courseParam := generate.CourseParam(0, 0, teacher)
	_, addCourseRes, err := AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	course := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity, model.NewCapacityCounter())

	// ======== 検証 ========

	// 存在しない講義IDでの課題ダウンロード
	hres, _, err := DownloadSubmissionsAction(ctx, teacher.Agent, course.ID, generate.GenULID())
	if err == nil {
		return errDownloadSubmissionsForUnknownClass(hres)
	}
	if err := verifyStatusCode(hres, []int{http.StatusNotFound}); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) prepareCheckSendAnnouncementAbnormal(ctx context.Context) error {
	errSendAnnouncementForUnknownCourse := func(hres *http.Response) error {
		return fails.ErrorInvalidResponse(errors.New("存在しない科目のお知らせ追加に成功しました"), hres)
	}

	// ======== 検証用データの準備 ========

	// 検証で使用する講師ユーザ
	teacher, err := s.getLoggedInTeacher(ctx)
	if err != nil {
		return err
	}

	// 存在しない科目
	courseParam := generate.CourseParam(0, 0, teacher)
	notRegisteredCourse := model.NewCourse(courseParam, generate.GenULID(), teacher, prepareCourseCapacity, model.NewCapacityCounter())

	// 存在しない講義
	classParam := generate.ClassParam(notRegisteredCourse, 1)
	class := model.NewClass(generate.GenULID(), classParam)

	// ======== 検証 ========

	// 存在しない科目IDでのお知らせ追加
	announcement := generate.Announcement(notRegisteredCourse, class)
	hres, err := SendAnnouncementAction(ctx, teacher.Agent, announcement)
	if err == nil {
		return errSendAnnouncementForUnknownCourse(hres)
	}
	if err := verifyStatusCode(hres, []int{http.StatusNotFound}); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) prepareCheckGetAnnouncementDetailAbnormal(ctx context.Context) error {
	errGetClassesForUnknownCourse := func(hres *http.Response) error {
		return fails.ErrorInvalidResponse(errors.New("存在しないお知らせの詳細取得に成功しました"), hres)
	}
	errGetClassesForNotRegisteredCourse := func(hres *http.Response) error {
		return fails.ErrorInvalidResponse(errors.New("履修していない科目のお知らせ詳細取得に成功しました"), hres)
	}

	// ======== 検証用データの準備 ========

	// 検証で使用する学生ユーザ
	student, err := s.getLoggedInStudent(ctx)
	if err != nil {
		return err
	}

	// 検証で使用する講師ユーザ
	teacher, err := s.getLoggedInTeacher(ctx)
	if err != nil {
		return err
	}

	// student が履修していない科目
	courseParam := generate.CourseParam(0, 0, teacher)
	_, addCourseRes, err := AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	notRegisteredCourse := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity, model.NewCapacityCounter())
	_, err = SetCourseStatusInProgressAction(ctx, teacher.Agent, notRegisteredCourse.ID)
	if err != nil {
		return err
	}
	notRegisteredCourse.SetStatusToInProgress()

	// notRegisteredCourse の講義
	classParam := generate.ClassParam(notRegisteredCourse, 1)
	_, addClassRes, err := AddClassAction(ctx, teacher.Agent, notRegisteredCourse, classParam)
	if err != nil {
		return err
	}
	class := model.NewClass(addClassRes.ClassID, classParam)

	// notRegisteredCourse に紐づくお知らせ
	announcement := generate.Announcement(notRegisteredCourse, class)
	_, err = SendAnnouncementAction(ctx, teacher.Agent, announcement)
	if err != nil {
		return err
	}

	// ======== 検証 ========

	// 存在しないお知らせIDでのお知らせ詳細取得
	hres, _, err := GetAnnouncementDetailAction(ctx, student.Agent, generate.GenULID())
	if err == nil {
		return errGetClassesForUnknownCourse(hres)
	}
	if err := verifyStatusCode(hres, []int{http.StatusNotFound}); err != nil {
		return err
	}

	// 履修していない科目に紐づくお知らせIDでのお知らせ詳細取得
	hres, _, err = GetAnnouncementDetailAction(ctx, student.Agent, announcement.ID)
	if err == nil {
		return errGetClassesForNotRegisteredCourse(hres)
	}
	if err := verifyStatusCode(hres, []int{http.StatusNotFound}); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) getLoggedInStudent(ctx context.Context) (*model.Student, error) {
	student, err := s.userPool.newStudent()
	if err != nil {
		panic("unreachable! studentPool is empty")
	}
	_, err = LoginAction(ctx, student.Agent, student.UserAccount)
	if err != nil {
		return nil, err
	}

	return student, nil
}

func (s *Scenario) getLoggedInTeacher(ctx context.Context) (*model.Teacher, error) {
	teacher := s.userPool.randomTeacher()
	isLoggedIn := teacher.LoginOnce(func(teacher *model.Teacher) {
		_, err := LoginAction(ctx, teacher.Agent, teacher.UserAccount)
		if err != nil {
			return
		}
		teacher.IsLoggedIn = true
	})
	if !isLoggedIn {
		return nil, fmt.Errorf("teacherのログインに失敗しました")
	}

	return teacher, nil
}
