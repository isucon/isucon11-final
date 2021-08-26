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
	"github.com/isucon/isucon11-final/benchmarker/scenario/util"
)

var (
	// Prepare, Load, Validationが返すエラー
	// Benchmarkが中断されたかどうか確認用
	Cancel failure.StringCode = "scenario-cancel"
)

type Scenario struct {
	Config

	sPubSub             *pubsub.PubSub
	cPubSub             *pubsub.PubSub
	courses             map[string]*model.Course
	emptyCourseManager  *util.CourseManager
	faculties           []*model.Teacher
	studentPool         *userPool
	activeStudents      []*model.Student // Poolから取り出された学生のうち、その後の検証を抜けてMyPageまでたどり着けた学生（goroutine数とイコール）
	finishedCourseCount int              // FIXME Debug
	language            string
	loadRequestEndTime  time.Time

	mu sync.RWMutex
}

type Config struct {
	BaseURL *url.URL
	UseTLS  bool
	NoLoad  bool
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
		Config: *config,

		sPubSub:            pubsub.NewPubSub(),
		cPubSub:            pubsub.NewPubSub(),
		courses:            map[string]*model.Course{},
		emptyCourseManager: util.NewCourseManager(),
		faculties:          faculties,
		studentPool:        NewUserPool(studentsData),
		activeStudents:     make([]*model.Student, 0, initialStudentsCount),
	}, nil
}

func (s *Scenario) Language() string {
	return s.language
}

func (s *Scenario) ActiveStudents() []*model.Student {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.activeStudents
}

func (s *Scenario) AddActiveStudent(student *model.Student) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.activeStudents = append(s.activeStudents, student)
}
func (s *Scenario) ActiveStudentCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.activeStudents)
}

func (s *Scenario) CourseCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.courses)
}
func (s *Scenario) FinishedCourseCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.finishedCourseCount
}

func (s *Scenario) AddCourse(course *model.Course) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.courses[course.ID] = course
}

func (s *Scenario) GetCourse(id string) (*model.Course, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	course, exists := s.courses[id]
	return course, exists
}

func (s *Scenario) Courses() map[string]*model.Course {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.courses
}

func (s *Scenario) GetRandomTeacher() *model.Teacher {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.faculties[rand.Intn(len(s.faculties))]
}
