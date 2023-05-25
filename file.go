package exturl

import (
	contextpkg "context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

//
// FileURL
//

type FileURL struct {
	Path string

	urlContext *Context
}

func (self *Context) NewFileURL(path string) *FileURL {
	if self == nil {
		self = NewContext()
	}

	return &FileURL{
		Path:       path,
		urlContext: self,
	}
}

func (self *Context) NewValidFileURL(path string) (*FileURL, error) {
	isDir := strings.HasSuffix(path, "/")

	if filepath.IsAbs(path) {
		path = filepath.Clean(path)
	} else {
		var err error
		if path, err = filepath.Abs(path); err != nil {
			return nil, err
		}
	}

	if info, err := os.Stat(path); err == nil {
		if isDir {
			if !info.Mode().IsDir() {
				return nil, fmt.Errorf("URL path does not point to a directory: %s", path)
			}
		} else if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("URL path does not point to a file: %s", path)
		}
	} else {
		return nil, err
	}

	return self.NewFileURL(path), nil
}

func (self *FileURL) NewValidRelativeFileURL(path string) (*FileURL, error) {
	isDir := strings.HasSuffix(path, "/")
	path = filepath.Join(self.Path, path)
	if isDir {
		path += "/"
	}
	return self.urlContext.NewValidFileURL(path)
}

// URL interface
// fmt.Stringer interface
func (self *FileURL) String() string {
	return self.Key()
}

// URL interface
func (self *FileURL) Format() string {
	return GetFormat(self.Path)
}

// URL interface
func (self *FileURL) Origin() URL {
	path := filepath.Dir(self.Path)
	if path != "/" {
		path += "/"
	}

	return &FileURL{
		Path:       path,
		urlContext: self.urlContext,
	}
}

// URL interface
func (self *FileURL) Relative(path string) URL {
	return self.urlContext.NewFileURL(filepath.Join(self.Path, path))
}

// URL interface
func (self *FileURL) Key() string {
	return "file:" + self.Path
}

// URL interface
func (self *FileURL) Open(context contextpkg.Context) (io.ReadCloser, error) {
	if reader, err := os.Open(self.Path); err == nil {
		return reader, nil
	} else {
		return nil, err
	}
}

// URL interface
func (self *FileURL) Context() *Context {
	return self.urlContext
}

func isValidFile(path string) bool {
	if info, err := os.Stat(path); err == nil {
		return info.Mode().IsRegular()
	} else {
		return false
	}
}
