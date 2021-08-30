package scenario

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
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
	// loadRequestTime はLoadシナリオ内でリクエストを送り続ける時間(Load自体のTimeoutより早めに終わらせる)
	loadRequestTime = 60 * time.Second
)

func (s *Scenario) Load(parent context.Context, step *isucandar.BenchmarkStep) error {
	if s.NoLoad {
		return nil
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	ContestantLogger.Printf("===> LOAD")
	AdminLogger.Printf("LOAD INFO")

	s.loadRequestEndTime = time.Now().Add(loadRequestTime)

	// 負荷走行では
	// アクティブ学生による負荷と
	// 登録されたコースによる負荷が存在する
	studentLoadWorker := s.createStudentLoadWorker(ctx, step) // Gradeの確認から始まるシナリオとAnnouncementsの確認から始まるシナリオの二種類を担うgoroutineがアクティブ学生ごとに起動している
	courseLoadWorker := s.createLoadCourseWorker(ctx, step)   // 登録されたコースにつき一つのgoroutineが起動している

	// LoadWorkerに初期負荷を追加
	// (負荷追加はScenarioのPubSub経由で行われるので引数にLoadWorkerは不要)
	wg := sync.WaitGroup{}
	wg.Add(initialCourseCount + 1)
	for i := 0; i < initialCourseCount; i++ {
		go func() {
			defer DebugLogger.Printf("[debug] initial Courses added")
			defer wg.Done()
			s.addCourseLoad(ctx, step)
		}()
	}
	go func() {
		defer DebugLogger.Printf("[debug] initial ActiveStudents added")
		defer wg.Done()
		s.addActiveStudentLoads(ctx, step, initialStudentsCount)
	}()
	wg.Wait()

	if len(s.courses) == 0 {
		step.AddError(failure.NewError(fails.ErrCritical, fmt.Errorf("コース登録が1つも成功しませんでした")))
		return nil
	}
	if s.ActiveStudentCount() == 0 {
		step.AddError(failure.NewError(fails.ErrCritical, fmt.Errorf("ログインに成功した学生が1人もいませんでした")))
		return nil
	}

	wg = sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer DebugLogger.Printf("[debug] studentLoadWorker finished")
		defer wg.Done()
		studentLoadWorker.Wait()
	}()
	go func() {
		defer DebugLogger.Printf("[debug] courseLoadWorker finished")
		defer wg.Done()
		courseLoadWorker.Wait()
	}()
	wg.Wait()

	// loadRequestTimeが終了しても最後に送ったリクエストの処理が終わるまで（loadTimeoutまで）待つ
	<-ctx.Done()
	AdminLogger.Printf("[debug] load finished")

	DebugLogger.Printf("========STATS_DATA=========")
	for k, v := range s.debugData.ints {
		var sum int64
		for _, t := range v {
			sum += t
		}
		avg := int64(float64(sum) / float64(len(v)))
		DebugLogger.Printf("%s: avg %d", k, avg)
	}
	s.debugData.Close()

	return nil
}

// isNoRequestTime はリクエスト送信できない期間かどうか（各Actionの前に必ず調べる）
func (s *Scenario) isNoRequestTime(ctx context.Context) bool {
	return time.Now().After(s.loadRequestEndTime) || ctx.Err() != nil
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
		studentLoadWorker.Do(s.registrationScenario(student, step))
		// おしらせ確認 + 既読追加
		studentLoadWorker.Do(s.readAnnouncementScenario(student, step))
	})
	return studentLoadWorker
}

