package model

type Submission struct {
	Title string
	Data  []byte
}

func NewSubmission(title string, data []byte) *Submission {
	return &Submission{title, data}
}
