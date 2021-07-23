package generate

import (
	"bufio"
	"os"
	"strings"

	"github.com/isucon/isucon11-final/benchmarker/model"
)

var (
	studentFile = "./generate/data/student.tsv"
)

func LoadFaculty() *model.UserAccount {
	return &model.UserAccount{
		Code:        "F0000001",
		Name:        "椅子昆",
		RawPassword: "password",
	}
}

func LoadStudentsData() ([]*model.UserAccount, error) {
	file, err := os.Open(studentFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	studentDataSet := make([]*model.UserAccount, 0)
	s := bufio.NewScanner(file)
	for i := 0; s.Scan(); i++ {
		line := strings.Split(s.Text(), "\t")
		code := line[0]
		name := line[1]
		rawPW := line[2]

		account := &model.UserAccount{
			Code:        code,
			Name:        name,
			RawPassword: rawPW,
		}
		studentDataSet = append(studentDataSet, account)
	}

	return studentDataSet, nil
}
