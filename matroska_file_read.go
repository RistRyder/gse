package gse

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	"golang.org/x/sys/cpu"
)

func (m *MatroskaFile) readBlockGroupElement(clusterElement *Element, clusterTimeCode int64, options *MatroskaFileOptions) error {
	var element *Element = &Element{}
	var elementErr error
	var subtitle *MatroskaSubtitle
	var subtitleErr error

	for m.FilePosition < clusterElement.EndPosition() && element != nil {
		element, elementErr = m.readElement()
		if elementErr != nil {
			return fmt.Errorf("failed to read cluster element: %w", elementErr)
		}

		if element == nil {
			return nil
		}

		switch element.Id {
		case ElementBlock:
			subtitle, subtitleErr = m.readSubtitleBlock(element, clusterTimeCode, options)
			if subtitleErr != nil {
				return fmt.Errorf("failed to read subtitle block: %w", subtitleErr)
			}

			if subtitle != nil {
				m.subtitles = append(m.subtitles, subtitle)
			}
		case ElementBlockDuration:
			duration, durationErr := m.readUInt(int(element.DataSize))
			if durationErr != nil {
				return fmt.Errorf("failed to read block duration element: %w", durationErr)
			}

			if subtitle != nil {
				subtitle.Duration = int64(math.Round(m.scaleTime64(float64(duration))))
			}
		default:
			newOffset, seekErr := m.File.Seek(element.DataSize, io.SeekCurrent)
			if seekErr != nil {
				return fmt.Errorf("failed to seek while reading block group element: %w", seekErr)
			}

			m.FilePosition = newOffset
		}
	}

	return nil
}

func (m *MatroskaFile) readCluster(clusterElement *Element, options *MatroskaFileOptions) error {
	clusterTimeCode := int64(0)
	var element *Element = &Element{}
	var elementErr error

	for m.FilePosition < clusterElement.EndPosition() && element != nil {
		element, elementErr = m.readElement()
		if elementErr != nil {
			return fmt.Errorf("failed to read cluster element: %w", elementErr)
		}

		if element == nil {
			return nil
		}

		switch element.Id {
		case ElementTimecode:
			ctc, clusterTimeCodeErr := m.readUInt(int(element.DataSize))
			if clusterTimeCodeErr != nil {
				return fmt.Errorf("failed to read cluster time code: %w", clusterTimeCodeErr)
			}

			clusterTimeCode = int64(ctc)
		case ElementBlockGroup:
			blockGroupElementErr := m.readBlockGroupElement(element, clusterTimeCode, options)
			if blockGroupElementErr != nil {
				return fmt.Errorf("failed to read block group element: %w", blockGroupElementErr)
			}
		case ElementSimpleBlock:
			subtitle, subtitleErr := m.readSubtitleBlock(element, clusterTimeCode, options)
			if subtitleErr != nil {
				return fmt.Errorf("failed to read subtitle block: %w", subtitleErr)
			}

			if subtitle != nil {
				m.subtitles = append(m.subtitles, subtitle)
			}
		default:
			newOffset, seekErr := m.File.Seek(element.DataSize, io.SeekCurrent)
			if seekErr != nil {
				return fmt.Errorf("failed to seek while reading cluster: %w", seekErr)
			}

			m.FilePosition = newOffset
		}
	}

	return nil
}

