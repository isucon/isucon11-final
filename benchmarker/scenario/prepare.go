package scenario

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucandar/parallel"
	"github.com/isucon/isucandar/random/useragent"
	"github.com/isucon/isucandar/worker"

	"github.com/isucon/isucon11-final/benchmarker/api"
	"github.com/isucon/isucon11-final/benchmarker/fails"
	"github.com/isucon/isucon11-final/benchmarker/generate"
	"github.com/isucon/isucon11-final/benchmarker/model"

	"github.com/pborman/uuid"
)

const (
	prepareTimeout           = 20
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
	)
	if err != nil {
		return failure.NewError(fails.ErrCritical, err)
	}

	a.Name = "benchmarker-initializer"

	ContestantLogger.Printf("start Initialize")
	_, err = InitializeAction(ctx, a)
	if err != nil {
		ContestantLogger.Printf("initializeが失敗しました")
		return failure.NewError(fails.ErrCritical, err)
	}

	err = s.prepareNormal(ctx, step)
	if err != nil {
		return failure.NewError(fails.ErrCritical, err)
	}

	err = s.prepareAnnouncementsList(ctx, step)
	if err != nil {
		return failure.NewError(fails.ErrCritical, err)
	}

	err = s.prepareAbnormal(ctx, step)
	if err != nil {
		return failure.NewError(fails.ErrCritical, err)
	}

	_, err = InitializeAction(ctx, a)
	if err != nil {
		ContestantLogger.Printf("initializeが失敗しました")
		return failure.NewError(fails.ErrCritical, err)
	}
	return nil
}