func (s *Scenario) registrationScenario(student *model.Student, step *isucandar.BenchmarkStep) func(ctx context.Context) {
	return func(ctx context.Context) {
		for ctx.Err() == nil {

			if s.isNoRequestTime(ctx) {
				return
			}

			// 学生は成績を確認し続ける
			expected := collectVerifyGradesData(student)
			_, getGradeRes, err := GetGradeAction(ctx, student.Agent)
			if err != nil {
				step.AddError(err)
				<-time.After(1 * time.Millisecond)
				continue
			}
			err = verifyGrades(expected, &getGradeRes)
			if err != nil {
				step.AddError(err)
			} else {
				step.AddScore(score.GetGrades)
			}

			// ----------------------------------------

			// Gradeが早くなった時、常にCapacityが0だとGradeを効率的に回せるようになって点数が高くなるという不正ができるかもしれない
			remainingRegistrationCapacity := registerCourseLimit - student.RegisteringCount()
			if remainingRegistrationCapacity == 0 {
				// DebugLogger.Printf("[履修スキップ（空きコマ不足)] code: %v, name: %v", student.Code, student.Name)
				continue
			}

			registerStart := time.Now()

			// remainingRegistrationCapacity * searchCountPerRegistration 回 検索を行う
			// remainingRegistrationCapacity 分のシラバス確認を行う
			for i := 0; i < remainingRegistrationCapacity; i++ {
				var checkTargetID string
				var nextPathParam string // 次にアクセスする検索一覧のページ
				// 履修希望コース1つあたり searchCountPerRegistration 回のコース検索を行う
				for searchCount := 0; searchCount < searchCountPerRegistration; searchCount++ {
					if s.isNoRequestTime(ctx) {
						return
					}

					param := generate.SearchCourseParam()
					hres, res, err := SearchCourseAction(ctx, student.Agent, param, nextPathParam)
					if err != nil {
						step.AddError(err)
						continue
					}
					if err := verifySearchCourseResults(res, param); err != nil {
						step.AddError(err)
						continue
					}
					step.AddScore(score.SearchCourses)

					if len(res) > 0 {
						checkTargetID = res[0].ID.String()
					}

					// Linkヘッダから次ページのPath + QueryParamを取得
					_, nextPathParam = parseLinkHeader(hres)
				}

				// 検索で得たコースのシラバスを確認する
				if checkTargetID == "" {
					continue
				}

				if s.isNoRequestTime(ctx) {
					return
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
					} else {
						step.AddScore(score.GetCourseDetail)
					}
				} else {
					step.AddScore(score.GetCourseDetailVerifySkipped)
				}
			}

			// ----------------------------------------

			if s.isNoRequestTime(ctx) {
				return
			}

			registeredSchedule := student.RegisteredSchedule()
			_, getRegisteredCoursesRes, err := GetRegisteredCoursesAction(ctx, student.Agent)
			if err != nil {
				step.AddError(err)
				<-time.After(1 * time.Millisecond)
				continue
			}
			if err := verifyRegisteredCourses(getRegisteredCoursesRes, registeredSchedule); err != nil {
				step.AddError(err)
			} else {
				step.AddScore(score.GetRegisteredCourses)
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

			if s.isNoRequestTime(ctx) {
				return
			}

			// ----------------------------------------

			// ベンチ内で仮登録できたコースがあればAPIに登録処理を投げる
			if len(semiRegistered) == 0 {
				DebugLogger.Printf("[履修スキップ（空き講義不足)] code: %v, name: %v", student.Code, student.Name)
				continue
			}

			// 冪等なので登録済みのコースにもう一回登録して成功すれば200が返ってくる
		L:
			_, err = TakeCoursesAction(ctx, student.Agent, semiRegistered)
			if err != nil {
				step.AddError(err)
				if err, ok := err.(*url.Error); ok && err.Timeout() {
					ContestantLogger.Printf("履修登録(POST /api/me/courses)がタイムアウトしました。学生はリトライを試みます。")
					// timeout したらもう一回リクエストする
					<-time.After(100 * time.Millisecond)
					goto L
				} else {
					// 失敗時の仮登録情報のロールバック
					for _, c := range semiRegistered {
						c.FailRegistration()
						student.ReleaseTimeslot(c.DayOfWeek, c.Period)
					}
				}
			} else {
				step.AddScore(score.RegisterCourses)
				for _, c := range semiRegistered {
					c.SuccessRegistration(student)
					student.AddCourse(c)
					c.SetClosingAfterSecAtOnce(5 * time.Second) // 初履修者からn秒後に履修を締め切る
				}
			}

			s.debugData.AddInt("registrationTime", time.Since(registerStart).Milliseconds())
			DebugLogger.Printf("[履修完了] code: %v, time: %d ms, register count: %d", student.Code, time.Since(registerStart).Milliseconds(), len(semiRegistered))
		}
	}
}

