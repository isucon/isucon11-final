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
	initialStudentsCount      = 50
	registerCourseLimit       = 20
	searchCountByRegistration = 3
	initialCourseCount        = 20
	courseProcessLimit        = 5
	// confirmAttendanceAnsTimeout は学生がクラス課題のお知らせを確認するのを待つ最大時間
	confirmAttendanceAnsTimeout = 5 * time.Second
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
	studentLoadWorker := s.createStudentLoadWorker(ctx, step)
	courseLoadWorker := s.createLoadCourseWorker(ctx, step)
	// LoadWorkerに初期負荷を追加
	// (負荷追加はScenarioのPubSub経由で行われるので引数にLoadWorkerは不要)

	// FIXME: コース追加前にコース登録アクションが必要
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

	// 成績確認 + (空きがあれば履修登録)
	s.sPubSub.Subscribe(ctx, func(mes interface{}) {
		var student *model.Student
		var ok bool
		if student, ok = mes.(*model.Student); !ok {
			AdminLogger.Println("sPubSub に *model.Student以外が飛んできました")
			return
		}
		AdminLogger.Println(student.Name, "の成績確認タスクが追加された") // FIXME: for debug

		// FIXME for Debug
		{
			s.mu.Lock()
			s.activeStudentCount++
			s.mu.Unlock()
		}

		studentLoadWorker.Do(func(ctx context.Context) {
			for ctx.Err() == nil {
				// 学生は成績を確認し続ける
				// TODO: verify grade
				_, _, err := GetGradeAction(ctx, student.Agent)
				if err != nil {
					step.AddError(err)
					<-time.After(3000 * time.Millisecond)
					continue
				}
				step.AddScore(score.CountGetGrades)
				AdminLogger.Printf("%vは成績を確認した", student.Name)

				wishRegisterCount := registerCourseLimit - student.RegisteringCount()

				// 履修希望コース * searchCountByRegistration 回 検索を行う
				for i := 0; i < wishRegisterCount*searchCountByRegistration; i++ {
					timer := time.After(300 * time.Millisecond)

					// TODO: verify course
					_, _, err := SearchCourseAction(ctx, student.Agent)

					if err != nil {
						step.AddError(err)
						<-timer
						continue
					}
					step.AddScore(score.CountSearchCourse)

					select {
					case <-ctx.Done():
						return
					case <-timer:
					}
				}
				AdminLogger.Printf("%vはコースを%v回検索した", student.Name, wishRegisterCount*searchCountByRegistration)

				select {
				case <-ctx.Done():
					return
				default:
				}

				// 仮登録(ベンチ内部では登録済みにする)
				// TODO: 1度も検索成功してなかったら登録しない
				semiRegistered := make([]*model.Course, 0, wishRegisterCount)

				randTimeSlots := generate.RandomIntSlice(30) // 平日分のコマ 5*6

				studentScheduleMutex := student.ScheduleMutex()
				studentScheduleMutex.Lock()
				for i := 0; i < len(randTimeSlots); i++ {
					if len(semiRegistered) >= wishRegisterCount {
						break
					}

					dayOfWeek := randTimeSlots[i]/6 + 1 // 日曜日分+1
					period := randTimeSlots[i] % 6

					if !student.IsEmptyTimeSlots(dayOfWeek, period) {
						continue
					}

					registeredCourse := s.emptyCourseManager.AddStudentForRegistrableCourse(student, dayOfWeek, period)
					if registeredCourse == nil { // 該当コマで空きコースがなかった
						continue
					}

					student.FillTimeslot(dayOfWeek, period)
					semiRegistered = append(semiRegistered, registeredCourse)
				}
				studentScheduleMutex.Unlock()

				select {
				case <-ctx.Done():
					return
				default:
				}

				// ベンチ内で登録できたコースがあればAPIにも登録処理を投げる
				if len(semiRegistered) > 0 {
					_, err := TakeCoursesAction(ctx, student.Agent, semiRegistered)

					if err != nil { // API側が原因のエラー（コースが登録不可ステータスだったり満席のエラーなら非該当）
						step.AddError(err)
					}

					isSuccess := err == nil
					if isSuccess {
						step.AddScore(score.CountRegisterCourses)
						for _, c := range semiRegistered {
							c.ReduceTempRegistered()
							c.SetUnRegistrableAfterSecAtOnce(5 * time.Second) // 初履修者からn秒後に履修を締め切る
							student.AddCourse(c)
							AdminLogger.Printf("%vは%vを履修した", student.Name, c.Name)
						}
					} else {
						for _, c := range semiRegistered {
							c.RemoveStudent(student)
							student.ReleaseTimeslot(c.DayOfWeek, c.Period)
						}
					}
				}
				// TODO: できれば登録に失敗したコースを抜いて再度登録する

				select {
				case <-ctx.Done():
					return
				case <-time.After(3000 * time.Millisecond):
				}
			}
		})
	})

	// おしらせ確認 + 既読追加
	s.sPubSub.Subscribe(ctx, func(mes interface{}) {
		var student *model.Student
		var ok bool
		if student, ok = mes.(*model.Student); !ok {
			AdminLogger.Println("sPubSub に *model.Student以外が飛んできました")
			return
		}
		AdminLogger.Println(student.Name, "のおしらせタスクが追加された") // FIXME: for debug
		studentLoadWorker.Do(func(ctx context.Context) {
			var next string // 次にアクセスするお知らせ一覧のページ
			for ctx.Err() == nil {
				// 学生はお知らせを確認し続ける
				hres, announceList, err := GetAnnouncementListAction(ctx, student.Agent, next)
				if err != nil {
					step.AddError(err)
					<-time.After(3000 * time.Millisecond)
					continue
				}
				// TODO: verify announceList
				step.AddScore(score.CountGetAnnouncements)

				for _, ans := range announceList.Announcements {
					if ans.Unread {
						select {
						case <-ctx.Done():
							return
						default:
						}
						_, _, err := GetAnnouncementDetailAction(ctx, student.Agent, ans.ID)
						if err != nil {
							step.AddError(err)
							continue // 次の未読おしらせの確認へ
						}
						student.ReadAnnouncement(ans.ID)
						step.AddScore(score.CountGetAnnouncementsDetail)
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
		})
	})
	return studentLoadWorker
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
		loadCourseWorker.Do(func(ctx context.Context) {
			// コースgoroutineは満員 or 履修締め切りまではなにもしない
			<-course.WaitFullOrUnRegistrable(ctx)

			faculty := course.Faculty()
			_, err := SetCourseStatusInProgressAction(ctx, faculty.Agent, course.ID)
			if err != nil {
				step.AddError(err)
				AdminLogger.Println("コースステータスをin-progressに変更するのが失敗しました")
				return
			}

			// コースの処理
			for i := 0; i < courseProcessLimit; i++ {
				timer := time.After(100 * time.Millisecond)

				// FIXME: verify class
				classParam := generate.ClassParam(uint8(i + 1))
				_, class, announcement, err := AddClassAction(ctx, faculty.Agent, course, classParam)
				if err != nil {
					step.AddError(err)
					<-timer
					continue
				}
				course.AddClass(class)
				course.BroadCastAnnouncement(announcement)
				step.AddScore(score.CountAddClass)
				step.AddScore(score.CountAddAssignment)

				errs := submitAssignments(ctx, course.Students(), course, class, announcement.ID)
				for _, e := range errs {
					step.AddError(e)
				}

				// TODO: verify data
				_, assignmentsData, err := DownloadSubmissionsAction(ctx, faculty.Agent, course.ID, class.ID)
				if err != nil {
					step.AddError(err)
					return
				}

				// TODO: 採点する
				_, err = scoringAssignments(ctx, course.ID, class.ID, faculty, course.Students(), assignmentsData)
				if err != nil {
					step.AddError(err)
					<-timer
					continue
				}

				step.AddScore(score.CountSubmitAssignment)
				step.AddScore(score.CountAddAssignmentScore)
				select {
				case <-ctx.Done():
					return
				case <-timer:
				}
			}

			// コースがおわった
			AdminLogger.Println(course.Name, "は終了した")
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
			return
		})
	})
	return loadCourseWorker
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

			_, err = LoginAction(ctx, student.Agent, student.UserAccount)
			if err != nil {
				ContestantLogger.Printf("学生%vのログインが失敗しました", userData.Name)
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
	faculty := s.GetRandomFaculty()
	courseParam := generate.CourseParam(faculty)

	_, err := LoginAction(ctx, faculty.Agent, faculty.UserAccount)
	if err != nil {
		ContestantLogger.Printf("facultyのログインに失敗しました")
		step.AddError(failure.NewError(fails.ErrCritical, err))
		return
	}

	_, res, err := AddCourseAction(ctx, faculty, courseParam)
	if err != nil {
		step.AddError(err)
		return
	}
	course := model.NewCourse(courseParam, res.ID, faculty)

	s.AddCourse(course)
	s.emptyCourseManager.AddEmptyCourse(course)
	s.cPubSub.Publish(course)
	step.AddScore(score.CountAddCourse)
}

func submitAssignments(ctx context.Context, students []*model.Student, course *model.Course, class *model.Class, announcementID string) []error {
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
			case <-time.After(confirmAttendanceAnsTimeout):
				AdminLogger.Printf("学生が%d秒以内に課題のお知らせを確認できなかったため課題を提出しませんでした", confirmAttendanceAnsTimeout)
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
				errs = append(errs, err)
			}

			select {
			case <-ctx.Done():
				return
			default:
			}

			// 課題を提出する
			submission := generate.Submission()
			_, err = SubmitAssignmentAction(ctx, s.Agent, course.ID, class.ID, submission)
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			} else {
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

func scoringAssignments(ctx context.Context, courseID, classID string, faculty *model.Faculty, students []*model.Student, assignments []byte) (*http.Response, error) {
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
	res, err := PostGradeAction(ctx, faculty.Agent, courseID, classID, scores)
	if err != nil {
		return nil, err
	}

	return res, nil
}