func (s *Scenario) prepareCheck(parent context.Context, step *isucandar.BenchmarkStep) error {
	initializeAgent, err := agent.NewAgent(
		agent.WithNoCache(),
		agent.WithNoCookie(),
		agent.WithTimeout(20*time.Second),
		agent.WithBaseURL(s.BaseURL.String()),
	)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	_, err = InitializeAction(ctx, initializeAgent)
	if err != nil {
		return err
	}

	//studentAgent, err := agent.NewAgent(agent.WithTimeout(prepareTimeout))
	//if err != nil {
	//	return err
	//}
	//student := s.prepareNewStudent()
	//student.Agent = studentAgent
	//
	//teacherAgent, err := agent.NewAgent(agent.WithTimeout(prepareTimeout))
	//if err != nil {
	//	return err
	//}
	//teacher := s.prepareNewTeacher()
	//teacher.Agent = teacherAgent

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
		teachers = append(teachers, s.GetRandomTeacher())
	}

	students := make([]*model.Student, 0, prepareStudentCount)
	for i := 0; i < prepareStudentCount; i++ {
		userData, err := s.studentPool.newUserData()
		if err != nil {
			return err
		}
		students = append(students, model.NewStudent(userData, s.BaseURL))
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
				step.AddError(failure.NewError(fails.ErrCritical, err))
				return
			}
			teacher.IsLoggedIn = true
		})
		if !isLoggedIn {
			return
		}

		_, getMeRes, err := GetMeAction(ctx, teacher.Agent)
		if err != nil {
			AdminLogger.Printf("teacherのユーザ情報取得に失敗しました")
			step.AddError(err)
			return
		}
		if err := verifyMe(&getMeRes, teacher.UserAccount, true); err != nil {
			step.AddError(err)
			return
		}

		param := generate.CourseParam((i/6)+1, i%6, teacher)
		_, res, err := AddCourseAction(ctx, teacher.Agent, param)
		if err != nil {
			step.AddError(err)
			return
		}
		course := model.NewCourse(param, res.ID, teacher, prepareCourseCapacity)
		mu.Lock()
		courses = append(courses, course)
		mu.Unlock()
	}, worker.WithLoopCount(prepareCourseCount))

	if err != nil {
		step.AddError(err)
		return err
	}

	w.Process(ctx)
	w.Wait()

	if hasErrors() {
		return failure.NewError(fails.ErrCritical, fmt.Errorf("アプリケーション互換性チェックに失敗しました"))
	}

	// 生徒のログインとコース登録
	w, err = worker.NewWorker(func(ctx context.Context, i int) {
		student := students[i]
		_, err := LoginAction(ctx, student.Agent, student.UserAccount)
		if err != nil {
			AdminLogger.Printf("studentのログインに失敗しました")
			step.AddError(failure.NewError(fails.ErrCritical, err))
			return
		}

		_, getMeRes, err := GetMeAction(ctx, student.Agent)
		if err != nil {
			AdminLogger.Printf("studentのユーザ情報取得に失敗しました")
			step.AddError(err)
			return
		}
		if err := verifyMe(&getMeRes, student.UserAccount, false); err != nil {
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
		step.AddError(err)
		return err
	}
	w.Process(ctx)
	w.Wait()
	if hasErrors() {
		return failure.NewError(fails.ErrCritical, fmt.Errorf("アプリケーション互換性チェックに失敗しました"))
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
	}, worker.WithLoopCount(prepareCourseCount))
	if err != nil {
		step.AddError(err)
		return err
	}
	w.Process(ctx)
	w.Wait()

	if hasErrors() {
		return failure.NewError(fails.ErrCritical, fmt.Errorf("アプリケーション互換性チェックに失敗しました"))
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
			_, ancRes, err := SendAnnouncementAction(ctx, teacher.Agent, announcement)
			if err != nil {
				step.AddError(err)
				return
			}
			announcement.ID = ancRes.ID
			course.BroadCastAnnouncement(announcement)

			courseStudents := course.Students()

			// 課題提出, ランダムでお知らせを読む
			// 生徒ごとのループ
			p := parallel.NewParallel(ctx, prepareStudentCount)
			for _, student := range courseStudents {
				student := student
				p.Do(func(ctx context.Context) {
					if classPart == checkAnnouncementDetailPart {
						_, res, err := GetAnnouncementDetailAction(ctx, student.Agent, announcement.ID)
						if err != nil {
							step.AddError(err)
							return
						}
						expected := student.GetAnnouncement(announcement.ID)
						if expected == nil {
							panic("unreachable! announcementID" + announcement.ID)
						}
						err = prepareCheckAnnouncementDetailContent(expected, &res)
						if err != nil {
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
			}
			p.Wait()

			// 課題ダウンロード
			_, assignmentsData, err := DownloadSubmissionsAction(ctx, teacher.Agent, course.ID, class.ID)
			if err != nil {
				step.AddError(err)
				return
			}
			if err := verifyAssignments(assignmentsData, class); err != nil {
				step.AddError(err)
				return
			}

			// 採点
			scores := make([]StudentScore, 0, len(students))
			for _, student := range students {
				sub := class.GetSubmissionByStudentCode(student.Code)
				if sub == nil {
					step.AddError(failure.NewError(fails.ErrCritical, fmt.Errorf("cannot find submission")))
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
			step.AddError(err)
			return err
		}
		w.Process(ctx)
		w.Wait()

		w, err = worker.NewWorker(func(ctx context.Context, i int) {
			student := students[i]
			expected := calculateGradeRes(student, studentByCode)
			_, res, err := GetGradeAction(ctx, student.Agent)
			if err != nil {
				step.AddError(failure.NewError(fails.ErrCritical, err))
				return
			}

			err = validateUserGrade(&expected, &res)
			if err != nil {
				step.AddError(err)
				return
			}
		}, worker.WithLoopCount(prepareStudentCount))
		if err != nil {
			step.AddError(err)
			return err
		}
		w.Process(ctx)
		w.Wait()
	}

	if hasErrors() {
		return failure.NewError(fails.ErrCritical, fmt.Errorf("アプリケーション互換性チェックに失敗しました"))
	}

	// お知らせの検証
	w, err = worker.NewWorker(func(ctx context.Context, i int) {
		student := students[i]
		expected := student.Announcements()

		// createdAtが新しい方が先頭に来るようにソート
		sort.Slice(expected, func(i, j int) bool {
			return expected[i].Announcement.CreatedAt > expected[j].Announcement.CreatedAt
		})
		expectedUnreadCount := 0
		for _, announcement := range expected {
			if announcement.Unread {
				expectedUnreadCount++
			}
		}
		_, err := prepareCheckAnnouncementsList(ctx, student.Agent, "", expected, expectedUnreadCount)
		if err != nil {
			step.AddError(err)
			return
		}

	}, worker.WithLoopCount(prepareStudentCount))
	if err != nil {
		step.AddError(err)
		return err
	}
	w.Process(ctx)
	w.Wait()
	if hasErrors() {
		return failure.NewError(fails.ErrCritical, fmt.Errorf("アプリケーション互換性チェックに失敗しました"))
	}

	return nil
}

func (s *Scenario) prepareAnnouncementsList(ctx context.Context, step *isucandar.BenchmarkStep) error {
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
		return failure.NewError(fails.ErrCritical, fmt.Errorf("アプリケーション互換性チェックに失敗しました"))
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
		return failure.NewError(fails.ErrCritical, fmt.Errorf("アプリケーション互換性チェックに失敗しました"))
	}

	// コース登録
	var mu sync.Mutex
	courses := make([]*model.Course, 0, prepareCheckAnnouncementListCourseCount)
	w, err := worker.NewWorker(func(ctx context.Context, i int) {
		teacher := teachers[i%len(teachers)]
		param := generate.CourseParam((i/6)+1, i%6, teacher)
		_, res, err := AddCourseAction(ctx, teacher.Agent, param)
		if err != nil {
			step.AddError(err)
			return
		}
		course := model.NewCourse(param, res.ID, teacher, prepareCourseCapacity)
		mu.Lock()
		courses = append(courses, course)
		mu.Unlock()
	}, worker.WithLoopCount(prepareCheckAnnouncementListCourseCount))
	if err != nil {
		step.AddError(err)
		return err
	}
	w.Process(ctx)
	w.Wait()

	if hasErrors() {
		return failure.NewError(fails.ErrCritical, fmt.Errorf("アプリケーション互換性チェックに失敗しました"))
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
		step.AddError(err)
		return err
	}
	w.Process(ctx)
	w.Wait()

	if hasErrors() {
		return failure.NewError(fails.ErrCritical, fmt.Errorf("アプリケーション互換性チェックに失敗しました"))
	}

	// クラス追加、おしらせ追加をする
	// そのたびにおしらせリストを確認する
	// 既読にはしない
	for classPart := 0; classPart < prepareCheckAnnouncementListClassCountPerCourse; classPart++ {
		for j := 0; j < prepareCheckAnnouncementListCourseCount; j++ {
			course := courses[j]
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
			_, ancRes, err := SendAnnouncementAction(ctx, teacher.Agent, announcement)
			if err != nil {
				step.AddError(err)
				return err
			}
			announcement.ID = ancRes.ID
			course.BroadCastAnnouncement(announcement)

			// 生徒ごとにおしらせリストの確認
			courseStudents := course.Students()
			p := parallel.NewParallel(ctx, int32(len(courseStudents)))
			for _, student := range courseStudents {
				student := student
				err := p.Do(func(ctx context.Context) {
					expected := student.Announcements()

					// createdAtが新しい方が先頭に来るようにソート
					sort.Slice(expected, func(i, j int) bool {
						return expected[i].Announcement.CreatedAt > expected[j].Announcement.CreatedAt
					})
					_, err := prepareCheckAnnouncementsList(ctx, student.Agent, "", expected, len(expected))
					if err != nil {
						step.AddError(err)
						return
					}
				})
				if err != nil {
					return err
				}
			}
			p.Wait()
		}
	}

	if hasErrors() {
		return failure.NewError(fails.ErrCritical, fmt.Errorf("アプリケーション互換性チェックに失敗しました"))
	}

	return nil
}

func prepareCheckAnnouncementsList(ctx context.Context, a *agent.Agent, path string, expected []*model.AnnouncementStatus, expectedUnreadCount int) (prev string, err error) {
	errHttp := failure.NewError(fails.ErrCritical, fmt.Errorf("/api/announcements へのリクエストが失敗しました"))
	errInvalidNext := failure.NewError(fails.ErrCritical, fmt.Errorf("link header の next によってページングできる回数が不正です"))

	hres, res, err := GetAnnouncementListAction(ctx, a, path)
	if err != nil {
		return "", errHttp
	}
	prev, next := parseLinkHeader(hres)

	if (len(expected) <= AnnouncementCountPerPage && next != "") || (len(expected) > AnnouncementCountPerPage && next == "") {
		return "", errInvalidNext
	}
	// 次のページが存在しない
	if next == "" {
		err = prepareCheckAnnouncementContent(expected, res, expectedUnreadCount)
		if err != nil {
			return "", err
		}
		return prev, nil
	}

	err = prepareCheckAnnouncementContent(expected[:AnnouncementCountPerPage], res, expectedUnreadCount)
	if err != nil {
		return "", err
	}

	// この_prevはpathと同じところを指すはず
	// _prevとpathが同じ文字列であるとは限らない（pathが"" で_prevが/api/announcements?page=1とか）
	_prev, err := prepareCheckAnnouncementsList(ctx, a, next, expected[AnnouncementCountPerPage:], expectedUnreadCount)
	if err != nil {
		return "", err
	}

	_, res, err = GetAnnouncementListAction(ctx, a, _prev)
	if err != nil {
		return "", errHttp
	}

	err = prepareCheckAnnouncementContent(expected[:AnnouncementCountPerPage], res, expectedUnreadCount)
	if err != nil {
		return "", err
	}

	return prev, nil
}

func prepareCheckAnnouncementContent(expected []*model.AnnouncementStatus, actual api.GetAnnouncementsResponse, expectedUnreadCount int) error {
	errNotSorted := failure.NewError(fails.ErrCritical, fmt.Errorf("/api/announcements の順序が不正です"))
	errNotMatch := failure.NewError(fails.ErrCritical, fmt.Errorf("announcement が期待したものと一致しませんでした"))
	errNoCount := failure.NewError(fails.ErrCritical, fmt.Errorf("announcement の数が期待したものと一致しませんでした"))
	errNoMatchUnreadCount := failure.NewError(fails.ErrCritical, fmt.Errorf("announcement の unread_count が期待したものと一致しませんでした"))

	if actual.UnreadCount != expectedUnreadCount {
		return errNoMatchUnreadCount
	}

	if len(expected) != len(actual.Announcements) {
		return errNoCount
	}

	if expected == nil && actual.Announcements == nil {
		return nil
	} else if (expected == nil && actual.Announcements != nil) || (expected != nil && actual.Announcements == nil) {
		return errNotMatch
	}

	lastCreatedAt := int64(math.MaxInt64)
	for _, announcement := range actual.Announcements {
		// 順序の検証
		if lastCreatedAt < announcement.CreatedAt {
			return errNotSorted
		}
		lastCreatedAt = announcement.CreatedAt
	}
	for i := 0; i < len(actual.Announcements); i++ {
		expect := expected[i].Announcement
		actual := actual.Announcements[i]
		if !AssertEqual("announcement unread", expected[i].Unread, actual.Unread) ||
			!AssertEqual("announcement ID", expect.ID, actual.ID) ||
			!AssertEqual("announcement CourseID", expect.CourseID, actual.CourseID) ||
			!AssertEqual("announcement Title", expect.Title, actual.Title) ||
			!AssertEqual("announcement CourseName", expect.CourseName, actual.CourseName) ||
			!AssertEqual("announcement CreatedAt", expect.CreatedAt, actual.CreatedAt) {
			AdminLogger.Printf("extra announcements ->name: %v, title:  %v", actual.CourseName, actual.Title)
			return errNotMatch
		}
	}

	return nil
}

func prepareCheckAnnouncementDetailContent(expected *model.AnnouncementStatus, actual *api.GetAnnouncementDetailResponse) error {
	errNotMatch := failure.NewError(fails.ErrCritical, fmt.Errorf("announcement が期待したものと一致しませんでした"))
	if !AssertEqual("announcement unread", expected.Unread, actual.Unread) ||
		!AssertEqual("announcement ID", expected.Announcement.ID, actual.ID) ||
		!AssertEqual("announcement Title", expected.Announcement.Title, actual.Title) ||
		!AssertEqual("announcement CourseID", expected.Announcement.CourseID, actual.CourseID) ||
		!AssertEqual("announcement CourseName", expected.Announcement.CourseName, actual.CourseName) ||
		!AssertEqual("announcement CreatedAt", expected.Announcement.CreatedAt, actual.CreatedAt) ||
		!AssertEqual("announcement Message", expected.Announcement.Message, actual.Message) {
		AdminLogger.Printf("extra announcements ->name: %v, title:  %v", actual.CourseName, actual.Title)
		return errNotMatch
	}

	return nil
}

func (s *Scenario) prepareAbnormal(ctx context.Context, step *isucandar.BenchmarkStep) error {
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

	// GET /api/syllabus/:courseID
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
	errAuthentication := failure.NewError(fails.ErrApplication, fmt.Errorf("未ログイン状態で認証が必要なAPIへのアクセスが成功しました"))
	checkAuthentication := func(hres *http.Response, err error) error {
		// リクエストが成功したらwebappの不具合
		if err == nil {
			return errAuthentication
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
	courseParam := generate.CourseParam(1, 0, teacher)
	_, addCourseRes, err := AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	course := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity)

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
	_, announcementRes, err := SendAnnouncementAction(ctx, teacher.Agent, announcement1)
	if err != nil {
		return err
	}
	announcement1.ID = announcementRes.ID

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

	courseParam = generate.CourseParam(1, 1, teacher)
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

	hres, _, err = GetAnnouncementListAction(ctx, agent, "")
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	announcement2 := generate.Announcement(course, submissionNotClosedClass)
	hres, _, err = SendAnnouncementAction(ctx, agent, announcement2)
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
	errAuthorization := failure.NewError(fails.ErrApplication, fmt.Errorf("学生ユーザで講師用APIへのアクセスが成功しました"))
	checkAuthorization := func(hres *http.Response, err error) error {
		// リクエストが成功したらwebappの不具合
		if err == nil {
			return errAuthorization
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
	courseParam := generate.CourseParam(1, 0, teacher)
	_, addCourseRes, err := AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	course := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity)

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

	courseParam = generate.CourseParam(1, 1, teacher)
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
	hres, _, err = SendAnnouncementAction(ctx, student.Agent, announcement)
	if err := checkAuthorization(hres, err); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) prepareCheckLoginAbnormal(ctx context.Context) error {
	errInvalidLogin := failure.NewError(fails.ErrApplication, fmt.Errorf("間違った認証情報でのログインに成功しました"))
	errRelogin := failure.NewError(fails.ErrApplication, fmt.Errorf("ログイン状態での再ログインに成功しました"))

	// ======== 検証用データの準備 ========

	// 検証で使用する学生ユーザ（未ログイン状態）
	userData, err := s.studentPool.newUserData()
	if err != nil {
		panic("unreachable! studentPool is empty")
	}
	student := model.NewStudent(userData, s.BaseURL)

	// ======== 検証 ========

	// 存在しないユーザでのログイン
	hres, err := LoginAction(ctx, student.Agent, &model.UserAccount{
		Code:        "X12345",
		Name:        "unknown",
		RawPassword: "password",
	})
	if err == nil {
		return errInvalidLogin
	}
	if err := verifyStatusCode(hres, []int{http.StatusUnauthorized}); err != nil {
		return err
	}

	// 間違ったパスワードでのログイン
	hres, err = LoginAction(ctx, student.Agent, &model.UserAccount{
		Code:        student.Code,
		Name:        student.Name,
		RawPassword: student.RawPassword + "abc",
	})
	if err == nil {
		return errInvalidLogin
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
		return errRelogin
	}
	if err := verifyStatusCode(hres, []int{http.StatusBadRequest}); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) prepareCheckRegisterCoursesAbnormal(ctx context.Context) error {
	errInvalidRegistration := failure.NewError(fails.ErrApplication, fmt.Errorf("履修登録できないはずの科目の履修に成功しました"))
	errInvalidErrorResponce := errInvalidResponse("履修登録失敗時のレスポンスが期待する内容と一致しません")

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
	courseParam := generate.CourseParam(1, 0, teacher)
	_, addCourseRes, err := AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	registrationCourse := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity)

	// ステータスが in-progress の科目
	courseParam = generate.CourseParam(1, 1, teacher)
	_, addCourseRes, err = AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	inProgressCourse := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity)
	_, err = SetCourseStatusInProgressAction(ctx, teacher.Agent, inProgressCourse.ID)
	if err != nil {
		return err
	}

	// ステータスが closed の科目
	courseParam = generate.CourseParam(1, 2, teacher)
	_, addCourseRes, err = AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	closedCourse := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity)
	_, err = SetCourseStatusClosedAction(ctx, teacher.Agent, closedCourse.ID)
	if err != nil {
		return err
	}

	// student が履修登録済みの科目
	courseParam = generate.CourseParam(2, 0, teacher)
	_, addCourseRes, err = AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	alreadyRegisteredCourse := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity)
	_, _, err = TakeCoursesAction(ctx, student.Agent, []*model.Course{alreadyRegisteredCourse})
	if err != nil {
		return err
	}

	// alreadyRegisteredCourse と時間割がコンフリクトする科目
	courseParam = generate.CourseParam(2, 0, teacher)
	_, addCourseRes, err = AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	conflictedCourse1 := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity)

	// 時間割がコンフリクトする2つの科目
	courseParam = generate.CourseParam(2, 1, teacher)
	_, addCourseRes, err = AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	conflictedCourse2 := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity)

	courseParam = generate.CourseParam(2, 1, teacher)
	_, addCourseRes, err = AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	conflictedCourse3 := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity)

	// 存在しない科目
	courseParam = generate.CourseParam(3, 0, teacher)
	unknownCourse := model.NewCourse(courseParam, uuid.NewRandom().String(), teacher, prepareCourseCapacity)

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
		return errInvalidRegistration
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
		return errInvalidErrorResponce
	}

	return nil
}

