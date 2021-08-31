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
	"github.com/isucon/isucandar/worker"

	"github.com/isucon/isucon11-final/benchmarker/api"
	"github.com/isucon/isucon11-final/benchmarker/fails"
	"github.com/isucon/isucon11-final/benchmarker/generate"
	"github.com/isucon/isucon11-final/benchmarker/model"
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
		prepareCourseRegisterLimit = 20
		prepareClassCountPerCourse = 5
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
		students = append(students, model.NewStudent(userData, s.BaseURL, prepareCourseRegisterLimit))
	}

	courses := make([]*model.Course, 0, prepareCourseCount)
	mu := sync.Mutex{}
	// 教師のログインとコース登録をするワーカー
	w, err := worker.NewWorker(func(ctx context.Context, i int) {
		teacher := teachers[i%len(teachers)]
		_, err := LoginAction(ctx, teacher.Agent, teacher.UserAccount)
		if err != nil {
			AdminLogger.Printf("teacherのログインに失敗しました")
			step.AddError(failure.NewError(fails.ErrCritical, err))
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

		param := generate.CourseParam(teacher, generate.WithPeriod(i%6), generate.WithDayOfWeek((i/6)+1))
		_, res, err := AddCourseAction(ctx, teacher.Agent, param)
		if err != nil {
			step.AddError(err)
			return
		}
		course := model.NewCourse(param, res.ID, teacher)
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

		_, err = TakeCoursesAction(ctx, student.Agent, courses)
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

			err = validateUserGrade(&expected, &res, len(students))
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

type SampleDataForPrepareAbnormal struct {
	RegistrationCourse    *model.Course
	InProgressCourse      *model.Course
	ClosedCourse          *model.Course
	SubmissionClosedClass *model.Class
	SubmissionOpenClass   *model.Class
	Announcement          *model.Announcement
}

func (s *Scenario) prepareAbnormal(ctx context.Context, step *isucandar.BenchmarkStep) error {
	const (
		prepareCourseRegisterLimit = 20
	)

	// チェックで使用する学生ユーザ
	userDataForStudent, err := s.studentPool.newUserData()
	if err != nil {
		panic("unreachable! studentPool is empty")
	}
	student := model.NewStudent(userDataForStudent, s.BaseURL, prepareCourseRegisterLimit)
	_, err = LoginAction(ctx, student.Agent, student.UserAccount)
	if err != nil {
		return nil
	}

	// チェックで使用する講師ユーザ
	teacher := s.GetRandomTeacher()
	_, err = LoginAction(ctx, teacher.Agent, teacher.UserAccount)
	if err != nil {
		return nil
	}

	// チェックで使用するサンプルデータの生成
	sampleData := &SampleDataForPrepareAbnormal{}

	courseParam := generate.CourseParam(teacher)
	_, addCourseRes, err := AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	sampleData.RegistrationCourse = model.NewCourse(courseParam, addCourseRes.ID, teacher)

	courseParam = generate.CourseParam(teacher)
	_, addCourseRes, err = AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	sampleData.InProgressCourse = model.NewCourse(courseParam, addCourseRes.ID, teacher)
	_, err = SetCourseStatusInProgressAction(ctx, teacher.Agent, sampleData.InProgressCourse.ID)
	if err != nil {
		return err
	}

	courseParam = generate.CourseParam(teacher)
	_, addCourseRes, err = AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		return err
	}
	sampleData.ClosedCourse = model.NewCourse(courseParam, addCourseRes.ID, teacher)
	_, err = SetCourseStatusClosedAction(ctx, teacher.Agent, sampleData.ClosedCourse.ID)
	if err != nil {
		return err
	}

	classParam := generate.ClassParam(sampleData.InProgressCourse, 1)
	_, addClassRes, err := AddClassAction(ctx, teacher.Agent, sampleData.InProgressCourse, classParam)
	if err != nil {
		return err
	}
	sampleData.SubmissionClosedClass = model.NewClass(addClassRes.ClassID, classParam)
	_, _, err = DownloadSubmissionsAction(ctx, teacher.Agent, sampleData.InProgressCourse.ID, sampleData.SubmissionClosedClass.ID)
	if err != nil {
		return err
	}

	classParam = generate.ClassParam(sampleData.InProgressCourse, 2)
	_, addClassRes, err = AddClassAction(ctx, teacher.Agent, sampleData.InProgressCourse, classParam)
	if err != nil {
		return err
	}
	sampleData.SubmissionOpenClass = model.NewClass(addClassRes.ClassID, classParam)

	announcement := generate.Announcement(sampleData.InProgressCourse, sampleData.SubmissionOpenClass)
	_, announcementRes, err := SendAnnouncementAction(ctx, teacher.Agent, announcement)
	if err != nil {
		return err
	}
	announcement.ID = announcementRes.ID
	sampleData.Announcement = announcement

	// ======== 未ログイン状態で行う異常系チェック ========

	// 未ログイン状態でのチェックは別ユーザで行う
	userDataForUnloggedInCheck, err := s.studentPool.newUserData()
	if err != nil {
		panic("unreachable! studentPool is empty")
	}
	studentForUnloggedInCheck := model.NewStudent(userDataForUnloggedInCheck, s.BaseURL, prepareCourseRegisterLimit)
	teacherForUnloggedInCheck := s.GetRandomTeacher()

	// エンドポイントの認証チェック
	if err := checkAuthentication(ctx, studentForUnloggedInCheck, teacherForUnloggedInCheck, sampleData); err != nil {
		return err
	}

	// ログインの異常系チェック
	// 渡したユーザは副作用としてログインされるので、これ以降未ログイン状態のチェックには使えない
	if err := checkLoginAbnormal(ctx, studentForUnloggedInCheck); err != nil {
		return err
	}

	// ======== ログイン状態で行う異常系チェック ========

	// 講師用APIの認可チェック
	if err := checkAdminAuthorization(ctx, student, teacher, sampleData); err != nil {
		return err
	}

	return nil
}

func checkLoginAbnormal(ctx context.Context, student *model.Student) error {
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

func checkAuthentication(ctx context.Context, student *model.Student, teacher *model.Teacher, sampleData *SampleDataForPrepareAbnormal) error {
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

	hres, _, err := GetMeAction(ctx, student.Agent)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	hres, _, err = GetRegisteredCoursesAction(ctx, student.Agent)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	hres, err = TakeCoursesAction(ctx, student.Agent, []*model.Course{sampleData.RegistrationCourse})
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	hres, _, err = GetGradeAction(ctx, student.Agent)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	param := generate.SearchCourseParam()
	hres, _, err = SearchCourseAction(ctx, student.Agent, param, "")
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	hres, _, err = GetCourseDetailAction(ctx, student.Agent, sampleData.RegistrationCourse.ID)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	courseParam := generate.CourseParam(teacher)
	hres, _, err = AddCourseAction(ctx, teacher.Agent, courseParam)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	hres, err = SetCourseStatusInProgressAction(ctx, teacher.Agent, sampleData.RegistrationCourse.ID)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	hres, _, err = GetClassesAction(ctx, student.Agent, sampleData.InProgressCourse.ID)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	classParam := generate.ClassParam(sampleData.InProgressCourse, 3)
	hres, _, err = AddClassAction(ctx, teacher.Agent, sampleData.InProgressCourse, classParam)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	submissionData, fileName := generate.SubmissionData(sampleData.InProgressCourse, sampleData.SubmissionOpenClass, student.UserAccount)
	hres, err = SubmitAssignmentAction(ctx, student.Agent, sampleData.InProgressCourse.ID, sampleData.SubmissionOpenClass.ID, fileName, submissionData)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	scores := []StudentScore{
		{
			score: 90,
			code:  student.Code,
		},
	}
	hres, err = PostGradeAction(ctx, teacher.Agent, sampleData.InProgressCourse.ID, sampleData.SubmissionClosedClass.ID, scores)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	hres, _, err = DownloadSubmissionsAction(ctx, teacher.Agent, sampleData.InProgressCourse.ID, sampleData.SubmissionOpenClass.ID)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	hres, _, err = GetAnnouncementListAction(ctx, student.Agent, "")
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	announcement := generate.Announcement(sampleData.InProgressCourse, sampleData.SubmissionOpenClass)
	hres, _, err = SendAnnouncementAction(ctx, teacher.Agent, announcement)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	hres, _, err = GetAnnouncementDetailAction(ctx, student.Agent, sampleData.Announcement.ID)
	if err := checkAuthentication(hres, err); err != nil {
		return err
	}

	return nil
}

func checkAdminAuthorization(ctx context.Context, student *model.Student, teacher *model.Teacher, sampleData *SampleDataForPrepareAbnormal) error {
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

	courseParam := generate.CourseParam(teacher)
	hres, _, err := AddCourseAction(ctx, student.Agent, courseParam)
	if err := checkAuthorization(hres, err); err != nil {
		return err
	}

	hres, err = SetCourseStatusInProgressAction(ctx, student.Agent, sampleData.RegistrationCourse.ID)
	if err := checkAuthorization(hres, err); err != nil {
		return err
	}

	classParam := generate.ClassParam(sampleData.InProgressCourse, 3)
	hres, _, err = AddClassAction(ctx, student.Agent, sampleData.InProgressCourse, classParam)
	if err := checkAuthorization(hres, err); err != nil {
		return err
	}

	scores := []StudentScore{
		{
			score: 90,
			code:  student.Code,
		},
	}
	hres, err = PostGradeAction(ctx, student.Agent, sampleData.InProgressCourse.ID, sampleData.SubmissionClosedClass.ID, scores)
	if err := checkAuthorization(hres, err); err != nil {
		return err
	}

	hres, _, err = DownloadSubmissionsAction(ctx, student.Agent, sampleData.InProgressCourse.ID, sampleData.SubmissionOpenClass.ID)
	if err := checkAuthorization(hres, err); err != nil {
		return err
	}

	announcement := generate.Announcement(sampleData.InProgressCourse, sampleData.SubmissionOpenClass)
	hres, _, err = SendAnnouncementAction(ctx, student.Agent, announcement)
	if err := checkAuthorization(hres, err); err != nil {
		return err
	}

	return nil
}
