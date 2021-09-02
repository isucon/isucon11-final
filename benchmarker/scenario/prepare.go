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

type prepareAbnormalScenario struct {
	Student *model.Student
	Teacher *model.Teacher
}

func (s *Scenario) newPrepareAbnormalScenario(student *model.Student, teacher *model.Teacher) *prepareAbnormalScenario {
	return &prepareAbnormalScenario{
		Student: student,
		Teacher: teacher,
	}
}

func (s *Scenario) prepareAbnormal(ctx context.Context, step *isucandar.BenchmarkStep) error {
	// チェックで使用する学生ユーザ
	userData, err := s.studentPool.newUserData()
	if err != nil {
		panic("unreachable! studentPool is empty")
	}
	student := model.NewStudent(userData, s.BaseURL)
	_, err = LoginAction(ctx, student.Agent, student.UserAccount)
	if err != nil {
		return err
	}

	// チェックで使用する講師ユーザ
	teacher := s.GetRandomTeacher()
	isLoggedIn := teacher.LoginOnce(func(teacher *model.Teacher) {
		_, err := LoginAction(ctx, teacher.Agent, teacher.UserAccount)
		if err != nil {
			return
		}
		teacher.IsLoggedIn = true
	})
	if !isLoggedIn {
		return failure.NewError(fails.ErrApplication, fmt.Errorf("teacherのログインに失敗しました"))
	}

	pas := s.newPrepareAbnormalScenario(student, teacher)

	// ======== 未ログイン状態で行う異常系チェック ========

	agent, _ := agent.NewAgent(
		agent.WithUserAgent(useragent.UserAgent()),
		agent.WithBaseURL(s.BaseURL.String()),
	)

	// 認証チェック
	if err := pas.prepareCheckAuthenticationAbnormal(ctx, agent); err != nil {
		return err
	}

	// ログインの異常系チェック用ユーザ
	userData, err = s.studentPool.newUserData()
	if err != nil {
		panic("unreachable! studentPool is empty")
	}
	studentForCheckLoginAbnormal := model.NewStudent(userData, s.BaseURL)

	// ログインの異常系チェック
	// 渡したユーザは副作用としてログインされる
	if err := pas.prepareCheckLoginAbnormal(ctx, studentForCheckLoginAbnormal); err != nil {
		return err
	}

	// ======== ログイン状態で行う異常系チェック ========

	// 講師用APIの認可チェック
	if err := pas.prepareCheckAdminAuthorizationAbnormal(ctx); err != nil {
		return err
	}

	if err := pas.prepareCheckRegisterCoursesAbnormal(ctx); err != nil {
		return err
	}

	if err := pas.prepareCheckGetCourseDetailAbnormal(ctx); err != nil {
		return err
	}

	return nil
}

