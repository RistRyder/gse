package interfaces

import "github.com/RistRyder/gse/common"

type BinaryParagraphWithPosition interface {
	BinaryParagraph
	EndTimeCode() common.TimeCode
	Position() common.Position
	ScreenSize() common.Size
	StartTimeCode() common.TimeCode
}
