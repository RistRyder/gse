package common

import (
	"io"
	"os"

	"github.com/cockroachdb/errors"
	"github.com/edsrzf/mmap-go"
)

type FileStream struct {
	file           *os.File
	filePosition   int64
	fileSize       int64
	isMemoryMapped bool
	isOpen         bool
	mmapFile       mmap.MMap
}

func (f *FileStream) offsetFilePosition(offset int) {
	f.filePosition += int64(offset)
}

func (f *FileStream) Close() error {
	if !f.isOpen {
		return nil
	}

	f.filePosition = -1
	f.fileSize = -1
	f.isOpen = false

	if f.isMemoryMapped {
		return f.mmapFile.Unmap()
	}

	return f.Close()
}

func NewFileStream(path string) (*FileStream, error) {
	file, openErr := os.Open(path)
	if openErr != nil {
		return nil, errors.Wrapf(openErr, "failed to open file %s", path)
	}

	stat, statErr := file.Stat()
	if statErr != nil {
		return nil, errors.Wrapf(statErr, "failed to read information while opening file %s", path)
	}

	mmap, mmapErr := mmap.Map(file, mmap.RDONLY, 0)
	if mmapErr != nil {
		return &FileStream{
			file:           file,
			filePosition:   0,
			fileSize:       stat.Size(),
			isMemoryMapped: false,
			isOpen:         true,
			mmapFile:       nil,
		}, nil
	}

	defer file.Close()

	return &FileStream{
		file:           file,
		filePosition:   0,
		fileSize:       stat.Size(),
		isMemoryMapped: true,
		isOpen:         true,
		mmapFile:       mmap,
	}, nil
}

func (f *FileStream) Position() int64 {
	return f.filePosition
}

func (f *FileStream) Read(b []byte) (int, error) {
	if f.isMemoryMapped {
		if f.filePosition == f.fileSize-1 {
			return 0, io.EOF
		}

		requestedByteCount := int64(len(b))
		endIndex := f.filePosition + requestedByteCount
		var error error

		if f.filePosition+requestedByteCount >= f.fileSize {
			endIndex = f.fileSize
			error = io.EOF
		}

		bytesCopied := copy(b, f.mmapFile[f.filePosition:endIndex])

		f.offsetFilePosition(bytesCopied)

		return bytesCopied, error
	}

	bytesRead, readErr := f.file.Read(b)
	if bytesRead == 0 || (readErr != nil && readErr != io.EOF) {
		return bytesRead, readErr
	}

	f.offsetFilePosition(bytesRead)

	return bytesRead, nil
}

func (f *FileStream) Seek(offset int64, whence int) (int64, error) {
	if f.isMemoryMapped {
		switch whence {
		case io.SeekCurrent:
			f.filePosition += offset
		case io.SeekEnd:
			f.filePosition = f.fileSize + offset
		case io.SeekStart:
			f.filePosition = offset
		}

		return f.filePosition, nil
	}

	newOffset, seekErr := f.file.Seek(offset, whence)
	if seekErr != nil {
		return newOffset, seekErr
	}

	f.filePosition = newOffset

	return f.filePosition, nil
}

func (f *FileStream) Size() int64 {
	return f.fileSize
}
