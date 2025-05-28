package gse

import (
	"fmt"
	"io"
	"os"
)

type MatroskaFile struct {
	Duration          float64
	File              *os.File
	FilePosition      int64
	FileSize          int64
	FrameRate         float64
	IsValid           bool
	MatroskaSubtitles []*MatroskaSubtitle
	Path              string
	PixelHeight       int
	PixelWidth        int
	SegmentElement    *Element
	TimeCodeScale     int64
	Tracks            []*MatroskaTrackInfo
	VideoCodecId      string
}

func (m *MatroskaFile) offsetFilePosition(offset int) {
	m.FilePosition += int64(offset)
}

func (m *MatroskaFile) scaleTime32(time float32) float64 {
	return float64(time) * float64(m.TimeCodeScale) / 1000000.0
}

func (m *MatroskaFile) scaleTime64(time float64) float64 {
	return time * float64(m.TimeCodeScale) / 1000000.0
}

func (m *MatroskaFile) Close() error {
	return m.File.Close()
}

func NewMatroskaFile(path string) (*MatroskaFile, error) {
	file, openErr := os.Open(path)
	if openErr != nil {
		return nil, fmt.Errorf("failed to open Matroska file %s: %w", path, openErr)
	}

	matroskaFile := &MatroskaFile{File: file, FilePosition: 0, IsValid: false, Path: path}

	headerElement, headerErr := matroskaFile.readElement()
	if headerErr != nil {
		defer matroskaFile.Close()

		return nil, headerErr
	}

	if headerElement != nil && headerElement.Id == ElementEbml {
		newOffset, seekErr := matroskaFile.File.Seek(headerElement.DataSize, io.SeekCurrent)
		if seekErr != nil {
			defer matroskaFile.Close()

			return nil, fmt.Errorf("failed to read Matroska file %s: %w", path, seekErr)
		}

		matroskaFile.FilePosition = newOffset

		segmentElement, segmentErr := matroskaFile.readElement()
		if segmentErr != nil {
			defer matroskaFile.Close()

			return nil, segmentErr
		}

		if segmentElement != nil && segmentElement.Id == ElementSegment {
			stat, statErr := file.Stat()
			if statErr != nil {
				defer matroskaFile.Close()

				return nil, fmt.Errorf("failed to get Matroska file info: %w", statErr)
			}

			matroskaFile.FileSize = stat.Size()
			matroskaFile.IsValid = true
			matroskaFile.SegmentElement = segmentElement

			return matroskaFile, nil
		}
	}

	defer matroskaFile.Close()

	return nil, fmt.Errorf("failed to read header of Matroska file %s", path)
}

func (m *MatroskaFile) String() string {
	return fmt.Sprintf("Duration: %v , FrameRate: %v", m.Duration, m.FrameRate)
}

func (m *MatroskaFile) Subtitles(trackNumber uint64, progressCallback func(int64, int64)) ([]*MatroskaSubtitle, error) {
	m.MatroskaSubtitles = nil

	matroskaFileOptions := &MatroskaFileOptions{SubtitleTrack: trackNumber}

	readSegmentClusterErr := m.readSegmentCluster(matroskaFileOptions, progressCallback)
	if readSegmentClusterErr != nil {
		return nil, fmt.Errorf("failed to read subtitles: %w", readSegmentClusterErr)
	}

	return m.MatroskaSubtitles, nil
}