func (m *MatroskaFile) readContentEncodingElement(contentEncodingElement *Element) (int, int, uint, error) {
	contentCompressionAlgorithm, contentEncodingType, contentEncodingScope := 0, 0, uint(0)
	var element *Element = &Element{}
	var elementErr error

	for m.FilePosition < contentEncodingElement.EndPosition() && element != nil {
		element, elementErr = m.readElement()
		if elementErr != nil {
			return 0, 0, 0, fmt.Errorf("failed to read content encoding element: %w", elementErr)
		}

		switch element.Id {
		case ElementContentEncodingOrder:
			_, contentEncodingOrderErr := m.readUInt(int(contentEncodingElement.DataSize))
			if contentEncodingOrderErr != nil {
				return 0, 0, 0, fmt.Errorf("failed to read content encoding order: %w", contentEncodingOrderErr)
			}
		case ElementContentEncodingScope:
			ces, contentEncodingScopeErr := m.readUInt(int(contentEncodingElement.DataSize))
			if contentEncodingScopeErr != nil {
				return 0, 0, 0, fmt.Errorf("failed to read content encoding scope: %w", contentEncodingScopeErr)
			}

			contentEncodingScope = uint(ces)
		case ElementContentEncodingType:
			cet, pixelHeightErr := m.readUInt(int(contentEncodingElement.DataSize))
			if pixelHeightErr != nil {
				return 0, 0, 0, fmt.Errorf("failed to read content encoding type: %w", pixelHeightErr)
			}

			contentEncodingType = int(cet)
		case ElementContentCompression:
			var compressionElement *Element = &Element{}
			var compressionElementErr error

			for m.FilePosition < element.EndPosition() && element != nil {
				compressionElement, compressionElementErr = m.readElement()
				if compressionElementErr != nil {
					return 0, 0, 0, fmt.Errorf("failed to read content compression element: %w", elementErr)
				}

				switch compressionElement.Id {
				case ElementContentCompAlgo:
					cca, pixelHeightErr := m.readUInt(int(compressionElement.DataSize))
					if pixelHeightErr != nil {
						return 0, 0, 0, fmt.Errorf("failed to read content compression algorithm: %w", pixelHeightErr)
					}

					contentCompressionAlgorithm = int(cca)
				case ElementContentCompSettings:
					_, contentCompSettingsErr := m.readUInt(int(compressionElement.DataSize))
					if contentCompSettingsErr != nil {
						return 0, 0, 0, fmt.Errorf("failed to read content encoding order: %w", contentCompSettingsErr)
					}
				default:
					newOffset, seekErr := m.File.Seek(element.DataSize, io.SeekCurrent)
					if seekErr != nil {
						return 0, 0, 0, fmt.Errorf("failed to seek while reading content compression element: %w", seekErr)
					}

					m.FilePosition = newOffset
				}
			}
		default:
			newOffset, seekErr := m.File.Seek(element.DataSize, io.SeekCurrent)
			if seekErr != nil {
				return 0, 0, 0, fmt.Errorf("failed to seek while reading content encoding element: %w", seekErr)
			}

			m.FilePosition = newOffset
		}
	}

	return contentCompressionAlgorithm, contentEncodingType, contentEncodingScope, nil
}

func (m *MatroskaFile) readElement() (*Element, error) {
	idElement, idErr := m.readVariableLengthUInt(false)
	if idErr != nil {
		return nil, fmt.Errorf("failed to read Id element from Matroska file: %w", idErr)
	}

	id := ElementId(idElement)
	if id == ElementNone {
		return nil, nil
	}

	sizeElement, sizeErr := m.readVariableLengthUIntDefault()
	if sizeErr != nil {
		return nil, fmt.Errorf("failed to read size element from Matroska file: %w", sizeErr)
	}

	return NewElement(id, m.FilePosition, int64(sizeElement)), nil
}

func (m *MatroskaFile) readFloat32() (float32, error) {
	data := make([]byte, 4)
	bytesRead, readErr := m.File.Read(data)
	if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
		return 0, fmt.Errorf("failed to read 32-bit float from Matroska file: %w", readErr)
	}

	m.offsetFilePosition(bytesRead)

	var bits uint32

	if cpu.IsBigEndian {
		bits = binary.BigEndian.Uint32(data)
	} else {
		bits = binary.LittleEndian.Uint32(data)
	}

	return math.Float32frombits(bits), nil
}

func (m *MatroskaFile) readFloat64() (float64, error) {
	data := make([]byte, 8)
	bytesRead, readErr := m.File.Read(data)
	if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
		return 0, fmt.Errorf("failed to read 64-bit float from Matroska file: %w", readErr)
	}

	m.offsetFilePosition(bytesRead)

	var bits uint64

	if cpu.IsBigEndian {
		bits = binary.BigEndian.Uint64(data)
	} else {
		bits = binary.LittleEndian.Uint64(data)
	}

	return math.Float64frombits(bits), nil
}

func (m *MatroskaFile) readInt16() (int16, error) {
	data := make([]byte, 2)
	bytesRead, readErr := m.File.Read(data)
	if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
		return 0, fmt.Errorf("failed to read 16-bit integer from Matroska file: %w", readErr)
	}

	m.offsetFilePosition(bytesRead)

	return int16(data[0]<<8 | data[1]), nil
}

