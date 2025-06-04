package matroska

import "fmt"

type Element struct {
	DataPosition int64
	DataSize     int64
	Id           ElementId
}

type ElementId uint32

const (
	ElementNone ElementId = 0

	ElementEbml    ElementId = 0x1A45DFA3
	ElementSegment ElementId = 0x18538067

	ElementInfo          ElementId = 0x1549A966
	ElementTimecodeScale ElementId = 0x2AD7B1
	ElementDuration      ElementId = 0x4489

	ElementTracks      ElementId = 0x1654AE6B
	ElementTrackEntry  ElementId = 0xAE
	ElementTrackNumber ElementId = 0xD7
	ElementTrackType   ElementId = 0x83
	ElementFlagDefault ElementId = 0x88
	ElementFlagForced  ElementId = 0x55AA

	ElementDefaultDuration      ElementId = 0x23E383
	ElementName                 ElementId = 0x536E
	ElementLanguage             ElementId = 0x22B59C
	ElementCodecId              ElementId = 0x86
	ElementCodecPrivate         ElementId = 0x63A2
	ElementVideo                ElementId = 0xE0
	ElementPixelWidth           ElementId = 0xB0
	ElementPixelHeight          ElementId = 0xBA
	ElementAudio                ElementId = 0xE1
	ElementContentEncodings     ElementId = 0x6D80
	ElementContentEncoding      ElementId = 0x6240
	ElementContentEncodingOrder ElementId = 0x5031
	ElementContentEncodingScope ElementId = 0x5032
	ElementContentEncodingType  ElementId = 0x5033
	ElementContentCompression   ElementId = 0x5034
	ElementContentCompAlgo      ElementId = 0x4254
	ElementContentCompSettings  ElementId = 0x4255

	ElementCluster       ElementId = 0x1F43B675
	ElementTimecode      ElementId = 0xE7
	ElementSimpleBlock   ElementId = 0xA3
	ElementBlockGroup    ElementId = 0xA0
	ElementBlock         ElementId = 0xA1
	ElementBlockDuration ElementId = 0x9B

	ElementChapters         ElementId = 0x1043A770
	ElementEditionEntry     ElementId = 0x45B9
	ElementChapterAtom      ElementId = 0xB6
	ElementChapterTimeStart ElementId = 0x91
	ElementChapterDisplay   ElementId = 0x80
	ElementChapString       ElementId = 0x85
)

func (e *Element) EndPosition() int64 {
	return e.DataPosition + e.DataSize
}

func NewElement(id ElementId, dataPosition int64, dataSize int64) *Element {
	return &Element{DataPosition: dataPosition, DataSize: dataSize, Id: id}
}

func (e *Element) String() string {
	return fmt.Sprintf("%d (%d)", e.Id, e.DataSize)
}
