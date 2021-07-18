package scenario

import (
	"context"
	"sync"
	"time"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/parallel"
	"github.com/isucon/isucon11-final/benchmarker/generate"
	"github.com/isucon/isucon11-final/benchmarker/model"
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


		studentLoadWorker.Do(func(ctx context.Context) {
			for ctx.Err() == nil {
				// 学生は成績を確認し続ける
				// TODO: verify grade
				_, err := GetGradeAction(ctx, student.Agent)
				if err != nil {
					step.AddError(err)
					return
				}
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
							return
						}

						select {
						case <-ctx.Done():
							return
						case <-timer:
						}
					}
					AdminLogger.Printf("%vはコースを%v回検索した", student.ID, SearchCourseLimit)

					course := s.selectUnregisteredCourse(student)
					// TODO: verify response
					_, err := TakeCourseAction(ctx, student.Agent, course)
					if err != nil {
						step.AddError(err)
						return
					}
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
			for ctx.Err() == nil {
				// 学生はお知らせを確認し続ける
				// TODO: verify announcement
				_, _, err := GetAnnouncementAction(ctx, student.Agent)
				if err != nil {
					step.AddError(err)
					return
				}

				AdminLogger.Println(student.ID, "はお知らせを確認した") // FIXME: for debug
				// 未読があったら
				// 内容を確認
				if student.HasUnreadAnnouncement() {
					unreadAnnouncement := student.PopOldestUnreadAnnouncements()
					// TODO: verify announcement detail
					_, _, err := GetAnnouncementDetailAction(ctx, student.Agent, unreadAnnouncement.ID)
					if err != nil {
						step.AddError(err)
						// unreadに戻しておく
						student.PushOldestUnreadAnnouncements(unreadAnnouncement)
						return
					}
					AdminLogger.Printf("%vは未読のお知らせを確認した", student.ID) // FIXME: for debug
				}

				select {
				case <-ctx.Done():
					return
				case <-time.After(3000 * time.Millisecond):
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
			for ctx.Err() == nil {
				AdminLogger.Println(course.Name, "は定員チェックをした") // FIXME: for debug
				//<-time.After(1 * time.Second)                  // FIXME: for debug

				// if course.IsFull() {
				// 		break
				// }
				//<-time.After(300 * time.Millisecond)
				break // FIXME: for debug
			}

			// コースの処理
			<-time.After(5 * time.Second)

			// コースがおわった
			AdminLogger.Println(course.Name, "は終了した")

			// コースを追加
			newCourse := generate.Course()
			// コース追加Actionで成功したら
			// ベンチのコースタスクも増やす
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
