package scenario

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
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
	// 登録された科目による負荷が存在する
	studentLoadWorker := s.createStudentLoadWorker(ctx, step) // Gradeの確認から始まるシナリオとAnnouncementsの確認から始まるシナリオの二種類を担うgoroutineがアクティブ学生ごとに起動している
	courseLoadWorker := s.createLoadCourseWorker(ctx, step)   // 登録された科目につき一つのgoroutineが起動している

	// コース履修が完了した際のカウントアップをするPubSubを設定する
	s.setFinishCourseCountPubSub(ctx, step)

	// LoadWorkerに初期負荷を追加
	// (負荷追加はScenarioのPubSub経由で行われるので引数にLoadWorkerは不要)
	wg := sync.WaitGroup{}
	wg.Add(initialCourseCount + 1)
	arr := generate.ShuffledInts(initialCourseCount)
	for i := 0; i < initialCourseCount; i++ {
		timeslot := arr[i] % 30
		dayOfWeek := timeslot / 6
		period := timeslot % 6
		go func() {
			defer DebugLogger.Printf("[debug] initial Courses added")
			defer wg.Done()
			s.addCourseLoad(ctx, dayOfWeek, period, step)
		}()
	}
	go func() {
		defer DebugLogger.Printf("[debug] initial ActiveStudents added")
		defer wg.Done()
		s.addActiveStudentLoads(ctx, step, initialStudentsCount)
	}()
	wg.Wait()

	if s.CourseManager.GetCourseCount() == 0 {
		step.AddError(failure.NewError(fails.ErrCritical, fmt.Errorf("科目登録が1つも成功しませんでした")))
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
		if len(v) == 0 {
			continue
		}

		var sum int64
		for _, t := range v {
			sum += t
		}
		avg := int64(float64(sum) / float64(len(v)))

		sorted := make([]int64, len(v))
		copy(sorted, v)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i] < sorted[j]
		})

		tile50 := sorted[int(float64(len(sorted))*0.5)]
		tile90 := sorted[int(float64(len(sorted))*0.9)]
		tile99 := sorted[int(float64(len(sorted))*0.99)]
		DebugLogger.Printf("%s: avg %d, 50tile %d, 90tile %d, 99tile %d", k, avg, tile50, tile90, tile99)
	}

	return nil
}

// isNoRequestTime はリクエスト送信できない期間かどうか（各Actionの前に必ず調べる）
func (s *Scenario) isNoRequestTime(ctx context.Context) bool {
	return time.Now().After(s.loadRequestEndTime) || ctx.Err() != nil
}

// isNoRetryTime はリクエストのリトライができない期間かどうか
func (s *Scenario) isNoRetryTime(ctx context.Context) bool {
	retryableTime := s.loadRequestEndTime.Add(5 * time.Second)
	return time.Now().After(retryableTime) || ctx.Err() != nil
}

