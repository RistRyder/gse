package matroska

import (
	"encoding/binary"
	"io"
	"math"

	"github.com/cockroachdb/errors"
	"golang.org/x/sys/cpu"
)

func (m *MatroskaFile) readBlockGroupElement(clusterElement *Element, clusterTimeCode int64, options *MatroskaFileOptions) error {
	var element *Element = &Element{}
	var elementErr error
	var subtitle *MatroskaSubtitle
	var subtitleErr error

	for m.filePosition < clusterElement.EndPosition() && element != nil {
		element, elementErr = m.readElement()
		if elementErr != nil {
			return errors.Wrap(elementErr, "failed to read cluster element")
		}

		if element == nil {
			return nil
		}

		switch element.Id {
		case ElementBlock:
			subtitle, subtitleErr = m.readSubtitleBlock(element, clusterTimeCode, options)
			if subtitleErr != nil {
				return errors.Wrap(subtitleErr, "failed to read subtitle block")
			}

			if subtitle != nil {
				m.subtitles = append(m.subtitles, subtitle)
			}
		case ElementBlockDuration:
			duration, durationErr := m.readUInt(int(element.DataSize))
			if durationErr != nil {
				return errors.Wrap(durationErr, "failed to read block duration element")
			}

			if subtitle != nil {
				subtitle.Duration = int64(math.Round(m.scaleTime64(float64(duration))))
			}
		default:
			newOffset, seekErr := m.file.Seek(element.DataSize, io.SeekCurrent)
			if seekErr != nil {
				return errors.Wrap(seekErr, "failed to seek while reading block groupd element")
			}

			m.filePosition = newOffset
		}
	}

	return nil
}

func (m *MatroskaFile) readCluster(clusterElement *Element, options *MatroskaFileOptions) error {
	clusterTimeCode := int64(0)
	var element *Element = &Element{}
	var elementErr error

	for m.filePosition < clusterElement.EndPosition() && element != nil {
		element, elementErr = m.readElement()
		if elementErr != nil {
			return errors.Wrap(elementErr, "failed to read cluster element")
		}

		if element == nil {
			return nil
		}

		switch element.Id {
		case ElementTimecode:
			ctc, clusterTimeCodeErr := m.readUInt(int(element.DataSize))
			if clusterTimeCodeErr != nil {
				return errors.Wrap(clusterTimeCodeErr, "failed to read cluster time code")
			}

			clusterTimeCode = int64(ctc)
		case ElementBlockGroup:
			blockGroupElementErr := m.readBlockGroupElement(element, clusterTimeCode, options)
			if blockGroupElementErr != nil {
				return errors.Wrap(blockGroupElementErr, "failed to read block group element")
			}
		case ElementSimpleBlock:
			subtitle, subtitleErr := m.readSubtitleBlock(element, clusterTimeCode, options)
			if subtitleErr != nil {
				return errors.Wrap(subtitleErr, "failed to read subtitle block")
			}

			if subtitle != nil {
				m.subtitles = append(m.subtitles, subtitle)
			}
		default:
			newOffset, seekErr := m.file.Seek(element.DataSize, io.SeekCurrent)
			if seekErr != nil {
				return errors.Wrap(seekErr, "failed to seek while reading cluster")
			}

			m.filePosition = newOffset
		}
	}

	return nil
}

