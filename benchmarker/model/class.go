package model

type Class struct {
	ID        string
	Title     string
	Desc      string
	Part      int // n回目のクラス
	CreatedAt int64
}

func NewClass(id, title, desc string, createdAt int64, part int) *Class {
	return &Class{
		ID:        id,
		Title:     title,
		Desc:      desc,
		Part:      part,
		CreatedAt: createdAt,
	}
}
