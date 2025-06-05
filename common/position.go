package common

type Position struct {
	Left int
	Top  int
}

func NewPosition(left, top int) *Position {
	return &Position{Left: left, Top: top}
}