func (s *Scenario) prepareCheckGetCourseDetailAbnormal(ctx context.Context) error {
	errGetUnknownCourseDetail := failure.NewError(fails.ErrApplication, fmt.Errorf("存在しない科目の詳細取得に成功しました"))

	// ======== 検証用データの準備 ========

	// 検証で使用する学生ユーザ
	student, err := s.getLoggedInStudent(ctx)
	if err != nil {
		return err
	}

	// ======== 検証 ========

	// 存在しない科目IDでの科目詳細取得
	hres, _, err := GetCourseDetailAction(ctx, student.Agent, uuid.NewRandom().String())
	if err == nil {
		return errGetUnknownCourseDetail
	}
	if err := verifyStatusCode(hres, []int{http.StatusNotFound}); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) prepareCheckAddCourseAbnormal(ctx context.Context) error {
	errAddInvalidCourse := failure.NewError(fails.ErrApplication, fmt.Errorf("不正な科目の追加に成功しました"))
	errAddConflictedCourse := failure.NewError(fails.ErrApplication, fmt.Errorf("コードが重複した科目の追加に成功しました"))

	// ======== 検証用データの準備 ========

	// 検証で使用する講師ユーザ
	teacher, err := s.getLoggedInTeacher(ctx)
	if err != nil {
		return err
	}

	// 適当な科目
	courseParam := generate.CourseParam(1, 0, teacher)
	_, addCourseRes, err := AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	course := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity)

	// ======== 検証 ========

	// Type が不正な科目の追加
	courseParam = generate.CourseParam(1, 1, teacher)
	courseParam.Type = "invalid-type"
	hres, _, err := AddCourseAction(ctx, teacher.Agent, courseParam)
	if err == nil {
		return errAddInvalidCourse
	}
	if err := verifyStatusCode(hres, []int{http.StatusBadRequest}); err != nil {
		return err
	}

	// DayOfWeek が不正な科目の追加
	// courseParam.DayOfWeek を0-6以外にしておくとAction側の処理で空文字列（不正な入力）として送信される
	courseParam = generate.CourseParam(-1, 0, teacher)
	hres, _, err = AddCourseAction(ctx, teacher.Agent, courseParam)
	if err == nil {
		return errAddInvalidCourse
	}
	if err := verifyStatusCode(hres, []int{http.StatusBadRequest}); err != nil {
		return err
	}

	// コンフリクトする科目の追加
	// Code を同じにし、少なくとも Period を変えることで course とコンフリクトさせる
	courseParam = generate.CourseParam(1, (course.Period+1)%6, teacher)
	courseParam.Code = course.Code
	hres, _, err = AddCourseAction(ctx, teacher.Agent, courseParam)
	if err == nil {
		return errAddConflictedCourse
	}
	if err := verifyStatusCode(hres, []int{http.StatusConflict}); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) prepareCheckSetCourseStatusAbnormal(ctx context.Context) error {
	errSetStatusForUnknownCourse := failure.NewError(fails.ErrApplication, fmt.Errorf("存在しない科目のステータス変更に成功しました"))

	// ======== 検証用データの準備 ========

	// 検証で使用する講師ユーザ
	teacher, err := s.getLoggedInTeacher(ctx)
	if err != nil {
		return err
	}

	// ======== 検証 ========

	// 存在しない科目IDでの科目ステータス変更
	hres, err := SetCourseStatusInProgressAction(ctx, teacher.Agent, uuid.NewRandom().String())
	if err == nil {
		return errSetStatusForUnknownCourse
	}
	if err := verifyStatusCode(hres, []int{http.StatusNotFound}); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) prepareCheckGetClassesAbnormal(ctx context.Context) error {
	errGetClassesForUnknownCourse := failure.NewError(fails.ErrApplication, fmt.Errorf("存在しない科目の講義一覧取得に成功しました"))

	// ======== 検証用データの準備 ========

	// 検証で使用する学生ユーザ
	student, err := s.getLoggedInStudent(ctx)
	if err != nil {
		return err
	}

	// ======== 検証 ========

	// 存在しない科目IDでの講義一覧取得
	hres, _, err := GetClassesAction(ctx, student.Agent, uuid.NewRandom().String())
	if err == nil {
		return errGetClassesForUnknownCourse
	}
	if err := verifyStatusCode(hres, []int{http.StatusNotFound}); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) prepareCheckAddClassAbnormal(ctx context.Context) error {
	errAddClassForUnknownCourse := failure.NewError(fails.ErrApplication, fmt.Errorf("存在しない科目に対する講義の追加に成功しました"))
	errAddConflictedClass := failure.NewError(fails.ErrApplication, fmt.Errorf("科目IDとパートが重複した講義の追加に成功しました"))

	// ======== 検証用データの準備 ========

	// 検証で使用する講師ユーザ
	teacher, err := s.getLoggedInTeacher(ctx)
	if err != nil {
		return err
	}

	// 適当な科目
	courseParam := generate.CourseParam(1, 0, teacher)
	_, addCourseRes, err := AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	course := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity)

	// course の講義
	classParam := generate.ClassParam(course, 1)
	_, addClassRes, err := AddClassAction(ctx, teacher.Agent, course, classParam)
	if err != nil {
		return err
	}
	class := model.NewClass(addClassRes.ClassID, classParam)

	// 存在しない科目
	courseParam = generate.CourseParam(1, 1, teacher)
	unknownCourse := model.NewCourse(courseParam, uuid.NewRandom().String(), teacher, prepareCourseCapacity)

	// ======== 検証 ========

	// 存在しない科目IDでの講義追加
	classParam = generate.ClassParam(unknownCourse, 1)
	hres, _, err := AddClassAction(ctx, teacher.Agent, unknownCourse, classParam)
	if err == nil {
		return errAddClassForUnknownCourse
	}
	if err := verifyStatusCode(hres, []int{http.StatusNotFound}); err != nil {
		return err
	}

	// コンフリクトする講義の追加
	// course と partを同じにし、少なくともタイトルを変えることでコンフリクトさせる。
	classParam = generate.ClassParam(course, 1)
	classParam.Title = class.Title + "追記：講義室が変更になりました。"
	hres, _, err = AddClassAction(ctx, teacher.Agent, course, classParam)
	if err == nil {
		return errAddConflictedClass
	}
	if err := verifyStatusCode(hres, []int{http.StatusConflict}); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) prepareCheckSubmitAssignmentAbnormal(ctx context.Context) error {
	errSubmitAssignmentForUnknownClass := failure.NewError(fails.ErrApplication, fmt.Errorf("存在しない講義に対する課題提出に成功しました"))
	errSubmitAssignmentForNotInProgressClass := failure.NewError(fails.ErrApplication, fmt.Errorf("ステータスがin-progressでない科目の講義に対する課題提出に成功しました"))
	errSubmitAssignmentForNotRegisteredCourse := failure.NewError(fails.ErrApplication, fmt.Errorf("履修していない科目の講義に対する課題提出に成功しました"))
	errSubmitAssignmentForSubmissionClosedClass := failure.NewError(fails.ErrApplication, fmt.Errorf("課題提出が締め切られた講義に対する課題提出に成功しました"))

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
	courseParam := generate.CourseParam(1, 0, teacher)
	_, addCourseRes, err := AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	inProgressCourse := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity)
	_, _, err = TakeCoursesAction(ctx, student.Agent, []*model.Course{inProgressCourse})
	if err != nil {
		return err
	}
	_, err = SetCourseStatusInProgressAction(ctx, teacher.Agent, inProgressCourse.ID)
	if err != nil {
		return err
	}

	// student が履修登録済みで、in-progressではない科目
	courseParam = generate.CourseParam(1, 1, teacher)
	_, addCourseRes, err = AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	notInProgressCourse := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity)
	_, _, err = TakeCoursesAction(ctx, student.Agent, []*model.Course{notInProgressCourse})
	if err != nil {
		return err
	}

	// student が履修していない、in-progressの科目
	courseParam = generate.CourseParam(1, 2, teacher)
	_, addCourseRes, err = AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	notRegisteredCourse := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity)
	_, err = SetCourseStatusInProgressAction(ctx, teacher.Agent, notRegisteredCourse.ID)
	if err != nil {
		return err
	}

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

	// notInProgressCourse の課題提出が締め切られていない講義
	classParam = generate.ClassParam(notInProgressCourse, 1)
	_, addClassRes, err = AddClassAction(ctx, teacher.Agent, notInProgressCourse, classParam)
	if err != nil {
		return err
	}
	submissionNotClosedClassOfNotInProgresCourse := model.NewClass(addClassRes.ClassID, classParam)

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
	hres, err := SubmitAssignmentAction(ctx, student.Agent, uuid.NewRandom().String(), submissionNotClosedClass.ID, fileName, submissionData)
	if err == nil {
		return errSubmitAssignmentForUnknownClass
	}
	if err := verifyStatusCode(hres, []int{http.StatusBadRequest}); err != nil {
		return err
	}

	// 存在しない講義IDでの課題提出
	hres, err = SubmitAssignmentAction(ctx, student.Agent, inProgressCourse.ID, uuid.NewRandom().String(), fileName, submissionData)
	if err == nil {
		return errSubmitAssignmentForUnknownClass
	}
	if err := verifyStatusCode(hres, []int{http.StatusBadRequest}); err != nil {
		return err
	}

	// in-progressでない科目の講義への課題提出
	hres, err = SubmitAssignmentAction(ctx, student.Agent, notInProgressCourse.ID, submissionNotClosedClassOfNotInProgresCourse.ID, fileName, submissionData)
	if err == nil {
		return errSubmitAssignmentForNotInProgressClass
	}
	if err := verifyStatusCode(hres, []int{http.StatusBadRequest}); err != nil {
		return err
	}

	// 履修していない科目への課題提出
	hres, err = SubmitAssignmentAction(ctx, student.Agent, notRegisteredCourse.ID, submissionNotClosedClassOfNotRegisteredCourse.ID, fileName, submissionData)
	if err == nil {
		return errSubmitAssignmentForNotRegisteredCourse
	}
	if err := verifyStatusCode(hres, []int{http.StatusBadRequest}); err != nil {
		return err
	}

	// 課題提出が締め切られた講義への課題提出
	hres, err = SubmitAssignmentAction(ctx, student.Agent, inProgressCourse.ID, submissionClosedClass.ID, fileName, submissionData)
	if err == nil {
		return errSubmitAssignmentForSubmissionClosedClass
	}
	if err := verifyStatusCode(hres, []int{http.StatusBadRequest}); err != nil {
		return err
	}

	// TODO: 不正な課題ファイルの提出で弾かれることのチェック
	// やるなら専用Action作らないといけないかも

	return nil
}

