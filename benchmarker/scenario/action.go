package scenario

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/isucon/isucandar/agent"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucon11-final/benchmarker/api"
	"github.com/isucon/isucon11-final/benchmarker/fails"
	"github.com/isucon/isucon11-final/benchmarker/generate"
	"github.com/isucon/isucon11-final/benchmarker/model"
)

func InitializeAction(ctx context.Context, agent *agent.Agent) (string, error) {
	language, err := api.Initialize(ctx, agent)
	if err != nil {
		return "", err
	}
	if language == "" {
		return "", failure.NewError(fails.ErrCritical, fmt.Errorf("実装言語が返却されていません"))
	}

	return language, nil
}

func LoginAction(ctx context.Context, agent *agent.Agent, u *model.UserData) []error {
	errs := api.AccessLoginPage(ctx, agent)
	if len(errs) > 0 {
		return errs
	}

	err := api.Login(ctx, agent, u.Number, u.RawPassword)
	if err != nil {
		return []error{err}
	}

	return nil
}

func SearchCoursesAction(ctx context.Context, agent *agent.Agent, course *model.Course) []error {
	syllabusIDs, err := api.SearchSyllabus(ctx, agent, course.Keyword[0])
	if err != nil {
		return []error{err}
	}
	// FIXME: pagingが実装されてないので修正

	var isContain bool
	for _, id := range syllabusIDs {
		if id == course.ID {
			isContain = true
		}
	}

	if !isContain {
		err := failure.NewError(fails.ErrApplication, fmt.Errorf(
			"検索結果に期待する講義が含まれませんでした: 講義(%s), 検索キーワード(%s)",
			course.Name, course.Keyword[0]),
		)
		return []error{err}
	}

	if errs := api.AccessSyllabusPage(ctx, agent, course.ID); len(errs) > 0 {
		return errs
	}
	return nil
}

func RegisterCoursesAction(ctx context.Context, student *model.Student, courses []*model.Course) error {
	coursesID := make([]string, 0, len(courses))
	for _, c := range courses {
		coursesID = append(coursesID, c.ID)
	}

	registeredCoursesID, err := api.RegisterCourses(ctx, student.Agent, coursesID)
	if err != nil {
		return err
	}
	// nolint:staticcheck
	if len(registeredCoursesID) == 0 {
		// FIXME:登録失敗した講義を除いて再登録したい
	}

	student.AddCourses(coursesID)
	for _, c := range courses {
		c.AddStudent(student)
	}

	return nil
}

func FetchRegisteredCoursesAction(ctx context.Context, student *model.Student) ([]string, error) {
	registeredCoursesID, err := api.FetchRegisteredCourses(ctx, student.Agent)
	if err != nil {
		return nil, err
	}

	return registeredCoursesID, nil
}

func AddClass(ctx context.Context, faculty *model.Faculty, course *model.Course) (*model.Class, error) {
	title, desc := generate.ClassDetail(course)

	id, err := api.AddClass(ctx, faculty.Agent, course.ID, title, desc)
	if err != nil {
		return nil, err
	}

	class := model.NewClass(id, course.ID, title, desc)
	course.AddHeldClasses(class)
	return class, nil
}

func AddClassAnnouncement(ctx context.Context, faculty *model.Faculty, course *model.Course, class *model.Class) error {
	title, message := generate.Announcement(class.Title)
	id, err := api.AddAnnouncement(ctx, faculty.Agent, class.CourseID, title, message)
	if err != nil {
		return err
	}

	anc := model.NewAnnouncement(id, title, message)
	class.AddAnnouncement(anc)
	for _, student := range course.Students() {
		err := student.AddAnnouncement(id, anc)
		if err != nil {
			return failure.NewError(fails.ErrCritical, err)
		}
	}
	return nil
}