func (m *MatroskaFile) readInfoElement(tracksElement *Element) error {
	var element *Element = &Element{}
	var elementErr error

	for m.FilePosition < tracksElement.EndPosition() && element != nil {
		element, elementErr = m.readElement()
		if elementErr != nil {
			return fmt.Errorf("failed to read tracks element: %w", elementErr)
		}

		switch element.Id {
		case ElementTimecodeScale:
			timecodeScale, timecodeScaleErr := m.readUInt(int(element.DataSize))
			if timecodeScaleErr != nil {
				return fmt.Errorf("failed to read timecode scale: %w", timecodeScaleErr)
			}

			m.TimeCodeScale = int64(timecodeScale)
		case ElementDuration:
			var duration32 float32
			var duration64 float64
			var durationErr error

			if element.DataSize == 4 {
				duration32, durationErr = m.readFloat32()

				m.Duration = m.scaleTime32(duration32)
			} else {
				duration64, durationErr = m.readFloat64()

				m.Duration = m.scaleTime64(duration64)
			}

			if durationErr != nil {
				return fmt.Errorf("failed to read duration: %w", durationErr)
			}
		default:
			newOffset, seekErr := m.File.Seek(element.DataSize, io.SeekCurrent)
			if seekErr != nil {
				return fmt.Errorf("failed to advance to next info element: %w", seekErr)
			}

			m.FilePosition = newOffset
		}
	}

	return nil
}

func (m *MatroskaFile) readSegmentCluster(options *MatroskaFileOptions, progressCallback func(int64, int64)) error {
	//go to segment
	newOffset, seekErr := m.File.Seek(m.SegmentElement.DataPosition, io.SeekStart)
	if seekErr != nil {
		return fmt.Errorf("failed to advance to segment cluster: %w", seekErr)
	}

	m.FilePosition = newOffset

	for m.FilePosition < m.SegmentElement.EndPosition() {
		beforeReadElementIdPosition := m.FilePosition
		rawElementId, elementIdErr := m.readVariableLengthUInt(false)
		if elementIdErr != nil {
			return fmt.Errorf("failed to read segment cluster element: %w", elementIdErr)
		}

		elementId := ElementId(rawElementId)
		if ElementId(elementId) == ElementNone && beforeReadElementIdPosition+1000 < m.FileSize {
			//Error mode: search for start of next cluster, will be very slow
			maxErrors := 5000000
			errorCount := 0
			max := m.FileSize

			for elementId != ElementCluster && beforeReadElementIdPosition+1000 < max {
				errorCount++
				if errorCount > maxErrors {
					//we give up
					return errors.New("maximum error count reached while searching for segment cluster")
				}

				beforeReadElementIdPosition++
				newOffset, seekErr = m.File.Seek(beforeReadElementIdPosition, io.SeekStart)
				if seekErr != nil {
					return fmt.Errorf("failed to advance while searching for segment cluster: %w", seekErr)
				}

				m.FilePosition = newOffset

				rawElementId, elementIdErr = m.readVariableLengthUInt(false)
				if elementIdErr != nil {
					return fmt.Errorf("failed to read element while searching for segment cluster: %w", elementIdErr)
				}

				elementId = ElementId(rawElementId)
			}
		}

		size, sizeErr := m.readVariableLengthUIntDefault()
		if sizeErr != nil {
			return fmt.Errorf("failed to read size for segment cluster: %w", sizeErr)
		}

		element := NewElement(elementId, m.FilePosition, int64(size))
		if element.Id == ElementCluster {
			m.readCluster(element, options)
		} else {
			newOffset, seekErr = m.File.Seek(element.DataSize, io.SeekCurrent)
			if seekErr != nil {
				return fmt.Errorf("failed to advance while reading segment cluster: %w", seekErr)
			}

			m.FilePosition = newOffset
		}

		progressCallback(element.EndPosition(), m.FileSize)
	}

	return nil
}

func (m *MatroskaFile) readSegmentInfoAndTracks() error {
	//go to segment
	newOffset, seekErr := m.File.Seek(m.SegmentElement.DataPosition, io.SeekStart)
	if seekErr != nil {
		return fmt.Errorf("failed to advance to segment element: %w", seekErr)
	}

	m.FilePosition = newOffset

	var element *Element = &Element{}
	var elementErr error

	for m.FilePosition < m.SegmentElement.EndPosition() && element != nil {
		element, elementErr = m.readElement()
		if elementErr != nil {
			return fmt.Errorf("failed to read tracks element: %w", elementErr)
		}

		switch element.Id {
		case ElementInfo:
			infoError := m.readInfoElement(element)
			if infoError != nil {
				return fmt.Errorf("failed to read info element: %w", infoError)
			}
		case ElementTracks:
			tracksError := m.readTracksElement(element)
			if tracksError != nil {
				return fmt.Errorf("failed to read tracks element: %w", tracksError)
			}
		default:
			newOffset, seekErr := m.File.Seek(element.DataSize, io.SeekCurrent)
			if seekErr != nil {
				return fmt.Errorf("failed to advance to next element: %w", seekErr)
			}

			m.FilePosition = newOffset
		}
	}

	return nil
}

