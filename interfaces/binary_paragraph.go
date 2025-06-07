package interfaces

import "image"

type BinaryParagraph interface {
	GetBitmap() image.Image
	IsForced() bool
}