func (m *MatroskaFile) readContentEncodingElement(contentEncodingElement *Element) (int, int, uint, error) {
	contentCompressionAlgorithm, contentEncodingType, contentEncodingScope := 0, 0, uint(0)
	var element *Element = &Element{}
	var elementErr error

	for m.filePosition < contentEncodingElement.EndPosition() && element != nil {
		element, elementErr = m.readElement()
		if elementErr != nil {
			return 0, 0, 0, errors.Wrap(elementErr, "failed to read content encoding element")
		}

		switch element.Id {
		case ElementContentEncodingOrder:
			_, contentEncodingOrderErr := m.readUInt(int(contentEncodingElement.DataSize))
			if contentEncodingOrderErr != nil {
				return 0, 0, 0, errors.Wrap(contentEncodingOrderErr, "failed to read content encoding order")
			}
		case ElementContentEncodingScope:
			ces, contentEncodingScopeErr := m.readUInt(int(contentEncodingElement.DataSize))
			if contentEncodingScopeErr != nil {
				return 0, 0, 0, errors.Wrap(contentEncodingScopeErr, "failed to read content encoding scope")
			}

			contentEncodingScope = uint(ces)
		case ElementContentEncodingType:
			cet, pixelHeightErr := m.readUInt(int(contentEncodingElement.DataSize))
			if pixelHeightErr != nil {
				return 0, 0, 0, errors.Wrap(pixelHeightErr, "failed to read content encoding type")
			}

			contentEncodingType = int(cet)
		case ElementContentCompression:
			var compressionElement *Element = &Element{}
			var compressionElementErr error

			for m.filePosition < element.EndPosition() && element != nil {
				compressionElement, compressionElementErr = m.readElement()
				if compressionElementErr != nil {
					return 0, 0, 0, errors.Wrap(elementErr, "failed to read content compression element")
				}

				switch compressionElement.Id {
				case ElementContentCompAlgo:
					cca, pixelHeightErr := m.readUInt(int(compressionElement.DataSize))
					if pixelHeightErr != nil {
						return 0, 0, 0, errors.Wrap(pixelHeightErr, "failed to read content compression algorithm")
					}

					contentCompressionAlgorithm = int(cca)
				case ElementContentCompSettings:
					_, contentCompSettingsErr := m.readUInt(int(compressionElement.DataSize))
					if contentCompSettingsErr != nil {
						return 0, 0, 0, errors.Wrap(contentCompSettingsErr, "failed to read content encoding order")
					}
				default:
					newOffset, seekErr := m.file.Seek(element.DataSize, io.SeekCurrent)
					if seekErr != nil {
						return 0, 0, 0, errors.Wrap(seekErr, "failed to seek while reading content compression element")
					}

					m.filePosition = newOffset
				}
			}
		default:
			newOffset, seekErr := m.file.Seek(element.DataSize, io.SeekCurrent)
			if seekErr != nil {
				return 0, 0, 0, errors.Wrap(seekErr, "failed to seek while reading content encoding element")
			}

			m.filePosition = newOffset
		}
	}

	return contentCompressionAlgorithm, contentEncodingType, contentEncodingScope, nil
}

func (m *MatroskaFile) readElement() (*Element, error) {
	idElement, idErr := m.readVariableLengthUInt(false)
	if idErr != nil {
		return nil, errors.Wrap(idErr, "failed to read Id element from Matroska file")
	}

	id := ElementId(idElement)
	if id == ElementNone {
		return nil, nil
	}

	sizeElement, sizeErr := m.readVariableLengthUIntDefault()
	if sizeErr != nil {
		return nil, errors.Wrap(sizeErr, "failed to read size element from Matroska file")
	}

	return NewElement(id, m.filePosition, int64(sizeElement)), nil
}

func (m *MatroskaFile) readFloat32() (float32, error) {
	data := make([]byte, 4)
	bytesRead, readErr := m.file.Read(data)
	if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
		return 0, errors.Wrap(readErr, "failed to read 32-bit float from Matroska file")
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
	bytesRead, readErr := m.file.Read(data)
	if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
		return 0, errors.Wrap(readErr, "failed to read 64-bit float from Matroska file")
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
	bytesRead, readErr := m.file.Read(data)
	if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
		return 0, errors.Wrap(readErr, "failed to read 16-bit integer from Matroska file")
	}

	m.offsetFilePosition(bytesRead)

	return int16(data[0]<<8 | data[1]), nil
}

func (m *MatroskaFile) readInfoElement(tracksElement *Element) error {
	var element *Element = &Element{}
	var elementErr error

	for m.filePosition < tracksElement.EndPosition() && element != nil {
		element, elementErr = m.readElement()
		if elementErr != nil {
			return errors.Wrap(elementErr, "failed to read tracks element")
		}

		switch element.Id {
		case ElementTimecodeScale:
			timecodeScale, timecodeScaleErr := m.readUInt(int(element.DataSize))
			if timecodeScaleErr != nil {
				return errors.Wrap(timecodeScaleErr, "failed to read timecode scale")
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
				return errors.Wrap(durationErr, "failed to read duration")
			}
		default:
			newOffset, seekErr := m.file.Seek(element.DataSize, io.SeekCurrent)
			if seekErr != nil {
				return errors.Wrap(seekErr, "failed to advance to next info element")
			}

			m.filePosition = newOffset
		}
	}

	return nil
}

