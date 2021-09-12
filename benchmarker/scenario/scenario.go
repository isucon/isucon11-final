package scenario

import (
	"net/url"
	"sync"
	"time"

	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucandar/pubsub"

	"github.com/isucon/isucon11-final/benchmarker/generate"
	"github.com/isucon/isucon11-final/benchmarker/model"
)

var (
	// Prepare, Load, Validationが返すエラー
	// Benchmarkが中断されたかどうか確認用
	Cancel failure.StringCode = "scenario-cancel"
)

type Scenario struct {
	Config
	CapacityCounter *model.CapacityCounter
	CourseManager   *model.CourseManager

	sPubSub             *pubsub.PubSub
	cPubSub             *pubsub.PubSub
	userPool            *userPool
	activeStudents      []*model.Student // Poolから取り出された学生のうち、その後の検証を抜けてMyPageまでたどり着けた学生（goroutine数とイコール）
	activeStudentsCount int64
	language            string
	loadRequestEndTime  time.Time
	debugData           *DebugData

	// initCourses は/initializeで追加されるコース
	// 中のデータの更新はしないこと
	initCourses []*model.Course

	rmu sync.RWMutex

	finishCoursePubSub        *pubsub.PubSub
	finishCourseStudentsCount int64
}

type Config struct {
	BaseURL          *url.URL
	UseTLS           bool
	NoLoad           bool
	NoPrepare        bool
	NoVerifyResource bool
	IsDebug          bool
}

func NewScenario(config *Config) (*Scenario, error) {
	studentsData := generate.LoadStudentsData()
	teachersData := generate.LoadTeachersData()

	teachers := make([]*model.Teacher, len(teachersData))
	teachersMap := make(map[string]*model.Teacher, len(teachersData))
	for i, account := range teachersData {
		teacher := model.NewTeacher(account, config.BaseURL)
		teachers[i] = teacher
		teachersMap[account.ID] = teacher
	}

	// 動作確認用科目の教員アカウント
	// ベンチではこのアカウントを操作することはないためUserAccountのみを管理する
	testTeacher := &model.Teacher{
		UserAccount: &model.UserAccount{
			ID:          "01FF4RXEKS0DG2EG20CKDWS7CC",
			Code:        "T99999",
			Name:        "isucon-teacher",
			RawPassword: "isucon",
		},
	}
	teachersMap[testTeacher.ID] = testTeacher

	cc := model.NewCapacityCounter()
	initCourses := generate.LoadInitialCourseData(teachersMap, StudentCapacityPerCourse, cc)

	return &Scenario{
		Config:          *config,
		CapacityCounter: cc,
		CourseManager:   model.NewCourseManager(cc),

		sPubSub:            pubsub.NewPubSub(),
		cPubSub:            pubsub.NewPubSub(),
		userPool:           NewUserPool(studentsData, teachers, config.BaseURL),
		activeStudents:     make([]*model.Student, 0, initialStudentsCount),
		debugData:          NewDebugData(config.IsDebug),
		finishCoursePubSub: pubsub.NewPubSub(),
		initCourses:        initCourses,
	}, nil
}

func (s *Scenario) Language() string {
	return s.language
}

func (s *Scenario) ActiveStudents() []*model.Student {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	return s.activeStudents
}

func (s *Scenario) AddActiveStudent(student *model.Student) {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	s.activeStudents = append(s.activeStudents, student)
}
func (s *Scenario) ActiveStudentCount() int {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	return len(s.activeStudents)
}

func (s *Scenario) Reset() {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	studentsData := generate.LoadStudentsData()
	teachersData := generate.LoadTeachersData()

	teachers := make([]*model.Teacher, len(teachersData))
	teachersMap := make(map[string]*model.Teacher, len(teachersData))
	for i, account := range teachersData {
		teacher := model.NewTeacher(account, s.BaseURL)
		teachers[i] = teacher
		teachersMap[account.ID] = teacher
	}

	testTeacher := &model.Teacher{
		UserAccount: &model.UserAccount{
			ID:          "01FF4RXEKS0DG2EG20CKDWS7CC",
			Code:        "T99999",
			Name:        "isucon-teacher",
			RawPassword: "isucon",
		},
	}
	teachersMap[testTeacher.ID] = testTeacher

	cc := model.NewCapacityCounter()
	initCourses := generate.LoadInitialCourseData(teachersMap, StudentCapacityPerCourse, cc)

	s.CapacityCounter = cc
	s.CourseManager = model.NewCourseManager(s.CapacityCounter)
	s.sPubSub = pubsub.NewPubSub()
	s.cPubSub = pubsub.NewPubSub()
	s.userPool = NewUserPool(studentsData, teachers, s.BaseURL)
	s.activeStudents = make([]*model.Student, 0, initialStudentsCount)
	s.debugData = NewDebugData(s.Config.IsDebug)
	s.finishCoursePubSub = pubsub.NewPubSub()
	s.finishCourseStudentsCount = 0
	s.initCourses = initCourses
}
