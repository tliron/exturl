package exturl

import (
	contextpkg "context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const PathSeparator = string(filepath.Separator)

//
// FileURL
//

type FileURL struct {
	Path string

	urlContext *Context
}

func (self *Context) NewFileURL(path string) *FileURL {
	return &FileURL{
		Path:       path,
		urlContext: self,
	}
}

func (self *Context) NewValidFileURL(path string) (*FileURL, error) {
	isDir := strings.HasSuffix(path, PathSeparator)

	if filepath.IsAbs(path) {
		path = filepath.Clean(path)
	} else {
		var err error
		if path, err = filepath.Abs(path); err != nil {
			return nil, err
		}
	}

	if isDir && !strings.HasSuffix(path, PathSeparator) {
		path += PathSeparator
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
	isDir := strings.HasSuffix(path, PathSeparator)
	path = filepath.Join(self.Path, path)
	if isDir {
		path += PathSeparator
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
	if path != PathSeparator {
		path += PathSeparator
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
	path := filepath.ToSlash(self.Path)
	if filepath.IsAbs(self.Path) {
		if strings.HasPrefix(path, "/") {
			return "file://" + path
		} else {
			// On Windows absolute paths usually do not start with a separator, e.g. "C:\Abs\Path"
			return "file:///" + path
		}
	} else {
		// The "file:" schema does not support relative paths
		return path
	}
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

// Utils

func URLPathToFilePath(path string) string {
	if filepath.Separator == '\\' {
		// We don't want the "/" prefix on Windows
		if strings.HasPrefix(path, "/") {
			path = path[1:]
		}
	}
	path = filepath.FromSlash(path)
	return path
}

func isValidFile(path string) bool {
	if info, err := os.Stat(path); err == nil {
		return info.Mode().IsRegular()
	} else {
		return false
	}
}
