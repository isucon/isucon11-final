package model

type ClassParam struct {
	Title     string
	Desc      string
	Part      uint8 // n回目のクラス
	CreatedAt int64
}

type Class struct {
	*ClassParam
	ID string
}

func NewClass(id string, param *ClassParam) *Class {
	return &Class{
		ClassParam: param,
		ID:         id,
	}
}
