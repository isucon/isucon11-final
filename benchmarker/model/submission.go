package model

type Submission struct {
	title string
	data  []byte
}

func NewSubmission(title string, data []byte) *Submission {
	return &Submission{title, data}
}