func (m *MatroskaFile) readString(length int) (string, error) {
	buffer := make([]byte, length)
	bytesRead, readErr := m.File.Read(buffer)
	if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
		return "", fmt.Errorf("failed to read string from Matroska file: %w", readErr)
	}

	m.offsetFilePosition(bytesRead)

	return string(buffer), nil
}

func (m *MatroskaFile) readSubtitleBlock(blockElement *Element, clusterTimeCode int64, options *MatroskaFileOptions) (*MatroskaSubtitle, error) {
	trackNumber, trackNumberErr := m.readVariableLengthUIntDefault()
	if trackNumberErr != nil {
		return nil, fmt.Errorf("failed to read subtitle track number: %w", trackNumberErr)
	}

	if options == nil || options.SubtitleTrack != trackNumber {
		newOffset, seekErr := m.File.Seek(blockElement.EndPosition(), io.SeekStart)
		if seekErr != nil {
			return nil, fmt.Errorf("failed to advance to next element: %w", seekErr)
		}

		m.FilePosition = newOffset

		return nil, nil
	}

	timeCode, timeCodeErr := m.readInt16()
	if timeCodeErr != nil {
		return nil, fmt.Errorf("failed to read subtitle time code: %w", timeCodeErr)
	}

	//lacing
	buffer := make([]byte, 1)
	bytesRead, readErr := m.File.Read(buffer)
	if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
		return nil, fmt.Errorf("failed to read flags for subtitle block: %w", readErr)
	}

	m.offsetFilePosition(bytesRead)

	flags := buffer[0]

	var frames int
	switch flags & 6 {
	//00000000 = No lacing
	//case 0:
	//fmt.Println("No lacing")
	//00000010 = Xiph lacing
	case 2:
		bytesRead, readErr = m.File.Read(buffer)
		if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
			return nil, fmt.Errorf("failed to read frames for subtitle block: %w", readErr)
		}

		m.offsetFilePosition(bytesRead)

		frames = int(buffer[0]) + 1
	//00000100 = Fixed-size lacing
	case 4:
		bytesRead, readErr = m.File.Read(buffer)
		if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
			return nil, fmt.Errorf("failed to read frames for subtitle block: %w", readErr)
		}

		m.offsetFilePosition(bytesRead)

		frames = int(buffer[0]) + 1

		for i := 0; i < frames; i++ {
			//frames
			bytesRead, readErr = m.File.Read(buffer)
			if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
				return nil, fmt.Errorf("failed to read frames for subtitle block: %w", readErr)
			}

			m.offsetFilePosition(bytesRead)
		}
	//00000110 = EMBL lacing
	case 6:
		bytesRead, readErr = m.File.Read(buffer)
		if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
			return nil, fmt.Errorf("failed to read frames for subtitle block: %w", readErr)
		}

		m.offsetFilePosition(bytesRead)

		frames = int(buffer[0]) + 1
	}

	//save subtitle data
	dataLength := blockElement.EndPosition() - m.FilePosition
	data := make([]byte, dataLength)
	bytesRead, readErr = m.File.Read(data)
	if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
		return nil, fmt.Errorf("failed to read data for subtitle: %w", readErr)
	}

	m.offsetFilePosition(bytesRead)

	subtitleStart := int64(math.Round(m.scaleTime64(float64(clusterTimeCode + int64(timeCode)))))

	return NewMatroskaSubtitle(data, subtitleStart), nil
}

