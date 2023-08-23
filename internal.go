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

// Note: we *must* use the "path" package rather than "filepath" to ensure consistency with Windows

type InternalURLProvider interface {
	OpenPath(context contextpkg.Context, path string) (io.ReadCloser, error)
}

var internal sync.Map // []byte or InternalURLProvider

// `content` can be []byte or an InternalURLProvider.
// Other types will be converted to string and then to []byte.
func RegisterInternalURL(path string, content any) error {
	if _, loaded := internal.LoadOrStore(path, fixInternalUrlContent(content)); !loaded {
		return nil
	} else {
		return fmt.Errorf("internal URL conflict: %s", path)
	}
}

func DeregisterInternalURL(path string) {
	internal.Delete(path)
}

func UpdateInternalURL(path string, content any) {
	internal.Store(path, fixInternalUrlContent(content))
}

func (self *Context) ReadToInternalURL(path string, reader io.Reader) (*InternalURL, error) {
	if closer, ok := reader.(io.Closer); ok {
		defer closer.Close()
	}
	if buffer, err := io.ReadAll(reader); err == nil {
		if err = RegisterInternalURL(path, buffer); err != nil {
			return nil, err
		}
	} else {
		return nil, err
	}
	return self.NewValidInternalURL(path)
}

func (self *Context) ReadToInternalURLFromStdin(context contextpkg.Context, format string) (*InternalURL, error) {
	path := fmt.Sprintf("<stdin:%s>", ksuid.New().String())
	if format != "" {
		path = fmt.Sprintf("%s.%s", path, format)
	}
	return self.ReadToInternalURL(path, util.NewContextualReader(context, os.Stdin))
}

func ReadToInternalURLsFromFS(context contextpkg.Context, fs fspkg.FS, root string, process func(path string) (string, bool)) error {
	if root == "" {
		root = "."
	}

	embedFs, isEmbedFs := fs.(embed.FS)

	return fspkg.WalkDir(fs, root, func(path string, dirEntry fspkg.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !dirEntry.IsDir() {
			if internalPath, ok := process(path); ok {
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

//
// InternalURL
//

type InternalURL struct {
	Path    string
	Content any // []byte or InternalURLProvider

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

func (self *InternalURL) NewValidRelativeInternalURL(path string) (*InternalURL, error) {
	return self.urlContext.NewValidInternalURL(pathpkg.Join(self.Path, path))
}

func (self *InternalURL) SetContent(content any) {
	self.Content = fixInternalUrlContent(content)
}

// URL interface
// fmt.Stringer interface
func (self *InternalURL) String() string {
	return self.Key()
}

// URL interface
func (self *InternalURL) Format() string {
	return GetFormat(self.Path)
}

// URL interface
func (self *InternalURL) Origin() URL {
	path := pathpkg.Dir(self.Path)
	if path != "/" {
		path += "/"
	}

	return &InternalURL{
		Path:       path,
		urlContext: self.urlContext,
	}
}

// URL interface
func (self *InternalURL) Relative(path string) URL {
	return self.urlContext.NewInternalURL(pathpkg.Join(self.Path, path))
}

// URL interface
func (self *InternalURL) Key() string {
	return "internal:" + self.Path
}

// URL interface
func (self *InternalURL) Open(context contextpkg.Context) (io.ReadCloser, error) {
	content := self.Content

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

// URL interface
func (self *InternalURL) Context() *Context {
	return self.urlContext
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
