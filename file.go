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

// TODO: support "dir packages" with ! character, e.g:
// file:///mydir!this/is/the/file
// Is this really necessary?

//
// FileURL
//

type FileURL struct {
	// This is an absolute OS file path.
	//
	// That means that when compiled on Windows it will expect and use
	// backslashes as path separators in addition to other Windows
	// filesystem convnentions.
	Path string

	urlContext *Context
}

// Note that the argument is treated as an OS file path
// (using backslashes on Windows). The path must be absolute.
//
// Directories must be suffixed with an OS path separator.
func (self *Context) NewFileURL(filePath string) *FileURL {
	return &FileURL{
		Path:       filePath,
		urlContext: self,
	}
}

// Note that the argument is treated as an OS file path
// (using backslashes on Windows). The path must be absolute.
//
// If the path is a directory, it will automatically be suffixed with
// an OS path separator if it doesn't already have one.
func (self *Context) NewValidFileURL(filePath string) (*FileURL, error) {
	if !filepath.IsAbs(filePath) {
		return nil, fmt.Errorf("file URL path is not absolute: %s", filePath)
	}

	isDir := strings.HasSuffix(filePath, PathSeparator)

	filePath = filepath.Clean(filePath)

	if isDir && !strings.HasSuffix(filePath, PathSeparator) {
		filePath += PathSeparator
	}

	if info, err := os.Stat(filePath); err == nil {
		if isDir {
			if !info.Mode().IsDir() {
				return nil, fmt.Errorf("file URL path does not point to a directory: %s", filePath)
			}
		} else if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("file URL path does not point to a file: %s", filePath)
		}
	} else if os.IsNotExist(err) {
		return nil, fmt.Errorf("file URL path not found: %s", filePath)
	} else {
		return nil, err
	}

	return self.NewFileURL(filePath), nil
}

// A valid URL for the working directory.
func (self *Context) NewWorkingDirFileURL() (*FileURL, error) {
	if path, err := os.Getwd(); err == nil {
		return self.NewValidFileURL(path + PathSeparator)
	} else {
		return nil, err
	}
}

// ([fmt.Stringer] interface)
func (self *FileURL) String() string {
	return self.Key()
}

// ([URL] interface)
func (self *FileURL) Format() string {
	return GetFormat(self.Path)
}

// ([URL] interface)
func (self *FileURL) Base() URL {
	path := filepath.Dir(self.Path)
	if path != PathSeparator {
		path += PathSeparator
	}

	return &FileURL{
		Path:       path,
		urlContext: self.urlContext,
	}
}

// Note that the argument can be a URL-type path or an OS file path
// (using backslashes on Windows).
//
// Directories must be suffixed with an OS path separator.
//
// ([URL] interface)
func (self *FileURL) Relative(path string) URL {
	return self.urlContext.NewFileURL(self.relative(path))
}

// Note that the argument can be a URL-type path or an OS file path
// (using backslashes on Windows).
//
// If the path is a directory, it will automatically be suffixed with
// an OS path separator if it doesn't already have one.
//
// ([URL] interface)
func (self *FileURL) ValidRelative(context contextpkg.Context, filePath string) (URL, error) {
	return self.urlContext.NewValidFileURL(self.relative(filePath))
}

// ([URL] interface)
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
		return self.Path
	}
}

// ([URL] interface)
func (self *FileURL) Open(context contextpkg.Context) (io.ReadCloser, error) {
	if reader, err := os.Open(self.Path); err == nil {
		return reader, nil
	} else {
		return nil, err
	}
}

// ([URL] interface)
func (self *FileURL) Context() *Context {
	return self.urlContext
}

// Utils

func (self *FileURL) relative(path string) string {
	isDir := strings.HasSuffix(path, PathSeparator)
	path = filepath.Join(self.Path, path)
	if isDir {
		// filepath.Join removes path suffixes
		path += PathSeparator
	}
	return path
}

func URLPathToFilePath(path string) string {
	if PathSeparator == `\` {
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