func (m *MatroskaFile) readSegmentCluster(options *MatroskaFileOptions, progressCallback func(int64, int64)) error {
	//go to segment
	newOffset, seekErr := m.file.Seek(m.SegmentElement.DataPosition, io.SeekStart)
	if seekErr != nil {
		return errors.Wrap(seekErr, "failed to advance to segment cluster")
	}

	m.filePosition = newOffset

	for m.filePosition < m.SegmentElement.EndPosition() {
		beforeReadElementIdPosition := m.filePosition
		rawElementId, elementIdErr := m.readVariableLengthUInt(false)
		if elementIdErr != nil {
			return errors.Wrap(elementIdErr, "failed to read segment cluster element")
		}

		elementId := ElementId(rawElementId)
		if ElementId(elementId) == ElementNone && beforeReadElementIdPosition+1000 < m.fileSize {
			//Error mode: search for start of next cluster, will be very slow
			maxErrors := 5000000
			errorCount := 0
			max := m.fileSize

			for elementId != ElementCluster && beforeReadElementIdPosition+1000 < max {
				errorCount++
				if errorCount > maxErrors {
					//we give up
					return errors.New("maximum error count reached while searching for segment cluster")
				}

				beforeReadElementIdPosition++
				newOffset, seekErr = m.file.Seek(beforeReadElementIdPosition, io.SeekStart)
				if seekErr != nil {
					return errors.Wrap(seekErr, "failed to advance while searching for segment cluster")
				}

				m.filePosition = newOffset

				rawElementId, elementIdErr = m.readVariableLengthUInt(false)
				if elementIdErr != nil {
					return errors.Wrap(elementIdErr, "failed to read element while searching for segment cluster")
				}

				elementId = ElementId(rawElementId)
			}
		}

		size, sizeErr := m.readVariableLengthUIntDefault()
		if sizeErr != nil {
			return errors.Wrap(sizeErr, "failed to read size for segment cluster")
		}

		element := NewElement(elementId, m.filePosition, int64(size))
		if element.Id == ElementCluster {
			m.readCluster(element, options)
		} else {
			newOffset, seekErr = m.file.Seek(element.DataSize, io.SeekCurrent)
			if seekErr != nil {
				return errors.Wrap(seekErr, "failed to advance while reading segment cluster")
			}

			m.filePosition = newOffset
		}

		progressCallback(element.EndPosition(), m.fileSize)
	}

	return nil
}

func (m *MatroskaFile) readSegmentInfoAndTracks() error {
	//go to segment
	newOffset, seekErr := m.file.Seek(m.SegmentElement.DataPosition, io.SeekStart)
	if seekErr != nil {
		return errors.Wrap(seekErr, "failed to advance to segment element")
	}

	m.filePosition = newOffset

	var element *Element = &Element{}
	var elementErr error

	for m.filePosition < m.SegmentElement.EndPosition() && element != nil {
		element, elementErr = m.readElement()
		if elementErr != nil {
			return errors.Wrap(elementErr, "failed to read tracks element")
		}

		switch element.Id {
		case ElementInfo:
			infoError := m.readInfoElement(element)
			if infoError != nil {
				return errors.Wrap(infoError, "failed to read info element")
			}
		case ElementTracks:
			tracksError := m.readTracksElement(element)
			if tracksError != nil {
				return errors.Wrap(tracksError, "failed to read tracks element")
			}
		default:
			newOffset, seekErr := m.file.Seek(element.DataSize, io.SeekCurrent)
			if seekErr != nil {
				return errors.Wrap(seekErr, "failed to advance to next element")
			}

			m.filePosition = newOffset
		}
	}

	return nil
}

func (m *MatroskaFile) readString(length int) (string, error) {
	buffer := make([]byte, length)
	bytesRead, readErr := m.file.Read(buffer)
	if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
		return "", errors.Wrap(readErr, "failed to read string from Matroska file")
	}

	m.offsetFilePosition(bytesRead)

	return string(buffer), nil
}

