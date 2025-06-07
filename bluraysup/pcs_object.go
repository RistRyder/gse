package bluraysup

import "image"

type PcsObject struct {
	IsForced bool
	ObjectId int
	Origin   image.Point
	WindowId int
}
