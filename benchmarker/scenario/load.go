package scenario

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucandar/parallel"

	"github.com/isucon/isucon11-final/benchmarker/fails"
	"github.com/isucon/isucon11-final/benchmarker/generate"
	"github.com/isucon/isucon11-final/benchmarker/model"
	"github.com/isucon/isucon11-final/benchmarker/score"
)

const (
	initialStudentsCount       = 50
	initialCourseCount         = 20
	registerCourseLimit        = 20
	searchCountPerRegistration = 3
	// classCountPerCourse は科目あたりのクラス数
	classCountPerCourse = 5
	// waitReadClassAnnouncementTimeout は学生がクラス課題のお知らせを確認するのを待つ最大時間
	waitReadClassAnnouncementTimeout = 5 * time.Second
)

func (s *Scenario) Load(parent context.Context, step *isucandar.BenchmarkStep) error {
	if s.NoLoad {
		return nil
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	ContestantLogger.Printf("===> LOAD")
	AdminLogger.Printf("LOAD INFO")

	// 負荷走行では
	// アクティブ学生による負荷と
	// 登録されたコースによる負荷が存在する
	studentLoadWorker := s.createStudentLoadWorker(ctx, step) // Gradeの確認から始まるシナリオとAnnouncementsの確認から始まるシナリオの二種類を担うgoroutineがアクティブ学生ごとに起動している
	courseLoadWorker := s.createLoadCourseWorker(ctx, step)   // 登録されたコースにつき一つのgoroutineが起動している
	// LoadWorkerに初期負荷を追加
	// (負荷追加はScenarioのPubSub経由で行われるので引数にLoadWorkerは不要)

	wg := sync.WaitGroup{}
	wg.Add(initialCourseCount)
	for i := 0; i < initialCourseCount; i++ {
		go func() {
			defer wg.Done()
			s.addCourseLoad(ctx, step)
		}()
	}
	wg.Wait()
	if len(s.courses) == 0 {
		step.AddError(failure.NewError(fails.ErrCritical, fmt.Errorf("コース登録が一つも成功しませんでした")))
		return nil
	}

	wg = sync.WaitGroup{}
	wg.Add(3)
	go func() {
		defer wg.Done()
		s.addActiveStudentLoads(ctx, step, initialStudentsCount)
	}()
	go func() {
		defer wg.Done()
		// LoadWorkerはisucandarのParallel
		studentLoadWorker.Do(func(ctx context.Context) {
			<-ctx.Done()
		})
		studentLoadWorker.Wait()
	}()
	go func() {
		defer wg.Done()
		courseLoadWorker.Do(func(ctx context.Context) {
			<-ctx.Done()
		})
		courseLoadWorker.Wait()
	}()
	wg.Wait()

	return nil
}

// アクティブ学生の負荷をかけ続けるLoadWorker(parallel.Parallel)を作成
func (s *Scenario) createStudentLoadWorker(ctx context.Context, step *isucandar.BenchmarkStep) *parallel.Parallel {
	// アクティブ学生は以下の2つのタスクを行い続ける
	// 「成績確認 + （空きがあれば履修登録）」
	// 「おしらせ確認 + （未読があれば詳細確認）」
	studentLoadWorker := parallel.NewParallel(ctx, -1)

	s.sPubSub.Subscribe(ctx, func(mes interface{}) {
		var student *model.Student
		var ok bool
		if student, ok = mes.(*model.Student); !ok {
			AdminLogger.Println("sPubSub に *model.Student以外が飛んできました")
			return
		}

		// 同時実行可能数を制限する際には注意
		// 成績確認 + (空きがあれば履修登録)
		AdminLogger.Println(student.Name, "の成績確認タスクが追加された") // FIXME: for debug
		studentLoadWorker.Do(registrationScenario(student, step, s))
		// おしらせ確認 + 既読追加
		AdminLogger.Println(student.Name, "のおしらせタスクが追加された") // FIXME: for debug
		studentLoadWorker.Do(readAnnouncementScenario(student, step))
	})
	return studentLoadWorker
}

func registrationScenario(student *model.Student, step *isucandar.BenchmarkStep, s *Scenario) func(ctx context.Context) {
	return func(ctx context.Context) {
		for ctx.Err() == nil {

			// 学生は成績を確認し続ける
			_, getGradeRes, err := GetGradeAction(ctx, student.Agent)
			if err != nil {
				step.AddError(err)
				<-time.After(3000 * time.Millisecond)
				continue
			}
			if err := verifyGrades(&getGradeRes); err != nil {
				step.AddError(err)
			} else {
				step.AddScore(score.CountGetGrades)
			}

			AdminLogger.Printf("%vは成績を確認した", student.Name)

			select {
			case <-ctx.Done():
				return
			default:
			}

			// ----------------------------------------

			// Gradeが早くなった時、常にCapacityが0だとGradeを効率的に回せるようになって点数が高くなるという不正ができるかもしれない
			remainingRegistrationCapacity := registerCourseLimit - student.RegisteringCount()
			if remainingRegistrationCapacity == 0 {
				continue
			}

			// remainingRegistrationCapacity * searchCountPerRegistration 回 検索を行う
			// remainingRegistrationCapacity 分のシラバス確認を行う
			for i := 0; i < remainingRegistrationCapacity; i++ {
				var checkTargetID string
				// 履修希望コース1つあたり searchCountPerRegistration 回のコース検索を行う
				for searchCount := 0; searchCount < searchCountPerRegistration; searchCount++ {
					param := generate.SearchCourseParam()
					_, res, err := SearchCourseAction(ctx, student.Agent, param)
					if err != nil {
						step.AddError(err)
						continue
					}
					errs := verifySearchCourseResults(res, param)
					for _, err := range errs {
						step.AddError(err)
						continue
					}
					step.AddScore(score.CountSearchCourse)

					if len(res) > 0 {
						checkTargetID = res[0].ID.String()
					}

					select {
					case <-ctx.Done():
						return
					default:
					}
				}

				// 検索で得たコースのシラバスを確認する
				if checkTargetID == "" {
					continue
				}

				_, res, err := GetCourseDetailAction(ctx, student.Agent, checkTargetID)
				if err != nil {
					step.AddError(err)
					continue
				}
				expected, exists := s.GetCourse(res.ID.String())
				// ベンチ側の登録がまだの場合は検証スキップ
				if exists {
					if err := verifyCourseDetail(&res, expected); err != nil {
						step.AddError(err)
					}
				}

				select {
				case <-ctx.Done():
					return
				default:
				}
			}

			AdminLogger.Printf("%vはコースを%v回検索した", student.Name, remainingRegistrationCapacity*searchCountPerRegistration)

			// ----------------------------------------

			registeredSchedule := student.RegisteredSchedule()
			_, getRegisteredCoursesRes, err := GetRegisteredCoursesAction(ctx, student.Agent)
			if err != nil {
				step.AddError(err)
				<-time.After(3000 * time.Millisecond)
				continue
			}
			if err := verifyRegisteredCourses(getRegisteredCoursesRes, registeredSchedule); err != nil {
				step.AddError(err)
			}

			select {
			case <-ctx.Done():
				return
			default:
			}

			// ----------------------------------------

			// 仮登録(ベンチ内部では登録済みにする)
			// TODO: 1度も検索成功してなかったら登録しない
			semiRegistered := make([]*model.Course, 0, remainingRegistrationCapacity)

			randTimeSlots := generate.ShuffledInts(30) // 平日分のコマ 5*6

			studentScheduleMutex := student.ScheduleMutex()
			studentScheduleMutex.Lock()
			for _, timeSlot := range randTimeSlots {
				// 仮登録数が追加履修可能数を超えていたら抜ける
				if len(semiRegistered) >= remainingRegistrationCapacity {
					break
				}

				dayOfWeek := timeSlot/6 + 1 // 日曜日分+1
				period := timeSlot % 6

				if !student.IsEmptyTimeSlots(dayOfWeek, period) {
					continue
				}

				// コースへの配分はCourseManagerが担うので、学生のワーカーは学生の空いているtimeslotを決めるところまで行えば良い
				registeredCourse := s.emptyCourseManager.AddStudentForRegistrableCourse(student, dayOfWeek, period)
				// 該当コマで履修可能なコースがなかった
				if registeredCourse == nil {
					continue
				}

				student.FillTimeslot(registeredCourse)
				semiRegistered = append(semiRegistered, registeredCourse)
			}
			studentScheduleMutex.Unlock()

			select {
			case <-ctx.Done():
				return
			default:
			}

			// ----------------------------------------

			// ベンチ内で仮登録できたコースがあればAPIに登録処理を投げる
			if len(semiRegistered) > 0 {
				_, err := TakeCoursesAction(ctx, student.Agent, semiRegistered)
				if err != nil {
					step.AddError(err)
					// 失敗時の仮登録情報のロールバック
					for _, c := range semiRegistered {
						c.FinishRegistration()
						c.RemoveStudent(student)
						student.ReleaseTimeslot(c.DayOfWeek, c.Period)
					}
				} else {
					step.AddScore(score.CountRegisterCourses)
					for _, c := range semiRegistered {
						c.FinishRegistration()
						c.SetClosingAfterSecAtOnce(5 * time.Second) // 初履修者からn秒後に履修を締め切る
						student.AddCourse(c)
						AdminLogger.Printf("%vは%vを履修した", student.Name, c.Name)
					}
				}
			}
			// TODO: できれば登録に失敗したコースを抜いて再度登録する

			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}
}

func readAnnouncementScenario(student *model.Student, step *isucandar.BenchmarkStep) func(ctx context.Context) {
	return func(ctx context.Context) {
		var next string // 次にアクセスするお知らせ一覧のページ
		for ctx.Err() == nil {

			// BrowserAccess(announce)
			// resource Verify

			// 学生はお知らせを確認し続ける
			hres, res, err := GetAnnouncementListAction(ctx, student.Agent, next)
			if err != nil {
				step.AddError(err)
				<-time.After(3000 * time.Millisecond)
				continue
			}
			errs := verifyAnnouncements(&res, student)
			for _, err := range errs {
				step.AddError(err)
			}
			if len(errs) == 0 {
				step.AddScore(score.CountGetAnnouncements)
			}

			AdminLogger.Printf("%vはお知らせ一覧を確認した", student.Name)

			for _, ans := range res.Announcements {
				select {
				case <-ctx.Done():
					return
				default:
				}

				if ans.Unread {
					announcementStatus := student.GetAnnouncement(ans.ID)
					if announcementStatus == nil {
						// webappでは認識されているが、ベンチではまだ認識されていないお知らせ
						// load中には検証できないのでskip
						continue
					}

					// お知らせの詳細を取得する
					_, res, err := GetAnnouncementDetailAction(ctx, student.Agent, ans.ID)
					if err != nil {
						step.AddError(err)
						continue // 次の未読おしらせの確認へ
					}

					if err := verifyAnnouncement(&res, announcementStatus); err != nil {
						step.AddError(err)
					} else {
						step.AddScore(score.CountGetAnnouncementsDetail)
					}

					student.ReadAnnouncement(ans.ID)
					AdminLogger.Printf("%vはお知らせ詳細を確認した", student.Name)
				}
			}

			_, next = parseLinkHeader(hres)
			// TODO: 現状: ページングで最後のページまで確認したら最初のページに戻る
			// TODO: 理想1: 未読お知らせを早く確認するため以降のページに未読が存在しないなら最初に戻る
			// TODO: 理想2: 10ページぐらい最低ページングする。10ページ目末尾のお知らせ以降に未読があればさらにページングする。無いならしない。
			// MEMO: Student.Announcementsはwebapp内のお知らせの順番(createdAt)と完全同期できていない
			// MEMO: 理想1,2を実現するためにはStudent.AnnouncementsをcreatedAtで保持する必要がある。insertできる木構造では持つのは辛いのでやりたくない。
			// ※ webappに追加するAnnouncementのcreatedAtはベンチ側が指定する

			select {
			case <-ctx.Done():
				return
			case <-time.After(1000 * time.Millisecond):
			}
		}
	}
}

func (s *Scenario) createLoadCourseWorker(ctx context.Context, step *isucandar.BenchmarkStep) *parallel.Parallel {
	// 追加されたコースの動作を回し続けるParallel
	loadCourseWorker := parallel.NewParallel(ctx, -1)
	s.cPubSub.Subscribe(ctx, func(mes interface{}) {
		var course *model.Course
		var ok bool
		if course, ok = mes.(*model.Course); !ok {
			AdminLogger.Println("cPubSub に *model.Course以外が飛んできました")
			return
		}

		AdminLogger.Println(course.Name, "のタスクが追加された") // FIXME: for debug
		loadCourseWorker.Do(courseScenario(course, step, s))
	})
	return loadCourseWorker
}

func courseScenario(course *model.Course, step *isucandar.BenchmarkStep, s *Scenario) func(ctx context.Context) {
	return func(ctx context.Context) {
		defer func() {
			for _, student := range course.Students() {
				student.ReleaseTimeslot(course.DayOfWeek, course.Period)
			}
		}()

		// コースgoroutineは満員 or 履修締め切りまではなにもしない
		<-course.WaitPreparedCourse(ctx)

		select {
		case <-ctx.Done():
			return
		default:
		}

		select {
		case <-ctx.Done():
			return
		default:
		}

		teacher := course.Teacher()
		// コースステータスをin-progressにする
		_, err := SetCourseStatusInProgressAction(ctx, teacher.Agent, course.ID)
		if err != nil {
			step.AddError(err)
			AdminLogger.Printf("%vのコースステータスをin-progressに変更するのが失敗しました", course.Name)
			return
		}
		AdminLogger.Printf("%vが開始した", course.Name) // FIXME: for debug

		select {
		case <-ctx.Done():
			return
		default:
		}

		// コースの処理
		for i := 0; i < classCountPerCourse; i++ {
			timer := time.After(100 * time.Millisecond)

			classParam := generate.ClassParam(course, uint8(i+1))
			_, class, announcement, err := AddClassAction(ctx, teacher.Agent, course, classParam)
			if err != nil {
				step.AddError(err)
				<-timer
				continue
			} else {
				step.AddScore(score.CountAddClass)
			}
			course.AddClass(class)
			course.BroadCastAnnouncement(announcement)
			AdminLogger.Printf("%vの第%v回講義が追加された", course.Name, i+1) // FIXME: for debug

			select {
			case <-ctx.Done():
				return
			default:
			}

			errs := submitAssignments(ctx, course.Students(), course, class, announcement.ID, step)
			for _, e := range errs {
				step.AddError(e)
			}
			AdminLogger.Printf("%vの第%v回講義の課題提出が完了した", course.Name, i+1) // FIXME: for debug

			select {
			case <-ctx.Done():
				return
			default:
			}

			_, assignmentsData, err := DownloadSubmissionsAction(ctx, teacher.Agent, course.ID, class.ID)
			if err != nil {
				step.AddError(err)
				continue
			}
			if err := verifyAssignments(assignmentsData, class); err != nil {
				step.AddError(err)
			}
			AdminLogger.Printf("%vの第%v回講義の課題DLが完了した", course.Name, i+1) // FIXME: for debug

			select {
			case <-ctx.Done():
				return
			default:
			}

			// TODO: 採点する
			_, err = scoringAssignments(ctx, course.ID, class.ID, teacher, course.Students(), assignmentsData)
			if err != nil {
				step.AddError(err)
				<-timer
				continue
			} else {
				step.AddScore(score.CountRegisterScore)
			}
			AdminLogger.Printf("%vの第%v回講義の採点が完了した", course.Name, i+1) // FIXME: for debug

			select {
			case <-ctx.Done():
				return
			case <-timer:
			}
		}

		// コースステータスをclosedにする
		_, err = SetCourseStatusClosedAction(ctx, teacher.Agent, course.ID)
		if err != nil {
			step.AddError(err)
			AdminLogger.Printf("%vのコースステータスをclosedに変更するのが失敗しました", course.Name)
			return
		}

		AdminLogger.Printf("%vが終了した", course.Name) // FIXME: for debug

		// FIXME: Debug
		{
			s.mu.Lock()
			s.finishedCourseCount++
			s.mu.Unlock()
		}

		// コースを追加
		s.addCourseLoad(ctx, step)
		s.addCourseLoad(ctx, step)

		// コースが追加されたのでベンチのアクティブ学生も増やす
		s.addActiveStudentLoads(ctx, step, 1)
	}
}

func (s *Scenario) addActiveStudentLoads(ctx context.Context, step *isucandar.BenchmarkStep, count int) {
	wg := sync.WaitGroup{}
	wg.Add(count)
	for i := 0; i < count; i++ {
		go func() {
			defer wg.Done()
			userData, err := s.studentPool.newUserData()
			if err != nil {
				return
			}
			student := model.NewStudent(userData, s.BaseURL, registerCourseLimit)

			hres, resources, err := AccessTopPageAction(ctx, student.Agent)
			if err != nil {
				AdminLogger.Printf("学生 %vがログイン画面にアクセスできませんでした", userData.Name)
				step.AddError(err)
				return
			}
			errs := verifyPageResource(hres, resources)
			if len(errs) != 0 {
				AdminLogger.Printf("学生 %vがアクセスしたログイン画面の検証に失敗しました", userData.Name)
				for _, err := range errs {
					step.AddError(err)
				}
				return
			}

			_, err = LoginAction(ctx, student.Agent, student.UserAccount)
			if err != nil {
				ContestantLogger.Printf("学生 %vのログインが失敗しました", userData.Name)
				step.AddError(err)
				return
			}

			select {
			case <-ctx.Done():
				return
			default:
			}

			_, res, err := GetMeAction(ctx, student.Agent)
			if err != nil {
				ContestantLogger.Printf("学生 %vのユーザ情報取得に失敗しました", userData.Name)
				step.AddError(err)
				return
			}
			if err := verifyMe(&res, userData, false); err != nil {
				step.AddError(err)
				return
			}

			s.AddActiveStudent(student)
			s.sPubSub.Publish(student)
		}()
	}
	wg.Wait()
}

func (s *Scenario) addCourseLoad(ctx context.Context, step *isucandar.BenchmarkStep) {
	teacher := s.GetRandomTeacher()
	courseParam := generate.CourseParam(teacher)

	_, err := LoginAction(ctx, teacher.Agent, teacher.UserAccount)
	if err != nil {
		ContestantLogger.Printf("teacherのログインに失敗しました")
		step.AddError(failure.NewError(fails.ErrCritical, err))
		return
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	_, getMeRes, err := GetMeAction(ctx, teacher.Agent)
	if err != nil {
		ContestantLogger.Printf("teacherのユーザ情報取得に失敗しました")
		step.AddError(err)
		return
	}
	if err := verifyMe(&getMeRes, teacher.UserAccount, true); err != nil {
		step.AddError(err)
		return
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	_, addCourseRes, err := AddCourseAction(ctx, teacher, courseParam)
	if err != nil {
		step.AddError(err)
		return
	} else {
		step.AddScore(score.CountAddCourse)
	}

	course := model.NewCourse(courseParam, addCourseRes.ID, teacher)
	s.AddCourse(course)
	s.emptyCourseManager.AddEmptyCourse(course)
	s.cPubSub.Publish(course)
}

func submitAssignments(ctx context.Context, students []*model.Student, course *model.Course, class *model.Class, announcementID string, step *isucandar.BenchmarkStep) []error {
	wg := sync.WaitGroup{}
	wg.Add(len(students))

	mu := sync.Mutex{}
	errs := make([]error, 0)
	for _, s := range students {
		s := s
		go func() {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			case <-time.After(waitReadClassAnnouncementTimeout):
				AdminLogger.Printf("学生が%d秒以内に課題のお知らせを確認できなかったため課題を提出しませんでした", waitReadClassAnnouncementTimeout/time.Second)
				return
			case <-s.WaitReadAnnouncement(announcementID):
				// 学生sが課題お知らせを読むまで待つ
			}

			// 講義一覧を取得する
			_, res, err := GetClassesAction(ctx, s.Agent, course.ID)
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}
			if err := verifyClasses(res, course.Classes()); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}

			select {
			case <-ctx.Done():
				return
			default:
			}

			// 課題を提出する
			submission := generate.Submission(course, class, s.UserAccount)
			_, err = SubmitAssignmentAction(ctx, s.Agent, course.ID, class.ID, submission)
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			} else {
				step.AddScore(score.CountSubmitAssignment)
				class.AddSubmittedAssignment(s.Code, submission.Data)
				s.AddSubmission(submission)
			}
		}()
	}
	wg.Wait()

	return errs
}

// これここじゃないほうがいいかも知れない
type StudentScore struct {
	score int
	code  string
}

func scoringAssignments(ctx context.Context, courseID, classID string, teacher *model.Teacher, students []*model.Student, assignments []byte) (*http.Response, error) {
	wg := sync.WaitGroup{}
	wg.Add(len(students))

	scores := make([]StudentScore, 0, len(students))
	for _, s := range students {
		score := rand.Intn(101)
		scores = append(scores, StudentScore{
			score: score,
			code:  s.Code,
		})
	}
	res, err := PostGradeAction(ctx, teacher.Agent, courseID, classID, scores)
	if err != nil {
		return nil, err
	}

	return res, nil
}