func (s *Scenario) readAnnouncementScenario(student *model.Student, step *isucandar.BenchmarkStep) func(ctx context.Context) {
	return func(ctx context.Context) {
		var nextPathParam string // 次にアクセスするお知らせ一覧のページ
		for ctx.Err() == nil {

			if s.isNoRequestTime(ctx) {
				return
			}

			// 学生はお知らせを確認し続ける
			hres, res, err := GetAnnouncementListAction(ctx, student.Agent, nextPathParam)
			if err != nil {
				step.AddError(err)
				<-time.After(1 * time.Millisecond)
				continue
			}
			if err := verifyAnnouncements(&res, student); err != nil {
				step.AddError(err)
			} else {
				step.AddScore(score.GetAnnouncementList)
			}

			for _, ans := range res.Announcements {

				if ans.Unread {
					announcementStatus := student.GetAnnouncement(ans.ID)
					if announcementStatus == nil {
						// webappでは認識されているが、ベンチではまだ認識されていないお知らせ
						// load中には検証できないのでskip
						continue
					}

					if s.isNoRequestTime(ctx) {
						return
					}

					// お知らせの詳細を取得する
					_, res, err := GetAnnouncementDetailAction(ctx, student.Agent, ans.ID)
					if err != nil {
						step.AddError(err)
						continue // 次の未読おしらせの確認へ
					}

					if err := verifyAnnouncementDetail(&res, announcementStatus); err != nil {
						step.AddError(err)
					} else {
						step.AddScore(score.GetAnnouncementsDetail)
					}

					student.ReadAnnouncement(ans.ID)
				}
			}

			_, nextPathParam = parseLinkHeader(hres)
			// TODO: 現状: ページングで最後のページまで確認したら最初のページに戻る
			// TODO: 理想1: 未読お知らせを早く確認するため以降のページに未読が存在しないなら最初に戻る
			// TODO: 理想2: 10ページぐらい最低ページングする。10ページ目末尾のお知らせ以降に未読があればさらにページングする。無いならしない。
			// MEMO: Student.Announcementsはwebapp内のお知らせの順番(createdAt)と完全同期できていない
			// MEMO: 理想1,2を実現するためにはStudent.AnnouncementsをcreatedAtで保持する必要がある。insertできる木構造では持つのは辛いのでやりたくない。
			// ※ webappに追加するAnnouncementのcreatedAtはベンチ側が指定する

			// 未読お知らせがないのなら少しwaitして1ページ目から見直す
			if res.UnreadCount == 0 {
				nextPathParam = ""
				<-time.After(200 * time.Millisecond)
			}

			endTimeDuration := s.loadRequestEndTime.Sub(time.Now())
			select {
			case <-ctx.Done():
				return
			case <-time.After(endTimeDuration):
				return
			default:
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

		waitStart := time.Now()

		// コースgoroutineは満員 or 履修締め切りまではなにもしない or LoadEndTime
		endTimeDuration := s.loadRequestEndTime.Sub(time.Now())
		select {
		case <-time.After(endTimeDuration):
			return
		case <-course.WaitPreparedCourse(ctx):
		}

		// selectでのwaitは複数該当だとランダムなのでここでも判定
		if s.isNoRequestTime(ctx) {
			return
		}

		teacher := course.Teacher()
		// コースステータスをin-progressにする
		_, err := SetCourseStatusInProgressAction(ctx, teacher.Agent, course.ID)
		if err != nil {
			step.AddError(err)
			AdminLogger.Printf("%vのコースステータスをin-progressに変更するのが失敗しました", course.Name)
			return
		}
		s.debugData.AddInt("waitCourseTime", time.Since(waitStart).Milliseconds())
		DebugLogger.Printf("[科目開始] id: %v, time: %v, registered students: %v", course.ID, time.Since(waitStart).Milliseconds(), len(course.Students()))

		studentLen := len(course.Students())
		switch {
		case studentLen < 10:
			step.AddScore(score.StartCourseUnder10)
		case studentLen < 20:
			step.AddScore(score.StartCourseUnder20)
		case studentLen < 30:
			step.AddScore(score.StartCourseUnder30)
		case studentLen < 40:
			step.AddScore(score.StartCourseUnder40)
		case studentLen < 50:
			step.AddScore(score.StartCourseUnder50)
		case studentLen == 50:
			step.AddScore(score.StartCourseFull)
		case studentLen > 50:
			step.AddScore(score.StartCourseOver50)
		}

		var classRaps [classCountPerCourse]int64

		// コースの処理
		for i := 0; i < classCountPerCourse; i++ {
			
      classStart := time.Now()
      
			if s.isNoRequestTime(ctx) {
				return
			}

			timer := time.After(1 * time.Millisecond)

			classParam := generate.ClassParam(course, uint8(i+1))
		L:
			_, classRes, err := AddClassAction(ctx, teacher.Agent, course, classParam)
			if err != nil {
				var urlError *url.Error
				if errors.As(err, &urlError) && urlError.Timeout() {
					ContestantLogger.Printf("クラス追加(POST /api/:courseID/classes)がタイムアウトしました。教師はリトライを試みます。")
					<-time.After(100 * time.Millisecond)
					goto L
				} else {
					step.AddError(err)
					<-timer
					continue
				}
			} else {
				step.AddScore(score.AddClass)
			}
			class := model.NewClass(classRes.ClassID, classParam)
			course.AddClass(class)

			if s.isNoRequestTime(ctx) {
				return
			}

			announcement := generate.Announcement(course, class)
		ancLoop:
			_, ancRes, err := SendAnnouncementAction(ctx, teacher.Agent, announcement)
			if err != nil {
				var urlError *url.Error
				if errors.As(err, &urlError) && urlError.Timeout() {
					ContestantLogger.Printf("お知らせ追加(POST /api/announcements)がタイムアウトしました。教師はリトライを試みます。")
					<-time.After(100 * time.Millisecond)
					goto ancLoop
				} else {
					step.AddError(err)
					<-timer
					continue
				}
			} else {
				announcement.ID = ancRes.ID
				step.AddScore(score.AddAnnouncement)
			}
			course.BroadCastAnnouncement(announcement)

			if s.isNoRequestTime(ctx) {
				return
			}

			s.submitAssignments(ctx, course.Students(), course, class, announcement.ID, step)

			if s.isNoRequestTime(ctx) {
				return
			}

			_, assignmentsData, err := DownloadSubmissionsAction(ctx, teacher.Agent, course.ID, class.ID)
			if err != nil {
				step.AddError(err)
				continue
			}
			if err := verifyAssignments(assignmentsData, class); err != nil {
				step.AddError(err)
			} else {
				step.AddScore(score.DownloadSubmissions)
			}

			if s.isNoRequestTime(ctx) {
				return
			}

			_, err = s.scoringAssignments(ctx, course, class, teacher, step)
			if err != nil {
				step.AddError(err)
				<-timer
				continue
			} else {
				step.AddScore(score.RegisterScore)
			}

			classRaps[i] = time.Since(classStart).Milliseconds()
		}

		// クラスのラップタイム表示
		var compCount int
		var sumTime int64
		for _, cr := range classRaps {
			sumTime += cr
			if cr != 0 {
				compCount++
			}
		}

		s.debugData.AddInt("classAvgTime", int64(float64(sumTime)/float64(compCount)))
		s.debugData.AddInt("courseSumTime", sumTime)
		DebugLogger.Printf("[debug] 科目完了 Sum: %d ms, Avg: %.f ms, List(ms): %d, %d, %d, %d, %d",
			sumTime, float64(sumTime)/float64(compCount), classRaps[0], classRaps[1], classRaps[2], classRaps[3], classRaps[4])

		if s.isNoRequestTime(ctx) {
			return
		}

		// コースステータスをclosedにする
		_, err = SetCourseStatusClosedAction(ctx, teacher.Agent, course.ID)
		if err != nil {
			step.AddError(err)
			AdminLogger.Printf("%vのコースステータスをclosedに変更するのが失敗しました", course.Name)
			return
		}

		step.AddScore(score.FinishCourses)

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

			if s.isNoRequestTime(ctx) {
				return
			}

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

			if s.isNoRequestTime(ctx) {
				return
			}

			_, err = LoginAction(ctx, student.Agent, student.UserAccount)
			if err != nil {
				AdminLogger.Printf("学生 %vのログインが失敗しました", userData.Name)
				step.AddError(err)
				return
			}

			if s.isNoRequestTime(ctx) {
				return
			}

			_, res, err := GetMeAction(ctx, student.Agent)
			if err != nil {
				AdminLogger.Printf("学生 %vのユーザ情報取得に失敗しました", userData.Name)
				step.AddError(err)
				return
			}
			if err := verifyMe(&res, userData, false); err != nil {
				step.AddError(err)
				return
			}

			s.AddActiveStudent(student)
			s.sPubSub.Publish(student)
			step.AddScore(score.ActiveStudents)
		}()
	}
	wg.Wait()
}

func (s *Scenario) addCourseLoad(ctx context.Context, step *isucandar.BenchmarkStep) {
	teacher := s.GetRandomTeacher()
	courseParam := generate.CourseParam(teacher)

	if s.isNoRequestTime(ctx) {
		return
	}

	_, err := LoginAction(ctx, teacher.Agent, teacher.UserAccount)
	if err != nil {
		AdminLogger.Printf("teacherのログインに失敗しました")
		step.AddError(failure.NewError(fails.ErrCritical, err))
		return
	}

	if s.isNoRequestTime(ctx) {
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

	if s.isNoRequestTime(ctx) {
		return
	}

L:
	_, addCourseRes, err := AddCourseAction(ctx, teacher, courseParam)
	if err != nil {
		var urlError *url.Error
		if errors.As(err, &urlError) && urlError.Timeout() {
			// timeout したらもう一回リクエストする
			ContestantLogger.Printf("講義追加(POST /api/courses)がタイムアウトしました。教師はリトライを試みます。")
			<-time.After(100 * time.Millisecond)
			goto L
		} else {
			// タイムアウト以外の何らかのエラーだったら終わり
			step.AddError(err)
			return
		}
	}
	step.AddScore(score.AddCourse)

	course := model.NewCourse(courseParam, addCourseRes.ID, teacher)
	s.AddCourse(course)
	s.emptyCourseManager.AddEmptyCourse(course)
	s.cPubSub.Publish(course)
}

func (s *Scenario) submitAssignments(ctx context.Context, students map[string]*model.Student, course *model.Course, class *model.Class, announcementID string, step *isucandar.BenchmarkStep) {
	wg := sync.WaitGroup{}
	wg.Add(len(students))

	var unsuccess int64

	for _, student := range students {
		student := student
		go func() {
			defer wg.Done()

			endTimeDuration := s.loadRequestEndTime.Sub(time.Now())
			select {
			case <-time.After(endTimeDuration):
				return
			case <-time.After(waitReadClassAnnouncementTimeout):
				atomic.AddInt64(&unsuccess, 1)
				return
			case <-student.WaitReadAnnouncement(ctx, announcementID):
				// 学生sが課題お知らせを読むまで待つ
			}

			// selectでのwaitは複数該当だとランダムなのでここでも判定
			if s.isNoRequestTime(ctx) {
				return
			}

			// 講義一覧を取得する
			_, res, err := GetClassesAction(ctx, student.Agent, course.ID)
			if err != nil {
				step.AddError(err)
				return
			}
			if err := verifyClasses(res, course.Classes()); err != nil {
				step.AddError(err)
			} else {
				step.AddScore(score.GetClasses)
			}

			// 課題を提出する
			submissionData, fileName := generate.SubmissionData(course, class, student.UserAccount)

			if s.isNoRequestTime(ctx) {
				return
			}

			_, err = SubmitAssignmentAction(ctx, student.Agent, course.ID, class.ID, fileName, submissionData)
			if err != nil {
				step.AddError(err)
			} else {
				step.AddScore(score.SubmitAssignment)
				submission := model.NewSubmission(fileName, submissionData)
				class.AddSubmission(student.Code, submission)
			}
		}()
	}
	wg.Wait()
	if unsuccess > 0 {
		DebugLogger.Printf("[debug] %d 人( %d 人)の学生が%d秒以内に課題のお知らせを確認できなかったため課題を提出しませんでした", unsuccess, len(students), 5)
	}
}

// これここじゃないほうがいいかも知れない
type StudentScore struct {
	score int
	code  string
}

func (s *Scenario) scoringAssignments(ctx context.Context, course *model.Course, class *model.Class, teacher *model.Teacher, step *isucandar.BenchmarkStep) (*http.Response, error) {
	students := course.Students()
	scores := make([]StudentScore, 0, len(students))
	for _, s := range students {
		sub := class.GetSubmissionByStudentCode(s.Code)
		if sub == nil {
			continue
		}

		scores = append(scores, StudentScore{
			score: rand.Intn(101),
			code:  s.Code,
		})
	}

	if s.isNoRequestTime(ctx) {
		return nil, nil
	}

L:
	hres, err := PostGradeAction(ctx, teacher.Agent, course.ID, class.ID, scores)
	if err != nil {
		var urlError *url.Error
		if errors.As(err, &urlError) && urlError.Timeout() {
			ContestantLogger.Printf("成績追加(PUT /api/:courseID/classes/:classID/assignments/scores)がタイムアウトしました。教師はリトライを試みます。")
			// timeout したらもう一回リクエストする
			<-time.After(100 * time.Millisecond)
			goto L
		} else if hres != nil && hres.StatusCode == http.StatusNoContent {
			// すでにwebappに登録されていたら続ける
		} else {
			// タイムアウト以外の何らかのエラーだったら終わり
			return nil, err
		}
	}

	step.AddScore(score.RegisterScore)

	// POST成功したスコアをベンチ内に保存する
	for _, scoreData := range scores {
		sub := class.GetSubmissionByStudentCode(scoreData.code)
		if sub == nil {
			continue
		}
		sub.SetScore(scoreData.score)
	}
	return hres, nil
}