// classに紐付いたお知らせを確認するアクション classに紐付いたお知らせ以外は触らない
func CheckClassAnnouncementAction(ctx context.Context, student *model.Student, class *model.Class) error {
	announcementList, err := api.FetchAnnouncements(ctx, student.Agent)
	if err != nil {
		return err
	}
	announcementsByID := map[string]*api.AnnouncementsResponse{}
	for _, anc := range announcementList {
		if announcementsByID[anc.ID] != nil {
			err := failure.NewError(fails.ErrCritical, fmt.Errorf(
				"重複したお知らせが返却されました: お知らせID(%s)", anc.ID,
			))
			return err
		}
		announcementsByID[anc.ID] = anc
	}

	classAnnouncements := class.Announcement()
	for _, classAnnouncement := range classAnnouncements {
		if announcementsByID[classAnnouncement.ID] == nil {
			err := failure.NewError(fails.ErrCritical, fmt.Errorf(
				"登録したお知らせが存在しませんでした: お知らせID(%s)", classAnnouncement.ID,
			))
			return err
		}

		isUnread := !student.IsReadAnnouncement(classAnnouncement.ID)
		if isUnread != announcementsByID[classAnnouncement.ID].Unread {
			err := failure.NewError(fails.ErrCritical, fmt.Errorf(
				"お知らせの未読状態が不正です: お知らせID(%s)", classAnnouncement.ID,
			))
			return err
		}

		// 未読状態だったら既読にして中身を確認する
		if !isUnread {
			continue
		}
		detail, err := api.FetchAnnouncementDetail(ctx, student.Agent, classAnnouncement.ID)
		if err != nil {
			return err
		}
		student.AddReadAnnouncement(classAnnouncement.ID)

		expect := student.AnnouncementByID(classAnnouncement.ID)
		if detail.ID != expect.ID || detail.Title != expect.Title || detail.Message != expect.Message {
			err := failure.NewError(fails.ErrCritical, fmt.Errorf(
				"登録されたお知らせが一致しません: お知らせID(%s) タイトル(%s) 内容(%s)",
				detail.ID, detail.Title, detail.Message,
			))
			return err
		}
	}
	return nil
}

func AddClassDocument(ctx context.Context, faculty *model.Faculty, class *model.Class) error {
	fileName := class.Title + "_資料.pdf"
	uploadFile := generate.DocumentFile()
	id, err := api.AddDocument(ctx, faculty.Agent, class.CourseID, class.ID, fileName, uploadFile)
	if err != nil {
		return err
	}

	err = class.AddDocHash(id, getHash(uploadFile))
	if err != nil {
		return err
	}
	return nil
}

// pdfの検証はしない
func CheckClassDoc(ctx context.Context, student *model.Student, class *model.Class) error {
	docIDs, err := api.FetchDocumentIDList(ctx, student.Agent, class.CourseID, class.ID)
	if err != nil {
		return nil
	}
	if !class.EqualDocumentIDs(docIDs) {
		return failure.NewError(fails.ErrCritical, fmt.Errorf(
			"登録されている講義資料が一致しません: コース名(%s) クラス名(%s)",
			class.CourseID, class.Title,
		))
	}

	for _, docID := range docIDs {
		_, err := api.FetchDocument(ctx, student.Agent, class.CourseID, docID)
		if err != nil {
			return err
		}
	}
	return nil
}

// pdfファイルのHash比較もする
func VerifyClassDoc(ctx context.Context, student *model.Student, class *model.Class) error {
	docIDs, err := api.FetchDocumentIDList(ctx, student.Agent, class.CourseID, class.ID)
	if err != nil {
		return nil
	}
	if !class.EqualDocumentIDs(docIDs) {
		return failure.NewError(fails.ErrCritical, fmt.Errorf(
			"登録されている講義資料が一致しません: コース名(%s) クラス名(%s)",
			class.CourseID, class.Title,
		))
	}

	for _, docID := range docIDs {
		docData, err := api.FetchDocument(ctx, student.Agent, class.CourseID, docID)
		if err != nil {
			return err
		}
		if !class.HasDocumentHash(docID, getHash(docData)) {
			err := failure.NewError(fails.ErrCritical, fmt.Errorf(
				"講義資料のデータが一致しません: コース名(%s) クラス名(%s) docID(%s)",
				class.CourseID, class.Title, docID,
			))
			return err
		}
	}
	return nil
}