func (s *Scenario) prepareCheckPostGradeAbnormal(ctx context.Context) error {
	errPostGradeForUnknownClass := failure.NewError(fails.ErrApplication, fmt.Errorf("存在しない講義に対する成績登録に成功しました"))
	errPostGradeForSubmissionNotClosedClass := failure.NewError(fails.ErrApplication, fmt.Errorf("課題提出が締め切られていない講義に対する成績登録に成功しました"))

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
	courseParam := generate.CourseParam(1, 0, teacher)
	_, addCourseRes, err := AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	course := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity)

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
	hres, err := PostGradeAction(ctx, teacher.Agent, course.ID, uuid.NewRandom().String(), scores)
	if err == nil {
		return errPostGradeForUnknownClass
	}
	if err := verifyStatusCode(hres, []int{http.StatusBadRequest}); err != nil {
		return err
	}

	// 課題提出が締め切られていない講義の成績登録
	hres, err = PostGradeAction(ctx, teacher.Agent, course.ID, submissionNotClosedClass.ID, scores)
	if err == nil {
		return errPostGradeForSubmissionNotClosedClass
	}
	if err := verifyStatusCode(hres, []int{http.StatusBadRequest}); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) prepareCheckDownloadSubmissionsAbnormal(ctx context.Context) error {
	errDownloadSubmissionsForUnknownClass := failure.NewError(fails.ErrApplication, fmt.Errorf("存在しない講義の課題ダウンロードに成功しました"))

	// ======== 検証用データの準備 ========

	// 検証で使用する講師ユーザ
	teacher, err := s.getLoggedInTeacher(ctx)
	if err != nil {
		return err
	}

	// 適当な科目
	courseParam := generate.CourseParam(1, 0, teacher)
	_, addCourseRes, err := AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	course := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity)

	// ======== 検証 ========

	// 存在しない講義IDでの課題ダウンロード
	hres, _, err := DownloadSubmissionsAction(ctx, teacher.Agent, course.ID, uuid.NewRandom().String())
	if err == nil {
		return errDownloadSubmissionsForUnknownClass
	}
	if err := verifyStatusCode(hres, []int{http.StatusBadRequest}); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) prepareCheckSendAnnouncementAbnormal(ctx context.Context) error {
	errSendAnnouncementForUnknownCourse := failure.NewError(fails.ErrApplication, fmt.Errorf("存在しない科目のお知らせ追加に成功しました"))

	// ======== 検証用データの準備 ========

	// 検証で使用する講師ユーザ
	teacher, err := s.getLoggedInTeacher(ctx)
	if err != nil {
		return err
	}

	// 存在しない科目
	courseParam := generate.CourseParam(1, 0, teacher)
	notRegisteredCourse := model.NewCourse(courseParam, uuid.NewRandom().String(), teacher, prepareCourseCapacity)

	// 存在しない講義
	classParam := generate.ClassParam(notRegisteredCourse, 1)
	class := model.NewClass(uuid.NewRandom().String(), classParam)

	// ======== 検証 ========

	// 存在しない科目IDでのお知らせ追加
	announcement := generate.Announcement(notRegisteredCourse, class)
	hres, _, err := SendAnnouncementAction(ctx, teacher.Agent, announcement)
	if err == nil {
		return errSendAnnouncementForUnknownCourse
	}
	if err := verifyStatusCode(hres, []int{http.StatusNotFound}); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) prepareCheckGetAnnouncementDetailAbnormal(ctx context.Context) error {
	errGetClassesForUnknownCourse := failure.NewError(fails.ErrApplication, fmt.Errorf("存在しないお知らせの詳細取得に成功しました"))
	errGetClassesForNotRegisteredCourse := failure.NewError(fails.ErrApplication, fmt.Errorf("履修していない科目のお知らせ詳細取得に成功しました"))

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
	courseParam := generate.CourseParam(1, 0, teacher)
	_, addCourseRes, err := AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	notRegisteredCourse := model.NewCourse(courseParam, addCourseRes.ID, teacher, prepareCourseCapacity)

	// notRegisteredCourse の講義
	classParam := generate.ClassParam(notRegisteredCourse, 1)
	_, addClassRes, err := AddClassAction(ctx, teacher.Agent, notRegisteredCourse, classParam)
	if err != nil {
		return err
	}
	class := model.NewClass(addClassRes.ClassID, classParam)

	// notRegisteredCourse に紐づくお知らせ
	announcement := generate.Announcement(notRegisteredCourse, class)
	_, announcementRes, err := SendAnnouncementAction(ctx, teacher.Agent, announcement)
	if err != nil {
		return err
	}
	announcement.ID = announcementRes.ID

	// ======== 検証 ========

	// 存在しないお知らせIDでのお知らせ詳細取得
	hres, _, err := GetAnnouncementDetailAction(ctx, student.Agent, uuid.NewRandom().String())
	if err == nil {
		return errGetClassesForUnknownCourse
	}
	if err := verifyStatusCode(hres, []int{http.StatusNotFound}); err != nil {
		return err
	}

	// 履修していない科目に紐づくお知らせIDでのお知らせ詳細取得
	hres, _, err = GetAnnouncementDetailAction(ctx, student.Agent, announcement.ID)
	if err == nil {
		return errGetClassesForNotRegisteredCourse
	}
	if err := verifyStatusCode(hres, []int{http.StatusNotFound}); err != nil {
		return err
	}

	return nil
}

func (s *Scenario) getLoggedInStudent(ctx context.Context) (*model.Student, error) {
	userData, err := s.studentPool.newUserData()
	if err != nil {
		panic("unreachable! studentPool is empty")
	}
	student := model.NewStudent(userData, s.BaseURL)
	_, err = LoginAction(ctx, student.Agent, student.UserAccount)
	if err != nil {
		return nil, err
	}

	return student, nil
}

func (s *Scenario) getLoggedInTeacher(ctx context.Context) (*model.Teacher, error) {
	teacher := s.GetRandomTeacher()
	isLoggedIn := teacher.LoginOnce(func(teacher *model.Teacher) {
		_, err := LoginAction(ctx, teacher.Agent, teacher.UserAccount)
		if err != nil {
			return
		}
		teacher.IsLoggedIn = true
	})
	if !isLoggedIn {
		return nil, failure.NewError(fails.ErrApplication, fmt.Errorf("teacherのログインに失敗しました"))
	}

	return teacher, nil
}