func (m *MatroskaFile) readTrackEntryElement(trackEntryElement *Element) (*MatroskaTrackInfo, error) {
	var element *Element = &Element{}
	var elementErr error
	track := &MatroskaTrackInfo{CodecId: "", IsDefault: true, Language: "eng", Name: ""}

	for m.FilePosition < trackEntryElement.EndPosition() && element != nil {
		element, elementErr = m.readElement()
		if elementErr != nil {
			return nil, fmt.Errorf("failed to read track entry element: %w", elementErr)
		}

		switch element.Id {
		case ElementDefaultDuration:
			defaultDuration, defaultDurationErr := m.readUInt(int(element.DataSize))
			if defaultDurationErr != nil {
				return nil, fmt.Errorf("failed to read track default duration: %w", defaultDurationErr)
			}

			track.DefaultDuration = int(defaultDuration)
		case ElementVideo:
			videoErr := m.readVideoElement(element)
			if videoErr != nil {
				return nil, fmt.Errorf("failed to read track video: %w", videoErr)
			}

			track.IsVideo = true
		case ElementAudio:
			track.IsAudio = true
		case ElementTrackNumber:
			trackNumber, trackNumberErr := m.readUInt(int(element.DataSize))
			if trackNumberErr != nil {
				return nil, fmt.Errorf("failed to read track number: %w", trackNumberErr)
			}

			track.TrackNumber = int(trackNumber)
		case ElementName:
			name, nameErr := m.readString(int(element.DataSize))
			if nameErr != nil {
				return nil, fmt.Errorf("failed to read track name: %w", nameErr)
			}

			track.Name = name
		case ElementLanguage:
			language, languageErr := m.readString(int(element.DataSize))
			if languageErr != nil {
				return nil, fmt.Errorf("failed to read track language: %w", languageErr)
			}

			track.Language = language
		case ElementCodecId:
			codecId, codecIdErr := m.readString(int(element.DataSize))
			if codecIdErr != nil {
				return nil, fmt.Errorf("failed to read track codec id: %w", codecIdErr)
			}

			track.CodecId = codecId
		case ElementTrackType:
			buffer := make([]byte, 1)
			bytesRead, readErr := m.File.Read(buffer)
			if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
				return nil, fmt.Errorf("failed to read track type: %w", readErr)
			}

			m.offsetFilePosition(bytesRead)

			switch buffer[0] {
			case 1:
				track.IsVideo = true
			case 2:
				track.IsAudio = true
			case 17:
				track.IsSubtitle = true
			}
		case ElementCodecPrivate:
			codecPrivateRaw := make([]byte, element.DataSize)
			bytesRead, readErr := m.File.Read(codecPrivateRaw)
			if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
				return nil, fmt.Errorf("failed to read track private codec: %w", readErr)
			}

			m.offsetFilePosition(bytesRead)
		case ElementContentEncodings:
			contentEncodingElement, contentEncodingElementErr := m.readElement()
			if contentEncodingElementErr != nil || contentEncodingElement == nil || contentEncodingElement.Id != ElementContentEncoding {
				return nil, fmt.Errorf("failed to read track content encoding element: %w", contentEncodingElementErr)
			}

			contentCompressionAlgorithm, contentEncodingType, contentEncodingScope, contentEncodingErr := m.readContentEncodingElement(contentEncodingElement)
			if contentEncodingErr != nil {
				return nil, fmt.Errorf("failed to read track content encoding: %w", contentEncodingElementErr)
			}

			track.ContentCompressionAlgorithm = contentCompressionAlgorithm
			track.ContentEncodingScope = contentEncodingScope
			track.ContentEncodingType = contentEncodingType
		case ElementFlagDefault:
			flagDefault, flagDefaultErr := m.readUInt(int(element.DataSize))
			if flagDefaultErr != nil {
				return nil, fmt.Errorf("failed to read track 'default' flag: %w", flagDefaultErr)
			}

			track.IsDefault = flagDefault == 1
		case ElementFlagForced:
			flagForced, flagForcedErr := m.readUInt(int(element.DataSize))
			if flagForcedErr != nil {
				return nil, fmt.Errorf("failed to read track 'default' flag: %w", flagForcedErr)
			}

			track.IsDefault = flagForced == 1
		}

		newOffset, seekErr := m.File.Seek(element.EndPosition(), io.SeekStart)
		if seekErr != nil {
			return nil, fmt.Errorf("failed to advance to next track entry: %w", seekErr)
		}

		m.FilePosition = newOffset
	}

	if track.IsVideo {
		if track.DefaultDuration > 0 {
			m.FrameRate = 1.0 / (float64(track.DefaultDuration) / 1000000000.0)
		}

		m.VideoCodecId = track.CodecId
	}

	return track, nil
}

