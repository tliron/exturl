package exturl

import (
	"bytes"
	contextpkg "context"
	"fmt"
	"io"
	"os"
	pathpkg "path"
	"sync"

	"github.com/segmentio/ksuid"
	"github.com/tliron/kutil/util"
)

// Note: we *must* use the "path" package rather than "filepath" to ensure consistency with Windows

var internal sync.Map

// `content` must be []byte or string
func RegisterInternalURL(path string, content any) error {
	if _, loaded := internal.LoadOrStore(path, util.ToBytes(content)); !loaded {
		return nil
	} else {
		return fmt.Errorf("internal URL conflict: %s", path)
	}
}

func DeregisterInternalURL(path string) {
	internal.Delete(path)
}

func UpdateInternalURL(path string, content any) {
	internal.Store(path, util.ToBytes(content))
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

//
// InternalURL
//

type InternalURL struct {
	Path    string
	Content []byte

	urlContext *Context
}

func (self *Context) NewInternalURL(path string) *InternalURL {
	return &InternalURL{
		Path:       path,
		urlContext: self,
	}
}

func (self *Context) NewValidInternalURL(path string) (*InternalURL, error) {
	if content, ok := internal.Load(path); ok {
		if self == nil {
			self = NewContext()
		}

		return &InternalURL{
			Path:       path,
			Content:    content.([]byte),
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
	self.Content = util.ToBytes(content)
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
	return io.NopCloser(bytes.NewReader(self.Content)), nil
}

// URL interface
func (self *InternalURL) Context() *Context {
	return self.urlContext
}