func (s *Scenario) setFinishCourseCountPubSub(ctx context.Context, step *isucandar.BenchmarkStep) {
	s.finishCoursePubSub.Subscribe(ctx, func(mes interface{}) {
		count, ok := mes.(int)
		if !ok {
			// unreachable
			panic("finishCoursePubSub に int以外が飛んできました")
		}

		for i := 0; i < count; i++ {
			step.AddScore(score.FinishCoursesStudents)
			result := atomic.AddInt64(&s.finishCourseStudentsCount, 1)
			if result%StudentCapacityPerCourse == 0 {
				s.addActiveStudentLoads(ctx, step, 1)
			}
		}
	})
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
			// unreachable
			panic("sPubSub に *model.Student以外が飛んできました")
		}

		s.AddActiveStudent(student)
		activeCount := atomic.AddInt64(&s.activeStudentsCount, 1)

		if activeCount%AnnouncePagingStudentInterval == 0 {
			studentLoadWorker.Do(s.readAnnouncementPagingScenario(student, step))
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

			if student.RegisteringCount() >= registerCourseLimitPerStudent {
				// gradeはTimeSlotの空きが発生したらリクエストを送る
				_ctx, cancel := context.WithDeadline(ctx, s.loadRequestEndTime)
				<-student.WaitReleaseTimeslot(_ctx, cancel, registerCourseLimitPerStudent)
			}

			// grade と search が早くなりすぎると科目登録が1つずつしか発生せずブレが発生する
			timer := time.After(100 * time.Millisecond)

			if s.isNoRequestTime(ctx) {
				return
			}

			// 履修したコースが0なら成績確認をしない
			if len(student.Courses()) != 0 {
				// 成績確認
				expected := collectVerifyGradesData(student)
				_, getGradeRes, err := GetGradeAction(ctx, student.Agent)
				if err != nil {
					step.AddError(err)
					time.Sleep(1 * time.Millisecond)
					continue
				}
				err = verifyGrades(expected, &getGradeRes)
				if err != nil {
					step.AddError(err)
				} else {
					step.AddScore(score.GetGrades)
				}
			}

			// ----------------------------------------
			{
				var checkTargetID string
				var nextPathParam string // 次にアクセスする検索一覧のページ
				var param *model.SearchCourseParam
				// 履修希望科目1つあたり searchCountPerRegistration 回の科目検索を行う
				for searchCount := 0; searchCount < searchCountPerRegistration; searchCount++ {
					if s.isNoRequestTime(ctx) {
						return
					}

					if nextPathParam == "" { // 2ページ目以降は同じパラメータで
						param = generate.SearchCourseParam()
					}

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

				if s.isNoRequestTime(ctx) {
					return
				}

				// 検索で得た科目のシラバスを確認する
				// TODO: 検索は何らかが必ずヒットするようにする
				if checkTargetID != "" {
					_, res, err := GetCourseDetailAction(ctx, student.Agent, checkTargetID)
					if err != nil {
						step.AddError(err)
						continue
					}
					expected, exists := s.CourseManager.GetCourseByID(res.ID.String())
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

			}

			// ----------------------------------------

			if s.isNoRequestTime(ctx) {
				return
			}

			registeredSchedule := student.RegisteredSchedule()
			_, getRegisteredCoursesRes, err := GetRegisteredCoursesAction(ctx, student.Agent)
			if err != nil {
				step.AddError(err)
				continue
			}
			if err := verifyRegisteredCourses(getRegisteredCoursesRes, registeredSchedule); err != nil {
				step.AddError(err)
			} else {
				step.AddScore(score.GetRegisteredCourses)
			}

			// ----------------------------------------

			// grade と search が早くなりすぎると科目登録が1つずつしか発生せずブレが発生する
			// あまりにも早い場合はここでMAX100ms待つ
			<-timer

			// 仮登録
			remainingRegistrationCapacity := registerCourseLimitPerStudent - student.RegisteringCount()
			if remainingRegistrationCapacity == 0 {
				// unreachable
				DebugLogger.Printf("[履修スキップ（空きコマ不足)] code: %v, name: %v", student.Code, student.Name)
				continue
			}
			temporaryReservedCourses := s.CourseManager.ReserveCoursesForStudent(student, remainingRegistrationCapacity)

			if s.isNoRequestTime(ctx) {
				return
			}

			// ----------------------------------------

			// ベンチ内で仮登録できた科目があればAPIに登録処理を投げる
			if len(temporaryReservedCourses) == 0 {
				// unreachable
				DebugLogger.Printf("[履修スキップ（空き科目不足)] code: %v, name: %v", student.Code, student.Name)
				continue
			}

			// 60秒以降のリトライリクエストかどうか
			isExtendRequest := false
		L:
			if s.isNoRetryTime(ctx) {
				return
			}
			// 冪等なので登録済みの科目にもう一回登録して成功すれば200が返ってくる
			_, _, err = TakeCoursesAction(ctx, student.Agent, temporaryReservedCourses)
			if err != nil {
				step.AddError(err)
				var urlError *url.Error
				if errors.As(err, &urlError) && urlError.Timeout() {
					ContestantLogger.Printf("履修登録(POST /api/me/courses)がタイムアウトしました。学生はリトライを試みます。")
					// timeout したらもう一回リクエストする
					time.Sleep(100 * time.Millisecond)
					isExtendRequest = s.isNoRequestTime(ctx)
					goto L
				} else {
					// 失敗時に科目の仮登録をロールバック
					for _, c := range temporaryReservedCourses {
						c.RollbackReservation()
						student.ReleaseTimeslot(c.DayOfWeek, c.Period)
					}
				}
			} else {
				if !isExtendRequest {
					step.AddScore(score.RegisterCourses)
				}
				for _, c := range temporaryReservedCourses {
					step.AddScore(score.RegisterCourseByStudent)
					c.CommitReservation(student)
					student.AddCourse(c)
					c.StartTimer(waitCourseFullTimeout)
				}
			}

			if student.RegisteringCount() == registerCourseLimitPerStudent {
				s.RegistarableStudentList[student.Code] = false
				if atomic.AddInt64(&s.RegistarableStudentCount, -1) == 0 {
					s.CourseManager.StartAllWaitingCourses()
				}
			}
			DebugLogger.Printf("[履修完了] code: %v, register count: %d", student.Code, len(temporaryReservedCourses))
		}
		// TODO: できれば登録に失敗した科目を抜いて再度登録する
	}
}

