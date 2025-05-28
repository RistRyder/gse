package gse

import "fmt"

const (
	ContentEncodingScopePrivateData = 2
	ContentEncodingScopeTracks      = 1
	ContentEncodingTypeCompression  = 0
)

type MatroskaTrackInfo struct {
	CodecId                     string
	ContentCompressionAlgorithm int
	ContentEncodingScope        uint
	ContentEncodingType         int
	DefaultDuration             int
	IsAudio                     bool
	IsDefault                   bool
	IsForced                    bool
	IsSubtitle                  bool
	IsVideo                     bool
	Language                    string
	Name                        string
	TrackNumber                 int
	Uid                         string
}

func (m *MatroskaTrackInfo) String() string {
	return fmt.Sprintf("Codec: %v , ContentCompressionAlgorithm: %v, ContentEncodingScope: %v , ContentEncodingType: %v , Duration: %v , Name: %v , Language: %v , Subtitle? %v , Video? %v", m.CodecId, m.ContentCompressionAlgorithm, m.ContentEncodingScope, m.ContentEncodingType, m.DefaultDuration, m.Name, m.Language, m.IsSubtitle, m.IsVideo)
}
