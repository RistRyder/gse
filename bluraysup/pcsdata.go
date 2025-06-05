package bluraysup

import "github.com/RistRyder/gse/common"

type PcsData struct {
	BitmapObjects       [][]OdsData
	CompNum             int
	CompositionState    CompositionState
	EndTime             int64
	FramesPerSecondType int
	PaletteId           int
	PaletteInfos        []PaletteInfo
	PaletteUpdate       bool
	PcsObjects          []PcsObject
	Size                common.Size
	StartTime           int64
}