func (m *MatroskaFile) readTracksElement(tracksElement *Element) error {
	m.tracks = []*MatroskaTrackInfo{}

	var element *Element = &Element{}
	var elementErr error

	for m.FilePosition < tracksElement.EndPosition() && element != nil {
		element, elementErr = m.readElement()
		if elementErr != nil {
			return fmt.Errorf("failed to read tracks element: %w", elementErr)
		}

		if element.Id == ElementTrackEntry {
			track, trackErr := m.readTrackEntryElement(element)
			if trackErr != nil {
				return fmt.Errorf("failed to read tracks entry element: %w", trackErr)
			}

			m.tracks = append(m.tracks, track)
		} else {
			newOffset, seekErr := m.File.Seek(element.DataSize, io.SeekCurrent)
			if seekErr != nil {
				return fmt.Errorf("failed to advance to next tracks entry: %w", seekErr)
			}

			m.FilePosition = newOffset
		}
	}

	return nil
}

func (m *MatroskaFile) readUInt(length int) (uint64, error) {
	data := make([]byte, length)
	bytesRead, readErr := m.File.Read(data)
	if readErr != nil && readErr != io.EOF {
		return 0, fmt.Errorf("failed to read uint from Matroska file: %w", readErr)
	}

	m.offsetFilePosition(bytesRead)

	//Convert the big endian byte array to a 64-bit unsigned integer.
	result := uint64(0)
	shift := uint64(0)
	for i := length - 1; i >= 0; i-- {
		result |= uint64(data[i]) << shift
		shift += 8
	}

	return result, nil
}

func (m *MatroskaFile) readVariableLengthUInt(unsetFirstBit bool) (uint64, error) {
	//Begin loop with byte set to newly read byte
	buffer := make([]byte, 1)
	length := 0

	bytesRead, readErr := m.File.Read(buffer)
	if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
		return 0, fmt.Errorf("failed to read byte from Matroska file: %w", readErr)
	}

	m.offsetFilePosition(bytesRead)

	//Begin by counting the bits unset before the highest set bit
	mask := byte(0x80)
	for i := 0; i < 8; i++ {
		//Start at left, shift to right
		if (buffer[0] & mask) == mask {
			length = i + 1
			break
		}
		mask >>= 1
	}

	if length == 0 {
		return 0, nil
	}

	//Read remaining big endian bytes and convert to 64-bit unsigned integer.
	var result uint64
	if unsetFirstBit {
		result = uint64(buffer[0] & (0xFF >> length))
	} else {
		result = uint64(buffer[0])
	}
	length -= 1
	result <<= uint64(length * 8)

	for i := 1; i <= length; i++ {
		bytesRead, readErr := m.File.Read(buffer)
		if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
			return 0, fmt.Errorf("failed to read byte from Matroska file: %w", readErr)
		}

		m.offsetFilePosition(bytesRead)

		result |= uint64(buffer[0]) << ((length - i) * 8)
	}

	return result, nil
}

func (m *MatroskaFile) readVariableLengthUIntDefault() (uint64, error) {
	return m.readVariableLengthUInt(true)
}

func (m *MatroskaFile) readVideoElement(videoElement *Element) error {
	var element *Element = &Element{}
	var elementErr error

	for m.FilePosition < videoElement.EndPosition() && element != nil {
		element, elementErr = m.readElement()
		if elementErr != nil {
			return fmt.Errorf("failed to read video element: %w", elementErr)
		}

		switch element.Id {
		case ElementPixelWidth:
			pixelWidth, pixelWidthErr := m.readUInt(int(videoElement.DataSize))
			if pixelWidthErr != nil {
				return fmt.Errorf("failed to read pixel width: %w", pixelWidthErr)
			}

			m.PixelWidth = int(pixelWidth)
		case ElementPixelHeight:
			pixelHeight, pixelHeightErr := m.readUInt(int(videoElement.DataSize))
			if pixelHeightErr != nil {
				return fmt.Errorf("failed to read pixel height: %w", pixelHeightErr)
			}

			m.PixelHeight = int(pixelHeight)
		default:
			newOffset, seekErr := m.File.Seek(videoElement.DataSize, io.SeekCurrent)
			if seekErr != nil {
				return fmt.Errorf("failed to seek while reading video element: %w", seekErr)
			}

			m.FilePosition = newOffset
		}
	}

	return nil
}
