package generate

import (
	"os"
	"testing"
)

func TestPDF(t *testing.T) {
	_ = os.WriteFile("sample.pdf", PDF("this is a sample pdf.\nnext line.", cyclicGetImage()), 0644)
	_ = os.WriteFile("sample.txt", PDF("this is a sample pdf.\nnext line.", cyclicGetImage()), 0644)
}
