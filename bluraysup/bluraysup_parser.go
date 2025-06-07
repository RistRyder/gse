/*
 * Copyright 2009 Volker Oth (0xdeadbeef)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * NOTE: Converted to C# and modified by Nikse.dk@gmail.com
 * NOTE: For more info see http://blog.thescorpius.com/index.php/2017/07/15/presentation-graphic-stream-sup-files-bluray-subtitle-format/
 * NOTE: Converted from C# to Go by github.com/RistRyder
 */

package bluraysup

import (
	"fmt"
	"image"
	"slices"

	"github.com/RistRyder/gse/common"
	"github.com/RistRyder/gse/containers/matroska"
	"github.com/cockroachdb/errors"
)

const headerSize = 13

type supSegment struct {
	PtsTimestamp int64
	Size         int
	Type         int
}

func bigEndianInt32(buffer []byte, index int) uint32 {
	if len(buffer) < 4 {
		return 0
	}

	return uint32(buffer[index+3]) + (uint32(buffer[index+2]) << 8) + (uint32(buffer[index+1]) << 0x10) + (uint32(buffer[index]) << 0x18)
}

func completePcs(pcs *PcsData, bitmapObjects map[int][]OdsData, palettes map[int][]PaletteInfo) bool {
	if pcs == nil || pcs.PcsObjects == nil || palettes == nil {
		return false
	}
	if len(pcs.PcsObjects) == 0 {
		return true
	}
	infos, exists := palettes[pcs.PaletteId]
	if !exists {
		return false
	}

	pcs.BitmapObjects = [][]OdsData{}
	pcs.PaletteInfos = infos
	found := false
	for index := 0; index < len(pcs.PcsObjects); index++ {
		if bitmapObjs, exists := bitmapObjects[pcs.PcsObjects[index].ObjectId]; exists {
			pcs.BitmapObjects = append(pcs.BitmapObjects, bitmapObjs)
			found = true
		}
	}

	return found
}

func containsBluRayStartSegment(buffer []byte) bool {
	const epochStart = 0x80
	position := 0
	for position+3 <= len(buffer) {
		segmentType := buffer[position]
		if segmentType == epochStart {
			return true
		}

		length := BigEndianInt16(buffer, position+1) + 3
		position += int(length)
	}

	return false
}

func getCompositionState(stateType byte) CompositionState {
	switch stateType {
	case 0x00:
		return CompositionStateNormal
	case 0x40:
		return CompositionStateAcquPoint
	case 0x80:
		return CompositionStateEpochStart
	case 0xC0:
		return CompositionStateEpochContinue
	default:
		return CompositionStateInvalid
	}
}

func parseOds(buffer []byte, segment supSegment, forceFirst bool) OdsData {
	objId := int(BigEndianInt16(buffer, 0)) //16bit object_id
	objVer := int(buffer[2])                //16bit object_id nikse - index 2 or 1???
	objSeq := int(buffer[3])                //8bit first_in_sequence (0x80), last_in_sequence (0x40), 6bits reserved

	first := (objSeq&0x80) == 0x80 || forceFirst
	last := (objSeq & 0x40) == 0x40

	info := &ImageObjectFragment{}

	if first {
		width := BigEndianInt16(buffer, 7)  //object_width
		height := BigEndianInt16(buffer, 9) //object_height

		info.ImagePacketSize = segment.Size - 11 //Image packet size (image bytes)
		info.ImageBuffer = make([]byte, info.ImagePacketSize)
		_ = copy(info.ImageBuffer, buffer[11:info.ImagePacketSize])

		messageSeq1 := ""
		if last {
			messageSeq1 = "/"
		}
		messageSeq2 := ""
		if last {
			messageSeq2 = "last"
		}

		return OdsData{Fragment: info, IsFirst: true, Message: fmt.Sprintf("ObjId: %v, ver: %v, seq: first%v%v, width: %v, height: %v", objId, objVer, messageSeq1, messageSeq2, width, height), ObjectId: objId, ObjectVersion: objVer, Size: common.Size{Height: int(height), Width: int(width)}}
	}

	info.ImagePacketSize = segment.Size - 4
	info.ImageBuffer = make([]byte, info.ImagePacketSize)
	_ = copy(info.ImageBuffer, buffer[4:info.ImagePacketSize])

	messageSeq1 := ""
	if last {
		messageSeq1 = "last"
	}

	return OdsData{Fragment: info, IsFirst: false, Message: fmt.Sprintf("Continued ObjId: %v, ver: %v, seq: %v", objId, objVer, messageSeq1), ObjectId: objId, ObjectVersion: objVer}
}

