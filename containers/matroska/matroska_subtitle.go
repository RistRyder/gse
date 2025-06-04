package matroska

import (
	"bytes"
	"compress/zlib"
	"io"

	"github.com/andybalholm/crlf"
	"github.com/cockroachdb/errors"
)

type MatroskaSubtitle struct {
	Data     []byte
	Duration int64
	Start    int64
}

func (m *MatroskaSubtitle) End() int64 {
	return m.Start + m.Duration
}

func NewMatroskaSubtitle(data []byte, start int64) *MatroskaSubtitle {
	return &MatroskaSubtitle{Data: data, Start: start}
}

func (m *MatroskaSubtitle) Text(matroskaTrackInfo *MatroskaTrackInfo) (string, error) {
	uncompressedData, uncompressedDataErr := m.UncompressedData(matroskaTrackInfo)
	if uncompressedDataErr != nil {
		return "", uncompressedDataErr
	}

	if uncompressedData == nil {
		return "", nil
	}

	//terminate string at first binary zero - https://github.com/Matroska-Org/ebml-specification/blob/master/specification.markdown#terminating-elements
	max := len(uncompressedData)
	for i := 0; i < max; i++ {
		if uncompressedData[i] == 0 {
			max = i
			break
		}
	}

	normalizedData := make([]byte, max)

	//The original .NET libse replaces all newlines with platform-specific newlines, but
	// here we simply turn everything into "\n"
	normalizer := new(crlf.Normalize)
	normalizer.Transform(normalizedData, uncompressedData[:max], true)

	text := string(normalizedData)

	return text, nil
}

func (m *MatroskaSubtitle) UncompressedData(matroskaTrackInfo *MatroskaTrackInfo) ([]byte, error) {
	if matroskaTrackInfo.ContentEncodingType != ContentEncodingTypeCompression || (matroskaTrackInfo.ContentEncodingScope&ContentEncodingScopeTracks) == 0 {
		return m.Data, nil
	}

	buffer := bytes.NewBuffer(m.Data)
	zlibReader, zlibReaderErr := zlib.NewReader(buffer)
	if zlibReaderErr != nil {
		return nil, errors.Wrap(zlibReaderErr, "failed to create zlib reader")
	}

	uncompressedData, uncompressedDataErr := io.ReadAll(zlibReader)
	if uncompressedDataErr != nil {
		return nil, errors.Wrap(uncompressedDataErr, "failed to read all data from zlib reader")
	}

	return uncompressedData, nil
}
