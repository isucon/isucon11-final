package generate

import (
	"bufio"
	"bytes"
	_ "embed"
	"strings"

	"github.com/isucon/isucon11-final/benchmarker/model"
)

var (
	//go:embed data/student.tsv
	studentsData []byte
	//go:embed data/teacher.tsv
	teachersData []byte
)

func LoadStudentsData() ([]*model.UserAccount, error) {
	return loadUserAccountData(studentsData, false)
}

func LoadTeachersData() ([]*model.UserAccount, error) {
	return loadUserAccountData(teachersData, true)
}

func loadUserAccountData(data []byte, isAdmin bool) ([]*model.UserAccount, error) {
	userDataSet := make([]*model.UserAccount, 0)
	s := bufio.NewScanner(bytes.NewReader(data))
	for s.Scan() {
		line := strings.Split(s.Text(), "\t")
		account := &model.UserAccount{
			ID:          line[0],
			Code:        line[1],
			Name:        line[2],
			RawPassword: line[3],
			IsAdmin:     isAdmin,
		}
		userDataSet = append(userDataSet, account)
	}
	return userDataSet, nil
}
