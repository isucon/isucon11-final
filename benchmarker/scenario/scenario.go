package scenario

import (
	"math/rand"
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
	CourseManager *model.CourseManager

	sPubSub            *pubsub.PubSub
	cPubSub            *pubsub.PubSub
	faculties          []*model.Teacher
	studentPool        *userPool
	activeStudents     []*model.Student // Poolから取り出された学生のうち、その後の検証を抜けてMyPageまでたどり着けた学生（goroutine数とイコール）
	language           string
	loadRequestEndTime time.Time
	debugData          *DebugData

	rmu sync.RWMutex

	finishCoursePubSub        *pubsub.PubSub
	finishCourseStudentsCount int64
}

type Config struct {
	BaseURL *url.URL
	UseTLS  bool
	NoLoad  bool
	IsDebug bool
}

func NewScenario(config *Config) (*Scenario, error) {
	studentsData, err := generate.LoadStudentsData()
	if err != nil {
		return nil, err
	}
	facultiesData, err := generate.LoadFacultiesData()
	if err != nil {
		return nil, err
	}

	faculties := make([]*model.Teacher, len(facultiesData))
	for i, f := range facultiesData {
		faculties[i] = model.NewTeacher(f, config.BaseURL)
	}

	return &Scenario{
		Config:        *config,
		CourseManager: model.NewCourseManager(),

		sPubSub:            pubsub.NewPubSub(),
		cPubSub:            pubsub.NewPubSub(),
		faculties:          faculties,
		studentPool:        NewUserPool(studentsData),
		activeStudents:     make([]*model.Student, 0, initialStudentsCount),
		debugData:          NewDebugData(config.IsDebug),
		finishCoursePubSub: pubsub.NewPubSub(),
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

func (s *Scenario) GetRandomTeacher() *model.Teacher {
	s.rmu.RLock()
	defer s.rmu.RUnlock()

	return s.faculties[rand.Intn(len(s.faculties))]
}

func (s *Scenario) Reset() {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	studentsData, err := generate.LoadStudentsData()
	if err != nil {
		panic(err)
	}
	facultiesData, err := generate.LoadFacultiesData()
	if err != nil {
		panic(err)
	}

	faculties := make([]*model.Teacher, len(facultiesData))
	for i, f := range facultiesData {
		faculties[i] = model.NewTeacher(f, s.Config.BaseURL)
	}

	s.CourseManager = model.NewCourseManager()
	s.sPubSub = pubsub.NewPubSub()
	s.cPubSub = pubsub.NewPubSub()
	s.faculties = faculties
	s.studentPool = NewUserPool(studentsData)
	s.activeStudents = make([]*model.Student, 0, initialStudentsCount)
	s.debugData = NewDebugData(s.Config.IsDebug)
	s.finishCoursePubSub = pubsub.NewPubSub()
	s.finishCourseStudentsCount = 0
}