func parsePcs(buffer []byte, offset int) PcsObject {
	pcs := PcsObject{ObjectId: int(BigEndianInt16(buffer, 11+offset)), WindowId: int(buffer[13+offset])}
	//composition_object:
	//16bit object_id_ref
	//skipped:  8bit  window_id_ref
	//object_cropped_flag: 0x80, forced_on_flag = 0x040, 6bit reserved
	forcedCropped := buffer[14+offset]
	pcs.IsForced = (forcedCropped & 0x40) == 0x40
	pcs.Origin = image.Point{X: int(BigEndianInt16(buffer, 15+offset)), Y: int(BigEndianInt16(buffer, 17+offset))}

	return pcs
}

func parsePds(buffer []byte, segment supSegment) PdsData {
	paletteInfo := PaletteInfo{Size: (segment.Size - 2) / 5}

	if paletteInfo.Size <= 0 {
		return PdsData{Message: "Empty palette"}
	}

	paletteInfo.Buffer = make([]byte, paletteInfo.Size*5)
	_ = copy(paletteInfo.Buffer, buffer[2:paletteInfo.Size*5])

	paletteId := int(buffer[0])     //8bit palette ID (0..7)
	paletteUpdate := int(buffer[1]) //8bit palette version number (incremented for each palette change)

	return PdsData{Id: paletteId, Message: fmt.Sprintf("PalId: %v, update: %v, %v entries", paletteId, paletteUpdate, paletteInfo.Size), PaletteInfo: &paletteInfo, Version: paletteUpdate}
}

func parsePicture(buffer []byte, segment supSegment) PcsData {
	if len(buffer) < 11 {
		return PcsData{CompositionState: CompositionStateInvalid}
	}

	pcs := PcsData{
		CompNum:             int(BigEndianInt16(buffer, 5)),
		CompositionState:    getCompositionState(buffer[7]),
		FramesPerSecondType: int(buffer[4]),
		PaletteId:           int(buffer[9]),
		PaletteUpdate:       buffer[8] == 0x80,
		Size:                common.Size{Height: int(BigEndianInt16(buffer, 2)), Width: int(BigEndianInt16(buffer, 0))},
		StartTime:           segment.PtsTimestamp,
	}
	//hi nibble: frame_rate, lo nibble: reserved
	//8bit  palette_update_flag (0x80), 7bit reserved
	//8bit  palette_id_ref
	compositionObjectCount := buffer[10] //8bit  number_of_composition_objects (0..2)

	if pcs.CompositionState == CompositionStateInvalid {
		//Illegal composition state Invalid
	} else {
		offset := 0
		pcs.PcsObjects = []PcsObject{}
		for compObjIndex := 0; compObjIndex < int(compositionObjectCount); compObjIndex++ {
			pcsObj := parsePcs(buffer, offset)
			pcs.PcsObjects = append(pcs.PcsObjects, pcsObj)

			offset += 8
		}
	}

	//TODO: Populate 'Message' field
	//pcs.Message = StringBuilder()
	return pcs
}

func parseSegmentHeader(buffer []byte) (supSegment, error) {
	segment := supSegment{}

	if buffer[0] == 0x50 && buffer[1] == 0x47 { //80 + 71 - P G
		segment.PtsTimestamp = int64(bigEndianInt32(buffer, 2)) //read PTS
		segment.Type = int(buffer[10])
		segment.Size = int(BigEndianInt16(buffer, 11))

		return segment, nil
	}

	return segment, errors.New("unable to read segment, PG missing")
}

func parseSegmentHeaderFromMatroska(buffer []byte) supSegment {
	size := BigEndianInt16(buffer, 1)

	return supSegment{Size: int(size), Type: int(buffer[0])}
}

func BigEndianInt16(buffer []byte, index int) uint16 {
	if len(buffer) < 2 {
		return 0
	}

	return uint16(buffer[index+1]) | (uint16(buffer[index]) << 8)
}

