package interfaces

import "image/draw"

type BinaryParagraph interface {
	GetBitmap() *draw.Image
	IsForced() bool
}
