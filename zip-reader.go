package exturl

import (
	"io"
	"os"

	"github.com/klauspost/compress/zip"
)

//
// ZipReader
//

type ZipReader struct {
	ZipReader *zip.Reader
	File      *os.File
}

func NewZipReader(reader *zip.Reader, file *os.File) *ZipReader {
	return &ZipReader{reader, file}
}

func NewZipReaderForFile(file *os.File) (*ZipReader, error) {
	if stat, err := file.Stat(); err == nil {
		size := stat.Size()
		if zipReader, err := zip.NewReader(file, size); err == nil {
			return NewZipReader(zipReader, file), nil
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}

// ([io.Closer] interface)
func (self *ZipReader) Close() error {
	return self.File.Close()
}

func (self *ZipReader) Open(path string) (*ZipEntryReader, error) {
	for _, file := range self.ZipReader.File {
		if path == file.Name {
			if entryReader, err := file.Open(); err == nil {
				return NewZipEntryReader(entryReader, self), nil
			} else {
				return nil, err
			}
		}
	}
	return nil, nil
}

func (self *ZipReader) Has(path string) bool {
	for _, file := range self.ZipReader.File {
		if path == file.Name {
			return true
		}
	}
	return false
}

func (self *ZipReader) Iterate(f func(*zip.File) bool) {
	for _, file := range self.ZipReader.File {
		if !f(file) {
			return
		}
	}
}

//
// ZipEntryReader
//

type ZipEntryReader struct {
	EntryReader io.ReadCloser
	ZipReader   *ZipReader
}

func NewZipEntryReader(entryReader io.ReadCloser, zipReader *ZipReader) *ZipEntryReader {
	return &ZipEntryReader{entryReader, zipReader}
}

// ([io.Reader] interface)
func (self *ZipEntryReader) Read(p []byte) (n int, err error) {
	return self.EntryReader.Read(p)
}

// ([io.Closer] interface)
func (self *ZipEntryReader) Close() error {
	err1 := self.EntryReader.Close()
	err2 := self.ZipReader.Close()
	if err1 != nil {
		return err1
	} else {
		return err2
	}
}
