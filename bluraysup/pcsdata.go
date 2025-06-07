package bluraysup

import (
	"image"
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

func (p *PcsData) GetBitmap() image.Image {
	if len(p.PcsObjects) == 1 {
		return DecodeImage(p.PcsObjects[0], p.BitmapObjects[0], p.PaletteInfos)
	}

	r := image.Rect(0, 0, 0, 0)
	for ioIndex := 0; ioIndex < len(p.PcsObjects); ioIndex++ {
		if ioIndex < len(p.BitmapObjects) {
			ioRect := image.Rect(p.PcsObjects[ioIndex].Origin.X, p.PcsObjects[ioIndex].Origin.Y, p.BitmapObjects[ioIndex][0].Size.Width, p.BitmapObjects[ioIndex][0].Size.Height)
			if r.Empty() {
				r = ioRect
			} else {
				r = ioRect.Union(r)
			}
		}
	}

	mergedBmp := image.NewRGBA(r)
	for ioIndex := 0; ioIndex < len(p.PcsObjects); ioIndex++ {
		if ioIndex < len(p.BitmapObjects) {
			offset := p.PcsObjects[ioIndex].Origin.Sub(r.Size())
			singleBmp := DecodeImage(p.PcsObjects[ioIndex], p.BitmapObjects[ioIndex], p.PaletteInfos)
			destinationRect := image.Rectangle{offset, offset.Add(singleBmp.Bounds().Size())}

			draw.Over.Draw(mergedBmp, destinationRect, singleBmp, singleBmp.Bounds().Min)
		}
	}

	return mergedBmp
}

func (p *PcsData) IsForced() bool {
	for _, pcsObject := range p.PcsObjects {
		if pcsObject.IsForced {
			return true
		}
	}

	return false
}