func (pas *prepareAbnormalScenario) prepareCheckLoginAbnormal(ctx context.Context, student *model.Student) error {
	errInvalidAuthenticationLogin := failure.NewError(fails.ErrApplication, fmt.Errorf("間違った認証情報でのログインに成功しました"))
	errRelogin := failure.NewError(fails.ErrApplication, fmt.Errorf("ログイン状態での再ログインに成功しました"))

	// 存在しないユーザでのログイン
	hres, err := LoginAction(ctx, student.Agent, &model.UserAccount{
		Code:        "X12345",
		Name:        "unknown",
		RawPassword: "password",
	})
	if err == nil {
		return errInvalidAuthenticationLogin
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
		return errInvalidAuthenticationLogin
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

func (pas *prepareAbnormalScenario) prepareCheckAuthenticationAbnormal(ctx context.Context, agent *agent.Agent) error {
	const (
		prepareCourseCapacity = 50
	)

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

	// ======== サンプルデータの生成 ========

	// 適当な科目
	courseParam := generate.CourseParam(1, 0, pas.Teacher)
	_, addCourseRes, err := AddCourseAction(ctx, pas.Teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	course := model.NewCourse(courseParam, addCourseRes.ID, pas.Teacher, prepareCourseCapacity)

	// 課題提出が締め切られた講義
	classParam := generate.ClassParam(course, 1)
	_, addClassRes, err := AddClassAction(ctx, pas.Teacher.Agent, course, classParam)
	if err != nil {
		return err
	}
	submissionClosedClass := model.NewClass(addClassRes.ClassID, classParam)
	_, _, err = DownloadSubmissionsAction(ctx, pas.Teacher.Agent, course.ID, submissionClosedClass.ID)
	if err != nil {
		return err
	}

	// 課題提出が締め切られていない講義
	classParam = generate.ClassParam(course, 2)
	_, addClassRes, err = AddClassAction(ctx, pas.Teacher.Agent, course, classParam)
	if err != nil {
		return err
	}
	submissionOpenClass := model.NewClass(addClassRes.ClassID, classParam)

	// course に紐づくお知らせ
	announcement1 := generate.Announcement(course, submissionOpenClass)
	_, announcementRes, err := SendAnnouncementAction(ctx, pas.Teacher.Agent, announcement1)
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

	courseParam = generate.CourseParam(2, 0, pas.Teacher)
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

	submissionData, fileName := generate.SubmissionData(course, submissionOpenClass, pas.Student.UserAccount)
	hres, err = SubmitAssignmentAction(ctx, agent, course.ID, submissionOpenClass.ID, fileName, submissionData)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	scores := []StudentScore{
		{
			score: 90,
			code:  pas.Student.Code,
		},
	}
	hres, err = PostGradeAction(ctx, agent, course.ID, submissionClosedClass.ID, scores)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	hres, _, err = DownloadSubmissionsAction(ctx, agent, course.ID, submissionOpenClass.ID)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	hres, _, err = GetAnnouncementListAction(ctx, agent, "")
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	announcement2 := generate.Announcement(course, submissionOpenClass)
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

func (pas *prepareAbnormalScenario) prepareCheckAdminAuthorizationAbnormal(ctx context.Context) error {
	const (
		prepareCourseCapacity = 50
	)

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

	// ======== サンプルデータの生成 ========

	// 適当な科目
	courseParam := generate.CourseParam(1, 0, pas.Teacher)
	_, addCourseRes, err := AddCourseAction(ctx, pas.Teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	course := model.NewCourse(courseParam, addCourseRes.ID, pas.Teacher, prepareCourseCapacity)

	// 課題提出が締め切られた講義
	classParam := generate.ClassParam(course, 1)
	_, addClassRes, err := AddClassAction(ctx, pas.Teacher.Agent, course, classParam)
	if err != nil {
		return err
	}
	submissionClosedClass := model.NewClass(addClassRes.ClassID, classParam)
	_, _, err = DownloadSubmissionsAction(ctx, pas.Teacher.Agent, course.ID, submissionClosedClass.ID)
	if err != nil {
		return err
	}

	// 課題提出が締め切られていない講義
	classParam = generate.ClassParam(course, 2)
	_, addClassRes, err = AddClassAction(ctx, pas.Teacher.Agent, course, classParam)
	if err != nil {
		return err
	}
	submissionOpenClass := model.NewClass(addClassRes.ClassID, classParam)

	// ======== 検証 ========

	courseParam = generate.CourseParam(2, 0, pas.Teacher)
	hres, _, err := AddCourseAction(ctx, pas.Student.Agent, courseParam)
	if err := checkAuthorization(hres, err); err != nil {
		return err
	}

	hres, err = SetCourseStatusInProgressAction(ctx, pas.Student.Agent, course.ID)
	if err := checkAuthorization(hres, err); err != nil {
		return err
	}

	classParam = generate.ClassParam(course, 3)
	hres, _, err = AddClassAction(ctx, pas.Student.Agent, course, classParam)
	if err := checkAuthorization(hres, err); err != nil {
		return err
	}

	scores := []StudentScore{
		{
			score: 90,
			code:  pas.Student.Code,
		},
	}
	hres, err = PostGradeAction(ctx, pas.Student.Agent, course.ID, submissionClosedClass.ID, scores)
	if err := checkAuthorization(hres, err); err != nil {
		return err
	}

	hres, _, err = DownloadSubmissionsAction(ctx, pas.Student.Agent, course.ID, submissionOpenClass.ID)
	if err := checkAuthorization(hres, err); err != nil {
		return err
	}

	announcement := generate.Announcement(course, submissionOpenClass)
	hres, _, err = SendAnnouncementAction(ctx, pas.Student.Agent, announcement)
	if err := checkAuthorization(hres, err); err != nil {
		return err
	}

	return nil
}

func (pas *prepareAbnormalScenario) prepareCheckRegisterCoursesAbnormal(ctx context.Context) error {
	const (
		prepareCourseCapacity = 50
	)

	errInvalidCourseRegistration := failure.NewError(fails.ErrApplication, fmt.Errorf("履修登録できないはずの科目の履修に成功しました"))
	errInvalidErrorResponce := errInvalidResponse("履修登録失敗時のレスポンスが期待する内容と一致しません")

	// ======== サンプルデータの生成 ========

	// ステータスが registration の科目
	courseParam := generate.CourseParam(1, 0, pas.Teacher)
	_, addCourseRes, err := AddCourseAction(ctx, pas.Teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	registrationCourse := model.NewCourse(courseParam, addCourseRes.ID, pas.Teacher, prepareCourseCapacity)

	// ステータスが in-progress の科目
	courseParam = generate.CourseParam(1, 1, pas.Teacher)
	_, addCourseRes, err = AddCourseAction(ctx, pas.Teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	inProgressCourse := model.NewCourse(courseParam, addCourseRes.ID, pas.Teacher, prepareCourseCapacity)
	_, err = SetCourseStatusInProgressAction(ctx, pas.Teacher.Agent, inProgressCourse.ID)
	if err != nil {
		return err
	}

	// ステータスが closed の科目
	courseParam = generate.CourseParam(1, 2, pas.Teacher)
	_, addCourseRes, err = AddCourseAction(ctx, pas.Teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	closedCourse := model.NewCourse(courseParam, addCourseRes.ID, pas.Teacher, prepareCourseCapacity)
	_, err = SetCourseStatusClosedAction(ctx, pas.Teacher.Agent, closedCourse.ID)
	if err != nil {
		return err
	}

	// pas.Student が履修登録済みの科目
	courseParam = generate.CourseParam(2, 0, pas.Teacher)
	_, addCourseRes, err = AddCourseAction(ctx, pas.Teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	alreadyRegisteredCourse := model.NewCourse(courseParam, addCourseRes.ID, pas.Teacher, prepareCourseCapacity)
	_, _, err = TakeCoursesAction(ctx, pas.Student.Agent, []*model.Course{alreadyRegisteredCourse})
	if err != nil {
		return err
	}

	// alreadyRegisteredCourse と時間割がコンフリクトする科目
	courseParam = generate.CourseParam(2, 0, pas.Teacher)
	_, addCourseRes, err = AddCourseAction(ctx, pas.Teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	conflictedCourse1 := model.NewCourse(courseParam, addCourseRes.ID, pas.Teacher, prepareCourseCapacity)

	// 時間割がコンフリクトする2つの科目
	courseParam = generate.CourseParam(2, 1, pas.Teacher)
	_, addCourseRes, err = AddCourseAction(ctx, pas.Teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	conflictedCourse2 := model.NewCourse(courseParam, addCourseRes.ID, pas.Teacher, prepareCourseCapacity)

	courseParam = generate.CourseParam(2, 1, pas.Teacher)
	_, addCourseRes, err = AddCourseAction(ctx, pas.Teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	conflictedCourse3 := model.NewCourse(courseParam, addCourseRes.ID, pas.Teacher, prepareCourseCapacity)

	// 存在しない科目
	courseParam = generate.CourseParam(3, 0, pas.Teacher)
	unknownCourse := model.NewCourse(courseParam, uuid.NewRandom().String(), pas.Teacher, prepareCourseCapacity)

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
	hres, eres, err := TakeCoursesAction(ctx, pas.Student.Agent, courses)
	if err == nil {
		return errInvalidCourseRegistration
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

func (pas *prepareAbnormalScenario) prepareCheckGetCourseDetailAbnormal(ctx context.Context) error {
	errGetUnknownCourseDetail := failure.NewError(fails.ErrApplication, fmt.Errorf("存在しない科目の詳細取得に成功しました"))

	// 存在しない科目IDでの科目詳細取得
	hres, _, err := GetCourseDetailAction(ctx, pas.Student.Agent, uuid.NewRandom().String())
	if err == nil {
		return errGetUnknownCourseDetail
	}
	if err := verifyStatusCode(hres, []int{http.StatusNotFound}); err != nil {
		return err
	}

	return nil
}