func GetAttendanceCode(ctx context.Context, faculty *model.Faculty, class *model.Class) (string, error) {
	code, err := api.GetAttendanceCode(ctx, faculty.Agent, class.CourseID, class.ID)
	if err != nil {
		return "", err
	}
	return code, nil
}
func PostAttendanceCode(ctx context.Context, student *model.Student, class *model.Class, code string) error {
	err := api.PostAttendance(ctx, student.Agent, code)
	if err != nil {
		return err
	}

	class.AddAttendedStudentsID(student.Number)
	return nil
}

func AddClassAssignments(ctx context.Context, faculty *model.Faculty, class *model.Class) error {
	name, desc := generate.Assignment(class.Title)
	deadline := time.Now().Add(90 * time.Second)
	code, err := api.AddAssignments(ctx, faculty.Agent, class.CourseID, class.ID, name, desc, deadline.UnixNano())
	if err != nil {
		return err
	}

	class.AddAssignmentID(code)
	return nil
}
func SubmitAssignment(ctx context.Context, student *model.Student, class *model.Class) error {
	title, submission := generate.Submission()
	err := api.SubmitAssignment(ctx, student.Agent, class.CourseID, class.AssignmentID(), title, submission)
	if err != nil {
		return err
	}

	// サーバ側でzipに固められたときのファイル名
	fileName := fmt.Sprintf("%s-%s", student.Number, title)
	class.AddSubmission(fileName, getHash(submission))
	return nil
}

func VerifyAttendances(ctx context.Context, faculty *model.Faculty, class *model.Class) error {
	attendedStudentIDs, err := api.GetAttendanceStudentIDs(ctx, faculty.Agent, class.CourseID, class.CourseID)
	if err != nil {
		return err
	}

	if len(attendedStudentIDs) != class.AttendedStudentsIDCount() {
		return failure.NewError(fails.ErrCritical, fmt.Errorf("出席した学生が一致しません"))
	}
	for _, studentID := range attendedStudentIDs {
		if !class.IsAttendedByStudentsID(studentID) {
			return failure.NewError(fails.ErrCritical, fmt.Errorf("出席した学生が一致しません"))
		}
	}
	return nil
}
func VerifySubmissions(ctx context.Context, faculty *model.Faculty, class *model.Class) error {
	zipData, err := api.ExportSubmissions(ctx, faculty.Agent, class.CourseID, class.AssignmentID())
	if err != nil {
		return err
	}

	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return failure.NewError(fails.ErrApplication, err)
	}

	submissions := class.Submissions()
	for fileName, hash := range submissions {
		var isContain bool
		for _, f := range zr.File {
			if f.Name == fileName {
				isContain = true

				fileData, err := readZipFileData(f)
				if err != nil {
					return failure.NewError(fails.ErrApplication, err)
				}
				if !bytes.Equal(hash, getHash(fileData)) {
					return failure.NewError(fails.ErrCritical, fmt.Errorf("提出した課題が一致しません"))
				}
			}
		}
		if !isContain {
			return failure.NewError(fails.ErrCritical, fmt.Errorf("提出した課題がzipに含まれていません"))
		}
	}
	return nil
}
func readZipFileData(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	buf := make([]byte, file.UncompressedSize64)
	_, err = io.ReadFull(rc, buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func RegisterGradeAction(ctx context.Context, faculty *model.Faculty, student *model.Student, courseID string) error {
	var grade uint32 = 1
	err := api.RegisterGrades(ctx, faculty.Agent, courseID, student.UserData.Number, grade)
	if err != nil {
		return err
	}
	student.SetGradesUnchecked(courseID, grade)
	return nil
}

func FetchGradesAction(ctx context.Context, student *model.Student) (map[string]uint32, error) {
	r, err := api.GetGrades(ctx, student.Agent, student.Number)
	mp := make(map[string]uint32, len(r.CourseGrades))
	for _, courseGrade := range r.CourseGrades {
		mp[courseGrade.ID] = courseGrade.Grade
	}

	if err != nil {
		return nil, err
	}
	return mp, nil
}

// 他のアクションに付随しないページアクセス
func AccessMyPageAction(ctx context.Context, agent *agent.Agent) []error {
	return api.AccessMyPage(ctx, agent)
}
func AccessRegPageAction(ctx context.Context, agent *agent.Agent) []error {
	return api.AccessCourseRegPage(ctx, agent)
}