func ParseBluRaySup(buffer []byte, bufferPos int, fromMatroskaFile bool, lastPalettes map[int][]PaletteInfo, bitmapObjects map[int][]OdsData) ([]PcsData, error) {
	forceFirstOds := true
	headerBufferLength := 3
	if !fromMatroskaFile {
		headerBufferLength = headerSize
	}
	headerBuffer := make([]byte, headerBufferLength)
	var latestPcs *PcsData
	palettes := make(map[int][]PaletteInfo)
	pcsList := []PcsData{}
	position := 0
	segmentCount := 0

	for copy(headerBuffer, buffer[position:position+headerBufferLength]) == headerBufferLength {
		position += headerBufferLength

		segment := supSegment{}
		var segmentErr error
		if fromMatroskaFile {
			segment = parseSegmentHeaderFromMatroska(headerBuffer)
		} else {
			segment, segmentErr = parseSegmentHeader(headerBuffer)
			if segmentErr != nil {
				return nil, segmentErr
			}
		}

		//Read segment data
		segmentBuffer := make([]byte, segment.Size)
		bytesCopied := copy(segmentBuffer, buffer[position:position+segment.Size])
		if bytesCopied < segment.Size {
			break
		}
		position += bytesCopied

		switch segment.Type {
		//Palette
		case 0x14:
			if latestPcs != nil {
				pds := parsePds(segmentBuffer, segment)
				if pds.PaletteInfo != nil {
					infos, exists := palettes[pds.Id]

					if !exists {
						palettes[pds.Id] = []PaletteInfo{}
					} else {
						if latestPcs.PaletteUpdate {
							palettes[pds.Id] = infos[:len(infos)-1]
						}
					}

					palettes[pds.Id] = append(infos, *pds.PaletteInfo)
				}
			}
		//Object Definition Segment (image bitmap data)
		case 0x15:
			if latestPcs != nil {
				ods := parseOds(segmentBuffer, segment, forceFirstOds)
				if !latestPcs.PaletteUpdate {
					if ods.IsFirst {
						bitmapObjects[ods.ObjectId] = []OdsData{ods}
					} else {
						odsList, exists := bitmapObjects[ods.ObjectId]
						if exists {
							bitmapObjects[ods.ObjectId] = append(odsList, ods)
						} else {
							//INVALID ObjectId {ods.ObjectId} in ODS, offset={position}
						}
					}
				} else {
					//Bitmap Data Ignore due to PaletteUpdate offset={position}
				}
				forceFirstOds = false
			}
		//Picture time codes
		case 0x16:
			if latestPcs != nil {
				palettesToUse := lastPalettes
				if len(palettes) > 0 {
					palettesToUse = palettes
				}
				if completePcs(latestPcs, bitmapObjects, palettesToUse) {
					pcsList = append(pcsList, *latestPcs)
				}
			}

			forceFirstOds = true
			nextPcs := parsePicture(segmentBuffer, segment)
			if nextPcs.StartTime > 0 && len(pcsList) > 0 && pcsList[len(pcsList)-1].EndTime == 0 {
				pcsList[len(pcsList)-1].EndTime = nextPcs.StartTime
			}

			latestPcs = &nextPcs
			if latestPcs.CompositionState == CompositionStateEpochStart {
				clear(bitmapObjects)
				clear(palettes)
			}
		//Window display
		case 0x17:
			//This only performs logging in the original .NET libse, which we currently do not do
			/*if latestPcs != nil {
				windowCount := int(segmentBuffer[0])
				offset := 0
				for nextWindow := 0; nextWindow < windowCount; nextWindow++ {
					windowId := segmentBuffer[1 + offset]
					x := BigEndianInt16(segmentBuffer, 2 + offset)
					y := BigEndianInt16(segmentBuffer, 4 + offset)
					width := BigEndianInt16(segmentBuffer, 6 + offset)
					height := BigEndianInt16(segmentBuffer, 8 + offset)
					offset += 9
				}
			}*/
		case 0x80:
			forceFirstOds = true

			if latestPcs != nil {
				palettesToUse := lastPalettes
				if len(palettes) > 0 {
					palettesToUse = palettes
				}
				if completePcs(latestPcs, bitmapObjects, palettesToUse) {
					pcsList = append(pcsList, *latestPcs)
				}
				latestPcs = nil
			}
		default:
			//0x?? - END offset={position} UNKNOWN SEGMENT TYPE={segment.Type}
		}

		segmentCount++

		if position+headerBufferLength >= len(buffer) {
			break
		}
	}

	if latestPcs != nil {
		palettesToUse := lastPalettes
		if len(palettes) > 0 {
			palettesToUse = palettes
		}
		if completePcs(latestPcs, bitmapObjects, palettesToUse) {
			pcsList = append(pcsList, *latestPcs)
		}
	}

	for pcsIndex := 1; pcsIndex < len(pcsList); pcsIndex++ {
		prev := pcsList[pcsIndex-1]
		if prev.EndTime == 0 {
			prev.EndTime = pcsList[pcsIndex].StartTime
		}
	}

	pcsList = slices.Collect(func(yield func(PcsData) bool) {
		for _, pcsData := range pcsList {
			if len(pcsData.PcsObjects) > 0 {
				if !yield(pcsData) {
					return
				}
			}
		}
	})

	for _, pcs := range pcsList {
		for i, odsList := range pcs.BitmapObjects {
			if len(odsList) <= 1 {
				continue
			}

			bufSize := 0
			for _, ods := range odsList {
				bufSize += ods.Fragment.ImagePacketSize
			}

			buf := make([]byte, bufSize)
			offset := 0
			for _, ods := range odsList {
				offset += copy(buf[offset:], ods.Fragment.ImageBuffer[:ods.Fragment.ImagePacketSize])
			}
			odsList[0].Fragment.ImageBuffer = buf
			odsList[0].Fragment.ImagePacketSize = bufSize

			pcs.BitmapObjects[i] = []OdsData{odsList[0]}

			break
		}
	}

	//TODO: if !Configuration.Settings.SubtitleSettings.BluRaySupSkipMerge || Configuration.Settings.SubtitleSettings.BluRaySupForceMergeAll

	if lastPalettes != nil && len(palettes) > 0 {
		clear(lastPalettes)
		for key, value := range palettes {
			lastPalettes[key] = value
		}
	}

	return pcsList, nil
}

