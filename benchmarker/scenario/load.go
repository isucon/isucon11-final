package scenario

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/parallel"
	"github.com/isucon/isucon11-final/benchmarker/generate"
	"github.com/isucon/isucon11-final/benchmarker/model"
	"github.com/isucon/isucon11-final/benchmarker/score"
)

const (
	// confirmAttendanceAnsTimeout は学生がクラス課題のお知らせを確認するのを待つ最大時間
	confirmAttendanceAnsTimeout time.Duration = 5 * time.Second
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
	wg.Add(InitialCourseCount)
	for i := 0; i < InitialCourseCount; i++ {
		go func() {
			defer wg.Done()
			step.AddScore(score.CountAddCourse)
			s.addCourseLoad(generate.Course())
		}()
	}
	wg.Wait()

	wg = sync.WaitGroup{}
	wg.Add(3)
	go func() {
		defer wg.Done()
		s.addActiveStudentLoads(50)
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
		AdminLogger.Println(student.ID, "の成績確認タスクが追加された") // FIXME: for debug

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
				_, err := GetGradeAction(ctx, student.Agent)
				if err != nil {
					step.AddError(err)
					<-time.After(3000 * time.Millisecond)
					continue
				}
				step.AddScore(score.CountGetGrades)
				AdminLogger.Printf("%vは成績を確認した", student.ID)

				// 空きがあったら
				// 履修登録
				if student.RegisteredCoursesCount() < RegisterCourseLimit {
					// 5回成功するまでだと、失敗し続けた場合永遠に負荷がかかってしまう
					// 今は5回だけやって失敗したら終わりにした
					for i := 0; i < SearchCourseLimit; i++ {
						timer := time.After(300 * time.Millisecond)

						// TODO: verify course
						_, _, err := SearchCourseAction(ctx, student.Agent)

						if err != nil {
							step.AddError(err)
							<-timer
							continue
						}

						select {
						case <-ctx.Done():
							return
						case <-timer:
						}
					}
					AdminLogger.Printf("%vはコースを%v回検索した", student.ID, SearchCourseLimit)
					step.AddScore(score.CountSearchCourse)

					course := s.selectUnregisteredCourse(student)
					// TODO: verify response
					_, err := TakeCourseAction(ctx, student.Agent, course)
					if err != nil {
						step.AddError(err)
						return
					}
					step.AddScore(score.CountRegisterCourse)
					student.AddCourse(course)
					AdminLogger.Printf("%vは%vを履修した", student.ID, course.Name)
				}

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
		AdminLogger.Println(student.ID, "のおしらせタスクが追加された") // FIXME: for debug
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

				for _, ans := range announceList {
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
			// コースgoroutineは満員になるまではなにもしない
			<-course.WaitRegister(ctx)

			// コースの処理
			for i := 0; i < CourseProcessLimit; i++ {
				timer := time.After(100 * time.Millisecond)
				faculty := course.Faculty()

				// FIXME: verify class
				_, class, announcement, err := AddClassAction(ctx, faculty.Agent, course)
				if err != nil {
					step.AddError(err)
					<-timer
					continue
				}
				course.BroadCastAnnouncement(announcement)
				step.AddScore(score.CountAddClass)
				step.AddScore(score.CountAddAssignment)

				errs := submitAssignments(ctx, course.Students(), class, announcement.ID)
				for _, e := range errs {
					step.AddError(e)
				}

				// TODO: verify data
				_, assignmentsData, err := DownloadSubmissionsAction(ctx, faculty.Agent, class.ID)
				if err != nil {
					step.AddError(err)
					return
				}

				// TODO: 採点する
				errs = scoringAssignments(ctx, class.ID, faculty, course.Students(), assignmentsData)
				for _, e := range errs {
					step.AddError(e)
				}
				if len(errs) > 0 {
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
			newCourse := generate.Course()
			// コース追加Actionで成功したら
			// ベンチのコースタスクも増やす
			step.AddScore(score.CountAddCourse)
			s.addCourseLoad(newCourse)
			s.addCourseLoad(newCourse)

			// コースが追加されたのでベンチのアクティブ学生も増やす
			s.addActiveStudentLoads(1)
			return
		})
	})
	return loadCourseWorker
}

func (s *Scenario) addActiveStudentLoads(count int) {
	// どこまでメソッドをわけるか（s.Studentの管理）
	for i := 0; i < count; i++ {
		activetedStudent := s.student[0] // FIXME
		//<-time.After(time.Duration(rand.Intn(2000)) * time.Millisecond)
		s.sPubSub.Publish(activetedStudent)
	}
}

func (s *Scenario) addCourseLoad(course *model.Course) {
	// どこまでメソッドをわけるか（コース登録処理, s.Courseの管理）
	s.mu.Lock()
	s.courses = append(s.courses, course)
	s.mu.Unlock()

	s.cPubSub.Publish(course)
}

// studentに履修するメソッドをもたせてそこでコースを選ぶようにしてもいいかも知れない
func (s *Scenario) selectUnregisteredCourse(student *model.Student) *model.Course {

	// FIXME
	return s.courses[0]
}

func submitAssignments(ctx context.Context, students []*model.Student, class *model.Class, annoucementID string) []error {
	wg := sync.WaitGroup{}
	wg.Add(len(students))

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
			case <-s.WaitReadAnnouncement(annoucementID):
				// 学生sが課題お知らせを読むまで待つ
			}

			submission := generate.Submission()
			_, err := SubmitAssignmentAction(ctx, s.Agent, class.ID, submission)
			if err != nil {
				errs = append(errs, err)
			} else {
				s.AddSubmission(submission)
			}
		}()
	}
	wg.Wait()

	return errs
}

func scoringAssignments(ctx context.Context, classID string, faculty *model.Faculty, students []*model.Student, assignments []byte) []error {
	wg := sync.WaitGroup{}
	wg.Add(len(students))

	errs := make([]error, 0)
	for _, s := range students {
		s := s
		go func(ctx context.Context) {
			defer wg.Done()
			score := rand.Intn(101)
			_, err := PostGradeAction(ctx, faculty.Agent, classID, score, s.UserAccount.Code)
			if err != nil {
				errs = append(errs, err)
			}
		}(ctx)
	}
	wg.Wait()

	return errs
}
