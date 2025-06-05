package bluraysup

import "github.com/RistRyder/gse/common"

type OdsData struct {
	Fragment      *ImageObjectFragment
	IsFirst       bool
	Message       string
	ObjectId      int
	ObjectVersion int
	Size          common.Size
}
