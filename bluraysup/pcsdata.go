package bluraysup

import (
	"image/draw"

	"github.com/RistRyder/gse/common"
)

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

func (p *PcsData) GetBitmap() *draw.Image {
	//TODO: Do
	return nil
}

func (p *PcsData) IsForced() bool {
	for _, pcsObject := range p.PcsObjects {
		if pcsObject.IsForced {
			return true
		}
	}

	return false
}