func (s *Scenario) readAnnouncementScenario(student *model.Student, step *isucandar.BenchmarkStep) func(ctx context.Context) {
	return func(ctx context.Context) {
		var nextPathParam string // 次にアクセスするお知らせ一覧のページ
		for ctx.Err() == nil {
			timer := time.After(50 * time.Millisecond)

			if s.isNoRequestTime(ctx) {
				return
			}

			expectAnnouncementList := student.AnnouncementsMap()

			startGetAnnouncementList := time.Now()
			// 学生はお知らせを確認し続ける
			hres, res, err := GetAnnouncementListAction(ctx, student.Agent, nextPathParam)
			if err != nil {
				step.AddError(err)
				time.Sleep(100 * time.Millisecond)
				continue
			}
			s.debugData.AddInt("GetAnnouncementListTime", time.Since(startGetAnnouncementList).Milliseconds())

			if err := verifyAnnouncementsList(&res, expectAnnouncementList, true); err != nil {
				step.AddError(err)
			} else {
				step.AddScore(score.GetAnnouncementList)
			}

			// このページに存在する未読お知らせ数（ページングするかどうかの判定用）
			var unreadCount int
			for _, ans := range res.Announcements {
				if ans.Unread {
					unreadCount++
				}

				announcementStatus := student.GetAnnouncement(ans.ID)
				if announcementStatus == nil {
					// webappでは認識されているが、ベンチではまだ認識されていないお知らせ
					// load中には検証できないので既読化しない
					continue
				}
				// 前にタイムアウトになってしまっていた場合、もしくはまだ見ていないお知らせの場合詳細を見に行く
				if !(announcementStatus.Dirty || ans.Unread) {
					continue
				}

				if s.isNoRequestTime(ctx) {
					return
				}

				startGetAnnouncementDetail := time.Now()
				// お知らせの詳細を取得する
				_, res, err := GetAnnouncementDetailAction(ctx, student.Agent, ans.ID)
				if err != nil {
					var urlError *url.Error
					if errors.As(err, &urlError) && urlError.Timeout() {
						student.MarkAnnouncementReadDirty(ans.ID)
					}
					step.AddError(err)
					continue // 次の未読おしらせの確認へ
				}
				s.debugData.AddInt("GetAnnouncementDetailTime", time.Since(startGetAnnouncementDetail).Milliseconds())

				if err := verifyAnnouncementDetail(&res, announcementStatus); err != nil {
					step.AddError(err)
				} else {
					step.AddScore(score.GetAnnouncementsDetail)
				}

				student.ReadAnnouncement(ans.ID)
			}

			_, nextPathParam = parseLinkHeader(hres)
			// MEMO: Student.Announcementsはwebapp内のお知らせの順番(createdAt)と完全同期できていない
			// MEMO: 理想1,2を実現するためにはStudent.AnnouncementsをcreatedAtで保持する必要がある。insertできる木構造では持つのは辛いのでやりたくない。
			// ※ webappに追加するAnnouncementのcreatedAtはベンチ側が指定する

			// 以降のページに未読お知らせがない（このページの未読数とレスポンスの未読数が一致）
			// DoSにならないように少しwaitして1ページ目から見直す
			if res.UnreadCount == unreadCount {
				nextPathParam = ""
				if !student.HasUnreadAnnouncement() {
					select {
					case <-time.After(400 * time.Millisecond):
					case <-student.WaitNewUnreadAnnouncement(ctx):
						// waitはお知らせ追加したらエスパーで即解消する
					}
				}
			}

			// 50msより短い間隔で一覧取得をしない
			<-timer
		}
	}
}