func (m *MatroskaFile) readSubtitleBlock(blockElement *Element, clusterTimeCode int64, options *MatroskaFileOptions) (*MatroskaSubtitle, error) {
	trackNumber, trackNumberErr := m.readVariableLengthUIntDefault()
	if trackNumberErr != nil {
		return nil, errors.Wrap(trackNumberErr, "failed to read subtitle track number")
	}

	if options == nil || options.SubtitleTrack != trackNumber {
		newOffset, seekErr := m.file.Seek(blockElement.EndPosition(), io.SeekStart)
		if seekErr != nil {
			return nil, errors.Wrap(seekErr, "failed to advance to next element")
		}

		m.filePosition = newOffset

		return nil, nil
	}

	timeCode, timeCodeErr := m.readInt16()
	if timeCodeErr != nil {
		return nil, errors.Wrap(timeCodeErr, "failed to read subtitle time code")
	}

	//lacing
	buffer := make([]byte, 1)
	bytesRead, readErr := m.file.Read(buffer)
	if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
		return nil, errors.Wrap(readErr, "failed to read flags for subtitle block")
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
		bytesRead, readErr = m.file.Read(buffer)
		if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
			return nil, errors.Wrap(readErr, "failed to read frames for subtitle block")
		}

		m.offsetFilePosition(bytesRead)

		frames = int(buffer[0]) + 1
	//00000100 = Fixed-size lacing
	case 4:
		bytesRead, readErr = m.file.Read(buffer)
		if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
			return nil, errors.Wrap(readErr, "failed to read frames for subtitle block")
		}

		m.offsetFilePosition(bytesRead)

		frames = int(buffer[0]) + 1

		for i := 0; i < frames; i++ {
			//frames
			bytesRead, readErr = m.file.Read(buffer)
			if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
				return nil, errors.Wrap(readErr, "failed to read frames for subtitle block")
			}

			m.offsetFilePosition(bytesRead)
		}
	//00000110 = EMBL lacing
	case 6:
		bytesRead, readErr = m.file.Read(buffer)
		if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
			return nil, errors.Wrap(readErr, "failed to read frames for subtitle block")
		}

		m.offsetFilePosition(bytesRead)

		frames = int(buffer[0]) + 1
	}

	//save subtitle data
	dataLength := blockElement.EndPosition() - m.filePosition
	data := make([]byte, dataLength)
	bytesRead, readErr = m.file.Read(data)
	if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
		return nil, errors.Wrap(readErr, "failed to read data for subtitle")
	}

	m.offsetFilePosition(bytesRead)

	subtitleStart := int64(math.Round(m.scaleTime64(float64(clusterTimeCode + int64(timeCode)))))

	return NewMatroskaSubtitle(data, subtitleStart), nil
}

