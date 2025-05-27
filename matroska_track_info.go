package gse

import "fmt"

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
	return fmt.Sprintf("Codec: %v , Duration: %v , Name: %v , Language: %v , Subtitle? %v , Video? %v", m.CodecId, m.DefaultDuration, m.Name, m.Language, m.IsSubtitle, m.IsVideo)
}