func (s *Scenario) readAnnouncementPagingScenario(student *model.Student, step *isucandar.BenchmarkStep) func(ctx context.Context) {
	return func(ctx context.Context) {
		var nextPathParam string // 次にアクセスするお知らせ一覧のページ
		for ctx.Err() == nil {
			timer := time.After(50 * time.Millisecond)

			if s.isNoRequestTime(ctx) {
				return
			}

			expectAnnounceList := student.AnnouncementsMap()

			startGetAnnouncementList := time.Now()
			// 学生はお知らせを確認し続ける
			hres, res, err := GetAnnouncementListAction(ctx, student.Agent, nextPathParam)
			if err != nil {
				step.AddError(err)
				<-timer
				continue
			}
			s.debugData.AddInt("GetAnnouncementListTime", time.Since(startGetAnnouncementList).Milliseconds())

			// 並列で走る既読にするシナリオが未読/既読状態を変更するので、こちらのシナリオでは未読/既読状態は検証しない
			if err := verifyAnnouncementsList(&res, expectAnnounceList, false); err != nil {
				step.AddError(err)
			} else {
				step.AddScore(score.GetAnnouncementList)
			}

			// このページ内で既読のおしらせを集める
			var readAnnouncementsID []string
			for _, ans := range res.Announcements {
				if !ans.Unread {
					readAnnouncementsID = append(readAnnouncementsID, ans.ID)
				}
			}

			if s.isNoRequestTime(ctx) {
				return
			}

			// 既読おしらせが存在したら1つだけ確認する
			if len(readAnnouncementsID) > 0 {
				targetID := readAnnouncementsID[rand.Intn(len(readAnnouncementsID))]

				expectStatus := student.GetAnnouncement(targetID)
				if expectStatus == nil {
					// unreachable
					// ベンチは認識していないお知らせを既読化することはない
					panic("read unknown announcement")
				}

				_, res, err := GetAnnouncementDetailAction(ctx, student.Agent, targetID)
				if err != nil {
					step.AddError(err)
					<-timer
					continue
				}

				if err := verifyAnnouncementDetail(&res, expectStatus); err != nil {
					step.AddError(err)
				} else {
					step.AddScore(score.GetAnnouncementsDetail)
				}
			}

			_, nextPathParam = parseLinkHeader(hres)

			// 50msより短い間隔で一覧取得をしない
			<-timer
		}
	}
}

func (s *Scenario) createLoadCourseWorker(ctx context.Context, step *isucandar.BenchmarkStep) *parallel.Parallel {
	// 追加された科目の動作を回し続けるParallel
	loadCourseWorker := parallel.NewParallel(ctx, -1)
	s.cPubSub.Subscribe(ctx, func(mes interface{}) {
		var course *model.Course
		var ok bool
		if course, ok = mes.(*model.Course); !ok {
			// unreachable
			panic("cPubSub に *model.Course以外が飛んできました")
		}
		loadCourseWorker.Do(s.courseScenario(course, step))
	})
	return loadCourseWorker
}