func (m *MatroskaFile) readTrackEntryElement(trackEntryElement *Element) (*MatroskaTrackInfo, error) {
	var element *Element = &Element{}
	var elementErr error
	track := &MatroskaTrackInfo{CodecId: "", IsDefault: true, Language: "eng", Name: ""}

	for m.filePosition < trackEntryElement.EndPosition() && element != nil {
		element, elementErr = m.readElement()
		if elementErr != nil {
			return nil, errors.Wrap(elementErr, "failed to read track entry element")
		}

		switch element.Id {
		case ElementDefaultDuration:
			defaultDuration, defaultDurationErr := m.readUInt(int(element.DataSize))
			if defaultDurationErr != nil {
				return nil, errors.Wrap(defaultDurationErr, "failed to read track default duration")
			}

			track.DefaultDuration = int(defaultDuration)
		case ElementVideo:
			videoErr := m.readVideoElement(element)
			if videoErr != nil {
				return nil, errors.Wrap(videoErr, "failed to read track video")
			}

			track.IsVideo = true
		case ElementAudio:
			track.IsAudio = true
		case ElementTrackNumber:
			trackNumber, trackNumberErr := m.readUInt(int(element.DataSize))
			if trackNumberErr != nil {
				return nil, errors.Wrap(trackNumberErr, "failed to read track number")
			}

			track.TrackNumber = int(trackNumber)
		case ElementName:
			name, nameErr := m.readString(int(element.DataSize))
			if nameErr != nil {
				return nil, errors.Wrap(nameErr, "failed to read track name")
			}

			track.Name = name
		case ElementLanguage:
			language, languageErr := m.readString(int(element.DataSize))
			if languageErr != nil {
				return nil, errors.Wrap(languageErr, "failed to read track language")
			}

			track.Language = language
		case ElementCodecId:
			codecId, codecIdErr := m.readString(int(element.DataSize))
			if codecIdErr != nil {
				return nil, errors.Wrap(codecIdErr, "failed to read track codec id")
			}

			track.CodecId = codecId
		case ElementTrackType:
			buffer := make([]byte, 1)
			bytesRead, readErr := m.file.Read(buffer)
			if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
				return nil, errors.Wrap(readErr, "failed to read track type")
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
			bytesRead, readErr := m.file.Read(codecPrivateRaw)
			if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
				return nil, errors.Wrap(readErr, "failed to read track private codec")
			}

			m.offsetFilePosition(bytesRead)
		case ElementContentEncodings:
			contentEncodingElement, contentEncodingElementErr := m.readElement()
			if contentEncodingElementErr != nil || contentEncodingElement == nil || contentEncodingElement.Id != ElementContentEncoding {
				return nil, errors.Wrap(contentEncodingElementErr, "failed to read track content encoding element")
			}

			contentCompressionAlgorithm, contentEncodingType, contentEncodingScope, contentEncodingErr := m.readContentEncodingElement(contentEncodingElement)
			if contentEncodingErr != nil {
				return nil, errors.Wrap(contentEncodingElementErr, "failed to read track content encoding")
			}

			track.ContentCompressionAlgorithm = contentCompressionAlgorithm
			track.ContentEncodingScope = contentEncodingScope
			track.ContentEncodingType = contentEncodingType
		case ElementFlagDefault:
			flagDefault, flagDefaultErr := m.readUInt(int(element.DataSize))
			if flagDefaultErr != nil {
				return nil, errors.Wrap(flagDefaultErr, "failed to read track 'default' flag")
			}

			track.IsDefault = flagDefault == 1
		case ElementFlagForced:
			flagForced, flagForcedErr := m.readUInt(int(element.DataSize))
			if flagForcedErr != nil {
				return nil, errors.Wrap(flagForcedErr, "failed to read track 'default' flag")
			}

			track.IsDefault = flagForced == 1
		}

		newOffset, seekErr := m.file.Seek(element.EndPosition(), io.SeekStart)
		if seekErr != nil {
			return nil, errors.Wrap(seekErr, "failed to advance to next track entry")
		}

		m.filePosition = newOffset
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

	for m.filePosition < tracksElement.EndPosition() && element != nil {
		element, elementErr = m.readElement()
		if elementErr != nil {
			return errors.Wrap(elementErr, "failed to read tracks element")
		}

		if element.Id == ElementTrackEntry {
			track, trackErr := m.readTrackEntryElement(element)
			if trackErr != nil {
				return errors.Wrap(trackErr, "failed to read tracks entry element")
			}

			m.tracks = append(m.tracks, track)
		} else {
			newOffset, seekErr := m.file.Seek(element.DataSize, io.SeekCurrent)
			if seekErr != nil {
				return errors.Wrap(seekErr, "failed to advance to next tracks entry")
			}

			m.filePosition = newOffset
		}
	}

	return nil
}

func (m *MatroskaFile) readUInt(length int) (uint64, error) {
	data := make([]byte, length)
	bytesRead, readErr := m.file.Read(data)
	if readErr != nil && readErr != io.EOF {
		return 0, errors.Wrap(readErr, "failed to read uint from Matroska file")
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

	bytesRead, readErr := m.file.Read(buffer)
	if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
		return 0, errors.Wrap(readErr, "failed to read byte from Matroska file")
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
		bytesRead, readErr := m.file.Read(buffer)
		if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
			return 0, errors.Wrap(readErr, "failed to read byte from Matroska file")
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

	for m.filePosition < videoElement.EndPosition() && element != nil {
		element, elementErr = m.readElement()
		if elementErr != nil {
			return errors.Wrap(elementErr, "failed to read video element")
		}

		switch element.Id {
		case ElementPixelWidth:
			pixelWidth, pixelWidthErr := m.readUInt(int(videoElement.DataSize))
			if pixelWidthErr != nil {
				return errors.Wrap(pixelWidthErr, "failed to read pixel width")
			}

			m.PixelWidth = int(pixelWidth)
		case ElementPixelHeight:
			pixelHeight, pixelHeightErr := m.readUInt(int(videoElement.DataSize))
			if pixelHeightErr != nil {
				return errors.Wrap(pixelHeightErr, "failed to read pixel height")
			}

			m.PixelHeight = int(pixelHeight)
		default:
			newOffset, seekErr := m.file.Seek(videoElement.DataSize, io.SeekCurrent)
			if seekErr != nil {
				return errors.Wrap(seekErr, "failed to seek while reading video element")
			}

			m.filePosition = newOffset
		}
	}

	return nil
}
