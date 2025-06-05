package matroska

import (
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/cockroachdb/errors"
)

type MatroskaFile struct {
	Duration       float64
	FrameRate      float64
	IsValid        bool
	Path           string
	PixelHeight    int
	PixelWidth     int
	SegmentElement *Element
	TimeCodeScale  int64
	VideoCodecId   string

	file         *os.File
	filePosition int64
	fileSize     int64
	isOpen       bool
	subtitles    []MatroskaSubtitle
	tracks       []MatroskaTrackInfo
}

func (m *MatroskaFile) offsetFilePosition(offset int) {
	m.filePosition += int64(offset)
}

func (m *MatroskaFile) scaleTime32(time float32) float64 {
	return float64(time) * float64(m.TimeCodeScale) / 1000000.0
}

func (m *MatroskaFile) scaleTime64(time float64) float64 {
	return time * float64(m.TimeCodeScale) / 1000000.0
}

func (m *MatroskaFile) Close() error {
	if !m.isOpen {
		return nil
	}

	m.Duration = -1
	m.FrameRate = -1
	m.filePosition = -1
	m.fileSize = -1
	m.isOpen = false
	m.IsValid = false
	m.Path = ""
	m.PixelHeight = 0
	m.PixelWidth = 0
	m.SegmentElement = nil
	m.subtitles = nil
	m.TimeCodeScale = -1
	m.tracks = nil
	m.VideoCodecId = ""

	return m.file.Close()
}

func NewMatroskaFile(path string) (*MatroskaFile, error) {
	file, openErr := os.Open(path)
	if openErr != nil {
		return nil, errors.Wrapf(openErr, "failed to open Matroska file %s", path)
	}

	matroskaFile := &MatroskaFile{file: file, filePosition: 0, isOpen: true, IsValid: false, Path: path}

	headerElement, headerErr := matroskaFile.readElement()
	if headerErr != nil {
		defer matroskaFile.Close()

		return nil, headerErr
	}

	if headerElement != InvalidElement && headerElement.Id == ElementEbml {
		newOffset, seekErr := matroskaFile.file.Seek(headerElement.DataSize, io.SeekCurrent)
		if seekErr != nil {
			defer matroskaFile.Close()

			return nil, errors.Wrapf(seekErr, "failed to seek while opening Matroska file %s", path)
		}

		matroskaFile.filePosition = newOffset

		segmentElement, segmentErr := matroskaFile.readElement()
		if segmentErr != nil {
			defer matroskaFile.Close()

			return nil, errors.Wrapf(segmentErr, "failed to read segment element while opening Matroska file %s", path)
		}

		if segmentElement != InvalidElement && segmentElement.Id == ElementSegment {
			stat, statErr := file.Stat()
			if statErr != nil {
				defer matroskaFile.Close()

				return nil, errors.Wrapf(statErr, "failed to read information while opening Matroska file %s", path)
			}

			matroskaFile.fileSize = stat.Size()
			matroskaFile.IsValid = true
			matroskaFile.SegmentElement = &segmentElement

			return matroskaFile, nil
		}
	}

	defer matroskaFile.Close()

	return nil, errors.Newf("failed to read header of Matroska file %s", path)
}

func (m *MatroskaFile) String() string {
	return fmt.Sprintf("Duration: %v , FrameRate: %v", m.Duration, m.FrameRate)
}

func (m *MatroskaFile) Subtitle(trackNumber uint64, progressCallback func(int64, int64)) ([]MatroskaSubtitle, error) {
	m.subtitles = nil

	matroskaFileOptions := MatroskaFileOptions{SubtitleTrack: trackNumber}

	readSegmentClusterErr := m.readSegmentCluster(matroskaFileOptions, progressCallback)
	if readSegmentClusterErr != nil {
		return nil, errors.Wrap(readSegmentClusterErr, "failed to read subtitles")
	}

	return m.subtitles, nil
}

func (m *MatroskaFile) Tracks(subtitleOnly bool) ([]MatroskaTrackInfo, error) {
	segmentInfoAndTracksErr := m.readSegmentInfoAndTracks()
	if segmentInfoAndTracksErr != nil {
		return nil, errors.Wrap(segmentInfoAndTracksErr, "failed to read tracks")
	}

	if m.tracks == nil {
		return []MatroskaTrackInfo{}, nil
	}

	if subtitleOnly {
		return slices.Collect(func(yield func(MatroskaTrackInfo) bool) {
			for _, track := range m.tracks {
				if track.IsSubtitle {
					if !yield(track) {
						return
					}
				}
			}
		}), nil
	}

	return m.tracks, nil
}