func ParseBluRaySupFromMatroska(matroskaSubtitleInfo matroska.MatroskaTrackInfo, matroska matroska.MatroskaFile) ([]PcsData, error) {
	subtitle, subtitleErr := matroska.Subtitle(uint64(matroskaSubtitleInfo.TrackNumber), nil)
	if subtitleErr != nil {
		return nil, errors.Wrap(subtitleErr, "failed to retrieve BluRaySup subtitle")
	}

	lastBitmapObjects := make(map[int][]OdsData)
	lastPalettes := make(map[int][]PaletteInfo)
	returnSubtitles := []PcsData{}

	for _, line := range subtitle {
		buffer, bufferErr := line.UncompressedData(matroskaSubtitleInfo)
		if bufferErr != nil {
			return nil, errors.Wrap(bufferErr, "failed to read uncompressed subtitle data")
		}

		if len(buffer) > 2 {
			if !containsBluRayStartSegment(buffer) {
				continue
			}

			lastSubtitleIndex := len(returnSubtitles) - 1

			if len(returnSubtitles) > 0 && returnSubtitles[lastSubtitleIndex].StartTime == returnSubtitles[lastSubtitleIndex].EndTime {
				returnSubtitles[lastSubtitleIndex].EndTime = (line.Start - 1) * 90
			}

			list, listErr := ParseBluRaySup(buffer, 0, true, lastPalettes, lastBitmapObjects)
			if listErr != nil {
				return nil, errors.Wrap(listErr, "failed to parse BluRaySup")
			}

			for _, sup := range list {
				sup.StartTime = (line.Start - 1) * 90
				sup.EndTime = (line.End() - 1) * 90
				returnSubtitles = append(returnSubtitles, sup)

				returnSubLen := len(returnSubtitles)

				//fix overlapping
				if returnSubLen > 1 && subtitle[returnSubLen-2].End() > subtitle[returnSubLen-1].Start {
					returnSubtitles[returnSubLen-2].EndTime = returnSubtitles[returnSubLen-1].StartTime - 1
				}
			}
		} else if len(returnSubtitles) > 0 {
			lastSub := returnSubtitles[len(returnSubtitles)-1]
			if lastSub.StartTime == lastSub.EndTime {
				lastSub.EndTime = (line.Start - 1) * 90
				if lastSub.EndTime-lastSub.StartTime > 1000000 {
					lastSub.EndTime = lastSub.StartTime
				}
			}
		}
	}

	return returnSubtitles, nil
}