func (s *Scenario) courseScenario(course *model.Course, step *isucandar.BenchmarkStep) func(ctx context.Context) {
	return func(ctx context.Context) {
		defer func() {
			for _, student := range course.Students() {
				student.ReleaseTimeslot(course.DayOfWeek, course.Period)
				if s.RegistarableStudentList[student.Code] == false {
					s.RegistarableStudentList[student.Code] = true
					atomic.AddInt64(&s.RegistarableStudentCount, 1)
				}
			}
		}()

		waitStart := time.Now()

		// 履修締め切りを待つ
		_ctx, cancel := context.WithDeadline(ctx, s.loadRequestEndTime)
		<-course.Wait(_ctx, cancel, func() {
			// 科目を追加
			s.addCourseLoad(ctx, course.DayOfWeek, course.Period, step)
			s.addCourseLoad(ctx, course.DayOfWeek, course.Period, step)
		})

		if s.isNoRequestTime(ctx) {
			return
		}

		// 履修登録を締め切ったので候補から取り除く
		s.CourseManager.RemoveRegistrationClosedCourse(course)

		teacher := course.Teacher()
		// 科目ステータスをin-progressにする
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
		case studentLen < 50:
			step.AddScore(score.StartCourseUnder50)
		case studentLen == 50:
			step.AddScore(score.StartCourseFull)
		case studentLen > 50:
			step.AddScore(score.StartCourseOver50)
		}

		var classTimes [ClassCountPerCourse]int64

		// 科目の処理
		for i := 0; i < ClassCountPerCourse; i++ {
			classStart := time.Now()

			if s.isNoRequestTime(ctx) {
				return
			}

			timer := time.After(1 * time.Millisecond)

			classParam := generate.ClassParam(course, uint8(i+1))

			// 60秒以降のリトライリクエストかどうか
			isExtendRequest := false
		L:
			if s.isNoRetryTime(ctx) {
				return
			}
			_, classRes, err := AddClassAction(ctx, teacher.Agent, course, classParam)
			if err != nil {
				var urlError *url.Error
				if errors.As(err, &urlError) && urlError.Timeout() {
					ContestantLogger.Printf("クラス追加(POST /api/:courseID/classes)がタイムアウトしました。教師はリトライを試みます。")
					time.Sleep(100 * time.Millisecond)
					isExtendRequest = s.isNoRequestTime(ctx)
					goto L
				} else {
					step.AddError(err)
					<-timer
					continue
				}
			} else {
				if !isExtendRequest {
					step.AddScore(score.AddClass)
				}
			}
			class := model.NewClass(classRes.ClassID, classParam)
			course.AddClass(class)

			if s.isNoRequestTime(ctx) {
				return
			}

			announcement := generate.Announcement(course, class)

			// 60秒以降のリトライリクエストかどうか
			isExtendRequest = false
		ancLoop:
			if s.isNoRetryTime(ctx) {
				return
			}
			_, ancRes, err := SendAnnouncementAction(ctx, teacher.Agent, announcement)
			if err != nil {
				var urlError *url.Error
				if errors.As(err, &urlError) && urlError.Timeout() {
					ContestantLogger.Printf("お知らせ追加(POST /api/announcements)がタイムアウトしました。教師はリトライを試みます。")
					time.Sleep(100 * time.Millisecond)
					isExtendRequest = s.isNoRequestTime(ctx)
					goto ancLoop
				} else {
					step.AddError(err)
					<-timer
					continue
				}
			} else {
				announcement.ID = ancRes.ID
				if !isExtendRequest {
					step.AddScore(score.AddAnnouncement)
				}
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
			}

			classTimes[i] = time.Since(classStart).Milliseconds()
		}

		// クラスのラップタイム表示
		var compCount int
		var sumTime int64
		for _, ct := range classTimes {
			sumTime += ct
			if ct != 0 {
				compCount++
			}
		}

		s.debugData.AddInt("classAvgTime", int64(float64(sumTime)/float64(compCount)))
		s.debugData.AddInt("classTotalTime", sumTime)
		DebugLogger.Printf("[debug] 科目完了 Sum: %d ms, Avg: %.f ms, List(ms): %d, %d, %d, %d, %d",
			sumTime, float64(sumTime)/float64(compCount), classTimes[0], classTimes[1], classTimes[2], classTimes[3], classTimes[4])

		if s.isNoRequestTime(ctx) {
			return
		}

		// 科目ステータスをclosedにする
		_, err = SetCourseStatusClosedAction(ctx, teacher.Agent, course.ID)
		if err != nil {
			step.AddError(err)
			AdminLogger.Printf("%vのコースステータスをclosedに変更するのが失敗しました", course.Name)
			return
		}

		step.AddScore(score.FinishCourses)

		// 科目が追加されたのでベンチのアクティブ学生も増やす
		s.finishCoursePubSub.Publish(len(course.Students()))
	}
}

func (s *Scenario) addActiveStudentLoads(ctx context.Context, step *isucandar.BenchmarkStep, count int) {
	wg := sync.WaitGroup{}
	wg.Add(count)
	for i := 0; i < count; i++ {
		go func() {
			defer wg.Done()

			student, err := s.userPool.newStudent()
			if err != nil {
				return
			}

			if s.isNoRequestTime(ctx) {
				return
			}

			hres, resources, err := AccessTopPageAction(ctx, student.Agent)
			if err != nil {
				AdminLogger.Printf("学生 %vがログイン画面にアクセスできませんでした", student.Name)
				step.AddError(err)
				return
			}
			errs := verifyPageResource(hres, resources)
			if len(errs) != 0 {
				AdminLogger.Printf("学生 %vがアクセスしたログイン画面の検証に失敗しました", student.Name)
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
				AdminLogger.Printf("学生 %vのログインが失敗しました", student.Name)
				step.AddError(err)
				return
			}

			if s.isNoRequestTime(ctx) {
				return
			}

			_, res, err := GetMeAction(ctx, student.Agent)
			if err != nil {
				AdminLogger.Printf("学生 %vのユーザ情報取得に失敗しました", student.Name)
				step.AddError(err)
				return
			}
			if err := verifyMe(&res, student.UserAccount, false); err != nil {
				step.AddError(err)
				return
			}

			s.sPubSub.Publish(student)
			step.AddScore(score.ActiveStudents)
		}()
	}
	wg.Wait()
}

