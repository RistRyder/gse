package boxes

import (
	"github.com/cockroachdb/errors"
	"github.com/ristryder/gse/common"
)

type Box struct {
	Buffer        []byte
	Name          string
	Position      uint64
	Size          uint64
	StartPosition uint64
}

func (b *Box) initSizeAndName(file *common.FileStream) (bool, error) {
	if b.StartPosition == 0 {
		b.StartPosition = uint64(file.Position()) - 8
	}

	b.Buffer = make([]byte, 8)
	bytesRead, readErr := file.Read(b.Buffer)
	if readErr != nil {
		return false, errors.Wrap(readErr, "failed to read mp4 file")
	}
	if bytesRead < len(b.Buffer) {
		return false, errors.Newf("expected %d bytes but read out %d from mp4 file", len(b.Buffer), bytesRead)
	}

	b.Size = uint64(b.UInt(0))
	b.Name = b.Str(4, 4)

	if b.Size == 0 {
		b.Size = uint64(file.Size() - file.Position())
	}
	if b.Size == 1 {
		bytesRead, readErr := file.Read(b.Buffer)
		if readErr != nil {
			return false, errors.Wrap(readErr, "failed to read mp4 file")
		}
		if bytesRead < len(b.Buffer) {
			return false, errors.Newf("expected %d bytes but read out %d from mp4 file", len(b.Buffer), bytesRead)
		}

		b.Size = b.UInt64(0) - 8
	}

	b.Position = uint64(file.Position()) + b.Size - 8

	return true, nil
}

func (b *Box) Int(index int) int32 {
	return (int32(b.Buffer[index]) << 24) + (int32(b.Buffer[index+1]) << 16) + (int32(b.Buffer[index+2]) << 8) + int32(b.Buffer[index+3])
}

func (b *Box) Str(index, count int) string {
	return string(b.Buffer[index : index+count])
}

func (b *Box) UInt(index int) uint32 {
	return (uint32(b.Buffer[index]) << 24) + (uint32(b.Buffer[index+1]) << 16) + (uint32(b.Buffer[index+2]) << 8) + uint32(b.Buffer[index+3])
}

func (b *Box) UInt64(index int) uint64 {
	return (uint64(b.Buffer[index]) << 56) + (uint64(b.Buffer[index+1]) << 48) | (uint64(b.Buffer[index+2]) << 40) | (uint64(b.Buffer[index+3]) << 32) | (uint64(b.Buffer[index+4]) << 24) | (uint64(b.Buffer[index+5]) << 16) | (uint64(b.Buffer[index+6]) << 8) | uint64(b.Buffer[index+7])
}

func (b *Box) Word(index int) int16 {
	return (int16(b.Buffer[index]) << 8) + int16(b.Buffer[index+1])
}
