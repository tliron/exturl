package exturl

import (
	"bytes"
	contextpkg "context"
	"embed"
	"fmt"
	"io"
	fspkg "io/fs"
	"os"
	pathpkg "path"
	"sync"

	"github.com/segmentio/ksuid"
	"github.com/tliron/kutil/util"
)

type InternalURLProvider interface {
	OpenPath(context contextpkg.Context, path string) (io.ReadCloser, error)
}

var internal sync.Map // []byte or InternalURLProvider

// Registers content for [InternalURL].
//
// "content" can be []byte or an [InternalURLProvider].
// Other types will be converted to string and then to []byte.
//
// Will return an error if there is already content registered at the path.
// For a version that always succeeds, use [UpdateInternalURL].
func RegisterInternalURL(path string, content any) error {
	if _, loaded := internal.LoadOrStore(path, fixInternalUrlContent(content)); !loaded {
		return nil
	} else {
		return fmt.Errorf("internal URL conflict: %s", path)
	}
}

// Deletes registers content for [InternalURL] or does nothing if the content is
// not registered.
func DeregisterInternalURL(path string) {
	internal.Delete(path)
}

// Updates registered content for [InternalURL] or registers it if not yet
// registered.
//
// "content" can be []byte or an [InternalURLProvider].
// Other types will be converted to string and then to []byte.
func UpdateInternalURL(path string, content any) {
	internal.Store(path, fixInternalUrlContent(content))
}

// Walks a [io/fs.FS] starting at "root" and registers its files' content for
// [InternalURL]. "root" can be an emptry string to read the entire FS.
//
// The optional argument "translatePath" is a function that can translate the FS
// path to an internal URL path and can also filter out entries when it returns
// false. If the argument is not provided all files will be read with their paths
// as is.
//
// If "fs" is an [embed.FS], this function is optimized to handle it more
// efficiently via calls to [embed.FS.ReadFile].
//
// Will return an error if there is already content registered at an internal path.
func ReadToInternalURLsFromFS(context contextpkg.Context, fs fspkg.FS, root string, translatePath func(path string) (string, bool)) error {
	if root == "" {
		root = "."
	}

	if translatePath == nil {
		translatePath = func(path string) (string, bool) {
			return path, true
		}
	}

	embedFs, isEmbedFs := fs.(embed.FS)

	return fspkg.WalkDir(fs, root, func(path string, dirEntry fspkg.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !dirEntry.IsDir() {
			if internalPath, ok := translatePath(path); ok {
				if isEmbedFs {
					// Optimized read for embed.FS
					if content, err := embedFs.ReadFile(path); err == nil {
						if err := RegisterInternalURL(internalPath, content); err != nil {
							return err
						}
					} else {
						return err
					}
				} else {
					if file, err := fs.Open(path); err == nil {
						reader := util.NewContextualReadCloser(context, file)
						defer reader.Close()
						if content, err := io.ReadAll(reader); err == nil {
							if err := RegisterInternalURL(internalPath, content); err != nil {
								return err
							}
						} else {
							return err
						}
					} else {
						return err
					}
				}
			}
		}

		return nil
	})
}

// Registers content for [InternalURL] from an [io.Reader].
//
// Will automatically close "reader" if it supports [io.Closer].
//
// Will return an error if there is already content registered at the path.
func (self *Context) ReadToInternalURL(path string, reader io.Reader) (*InternalURL, error) {
	if closer, ok := reader.(io.Closer); ok {
		defer closer.Close()
	}

	if buffer, err := io.ReadAll(reader); err == nil {
		if err = RegisterInternalURL(path, buffer); err == nil {
			return &InternalURL{
				Path:            path,
				OverrideContent: buffer,
				urlContext:      self,
			}, nil
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}

// Registers content for [InternalURL] from [os.Stdin]. "format" can be an empty
// string.
//
// The URL will have a globally unique path in the form of "/stdin/GUID[.FORMAT]".
// The "format" extension is used to support URL.Format.
func (self *Context) ReadToInternalURLFromStdin(context contextpkg.Context, format string) (*InternalURL, error) {
	path := "/stdin/" + ksuid.New().String()
	if format != "" {
		path += "." + format
	}
	return self.ReadToInternalURL(path, util.NewContextualReader(context, os.Stdin))
}

//
// InternalURL
//

type InternalURL struct {
	Path string

	// If explicitly set to a non-nil value then it will override any globally
	// registered content when calling InternalURL.Open.
	//
	// []byte or InternalURLProvider.
	OverrideContent any

	urlContext *Context
}

func (self *Context) NewInternalURL(path string) *InternalURL {
	return &InternalURL{
		Path:       path,
		urlContext: self,
	}
}

func (self *Context) NewValidInternalURL(path string) (*InternalURL, error) {
	if _, ok := internal.Load(path); ok {
		return &InternalURL{
			Path:       path,
			urlContext: self,
		}, nil
	} else {
		return nil, NewNotFoundf("internal URL not found: %s", path)
	}
}

// ([fmt.Stringer] interface)
func (self *InternalURL) String() string {
	return self.Key()
}

// ([URL] interface)
func (self *InternalURL) Format() string {
	return GetFormat(self.Path)
}

// ([URL] interface)
func (self *InternalURL) Base() URL {
	path := pathpkg.Dir(self.Path)
	if path != "/" {
		path += "/"
	}

	return &InternalURL{
		Path:       path,
		urlContext: self.urlContext,
	}
}

// ([URL] interface)
func (self *InternalURL) Relative(path string) URL {
	return self.urlContext.NewInternalURL(pathpkg.Join(self.Path, path))
}

// ([URL] interface)
func (self *InternalURL) ValidRelative(context contextpkg.Context, path string) (URL, error) {
	return self.urlContext.NewValidInternalURL(pathpkg.Join(self.Path, path))
}

// ([URL] interface)
func (self *InternalURL) Key() string {
	return "internal:" + self.Path
}

// If InternalURL.Content was set to a non-nil value, then it will be used.
// Otherwise will attempt to retrieve the globally registered content.
//
// ([URL] interface)
func (self *InternalURL) Open(context contextpkg.Context) (io.ReadCloser, error) {
	content := self.OverrideContent

	if content == nil {
		var ok bool
		if content, ok = internal.Load(self.Path); !ok {
			return nil, NewNotFoundf("internal URL not found: %s", self.Path)
		}
	}

	if provider, ok := content.(InternalURLProvider); ok {
		return provider.OpenPath(context, self.Path)
	} else {
		return io.NopCloser(bytes.NewReader(content.([]byte))), nil
	}
}

// ([URL] interface)
func (self *InternalURL) Context() *Context {
	return self.urlContext
}

// Updates the contents of this instance only. To change the globally registered
// content use [UpdateInternalURL].
//
// "content" can be []byte or an [InternalURLProvider].
// Other types will be converted to string and then to []byte.
func (self *InternalURL) SetContent(content any) {
	self.OverrideContent = fixInternalUrlContent(content)
}

// Utils

var emptyByteArray = []byte{}

func fixInternalUrlContent(content any) any {
	if content == nil {
		return emptyByteArray
	} else if _, ok := content.(InternalURLProvider); ok {
		return content
	} else {
		return util.ToBytes(content)
	}
}