// CourseManagerと整合性を取るためdayOfWeekとPeriodを前回から引き継ぐ必要がある（初回を除く）
func (s *Scenario) addCourseLoad(ctx context.Context, dayOfWeek, period int, step *isucandar.BenchmarkStep) {
	teacher := s.userPool.randomTeacher()
	courseParam := generate.CourseParam(dayOfWeek, period, teacher)

	if s.isNoRequestTime(ctx) {
		return
	}

	isLoggedIn := teacher.LoginOnce(func(teacher *model.Teacher) {
		_, err := LoginAction(ctx, teacher.Agent, teacher.UserAccount)
		if err != nil {
			step.AddError(err)
			return
		}
		teacher.IsLoggedIn = true
	})
	if !isLoggedIn {
		// ログインに失敗したらコース追加中断
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

	// 60秒以降のリトライリクエストかどうか
	isExtendRequest := false
L:
	if s.isNoRetryTime(ctx) {
		return
	}
	_, addCourseRes, err := AddCourseAction(ctx, teacher.Agent, courseParam)
	if err != nil {
		var urlError *url.Error
		if errors.As(err, &urlError) && urlError.Timeout() {
			// timeout したらもう一回リクエストする
			ContestantLogger.Printf("講義追加(POST /api/courses)がタイムアウトしました。教師はリトライを試みます。")
			time.Sleep(100 * time.Millisecond)
			isExtendRequest = s.isNoRequestTime(ctx)
			goto L
		} else {
			// タイムアウト以外の何らかのエラーだったら終わり
			step.AddError(err)
			return
		}
	}
	if !isExtendRequest {
		step.AddScore(score.AddCourse)
	}

	course := model.NewCourse(courseParam, addCourseRes.ID, teacher, StudentCapacityPerCourse)
	s.CourseManager.AddNewCourse(course)
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

			waitStartTime := time.Now()
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
			s.debugData.AddInt("waitReadAnnouncement", time.Since(waitStartTime).Milliseconds())

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

			// 60秒以降のリトライリクエストかどうか
			isExtendRequest := false
		L:
			if s.isNoRetryTime(ctx) {
				return
			}
			_, err = SubmitAssignmentAction(ctx, student.Agent, course.ID, class.ID, fileName, submissionData)
			var urlError *url.Error
			if errors.As(err, &urlError) && urlError.Timeout() {
				ContestantLogger.Printf("課題提出(POST /api/:courseID/classes/:classID/assignments)がタイムアウトしました。学生はリトライを試みます。")
				time.Sleep(100 * time.Millisecond)
				isExtendRequest = s.isNoRequestTime(ctx)
				goto L
			}
			if err != nil {
				step.AddError(err)
			} else {
				if !isExtendRequest {
					step.AddScore(score.SubmitAssignment)
				}
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

	// 60秒以降のリトライリクエストかどうか
	isExtendRequest := false
L:
	if s.isNoRetryTime(ctx) {
		return nil, nil
	}
	hres, err := PostGradeAction(ctx, teacher.Agent, course.ID, class.ID, scores)
	if err != nil {
		var urlError *url.Error
		if errors.As(err, &urlError) && urlError.Timeout() {
			ContestantLogger.Printf("成績追加(PUT /api/:courseID/classes/:classID/assignments/scores)がタイムアウトしました。教師はリトライを試みます。")
			// timeout したらもう一回リクエストする
			time.Sleep(100 * time.Millisecond)
			isExtendRequest = s.isNoRequestTime(ctx)
			goto L
		} else if hres != nil && hres.StatusCode == http.StatusNoContent {
			// すでにwebappに登録されていたら続ける
		} else {
			// タイムアウト以外の何らかのエラーだったら終わり
			return nil, err
		}
	}

	if !isExtendRequest {
		step.AddScore(score.RegisterScore)
	}

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
