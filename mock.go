package exturl

import (
	"bytes"
	contextpkg "context"
	"io"
	pathpkg "path"
)

//
// MockURL
//

type MockURL struct {
	Scheme  string
	Path    string
	Content any // []byte or InternalURLProvider

	urlContext *Context
}

// "content" can be []byte or an [InternalURLProvider].
// Other types will be converted to string and then to []byte.
func (self *Context) NewMockURL(scheme string, path string, content any) *MockURL {
	return &MockURL{
		Scheme:     scheme,
		Path:       path,
		Content:    fixInternalUrlContent(content),
		urlContext: self,
	}
}

// ([URL] interface, [fmt.Stringer] interface)
func (self *MockURL) String() string {
	return self.Key()
}

// ([URL] interface)
func (self *MockURL) Format() string {
	return GetFormat(self.Path)
}

// ([URL] interface)
func (self *MockURL) Base() URL {
	path := pathpkg.Dir(self.Path)
	if path != "/" {
		path += "/"
	}

	return &MockURL{
		Scheme:     self.Scheme,
		Path:       path,
		Content:    self.Content,
		urlContext: self.urlContext,
	}
}

// ([URL] interface)
func (self *MockURL) Relative(path string) URL {
	return &MockURL{
		Scheme:     self.Scheme,
		Path:       pathpkg.Join(self.Path, path),
		Content:    self.Content,
		urlContext: self.urlContext,
	}
}

// ([URL] interface)
func (self *MockURL) ValidRelative(context contextpkg.Context, path string) (URL, error) {
	return self.Relative(path), nil
}

// ([URL] interface)
func (self *MockURL) Key() string {
	return self.Scheme + ":" + self.Path
}

// ([URL] interface)
func (self *MockURL) Open(context contextpkg.Context) (io.ReadCloser, error) {
	if provider, ok := self.Content.(InternalURLProvider); ok {
		return provider.OpenPath(context, self.Path)
	} else {
		return io.NopCloser(bytes.NewReader(self.Content.([]byte))), nil
	}
}

// ([URL] interface)
func (self *MockURL) Context() *Context {
	return self.urlContext
}

// Updates the contents of this instance only. To change the globally registered
// content use [UpdateInternalURL].
//
// "content" can be []byte or an [InternalURLProvider].
// Other types will be converted to string and then to []byte.
func (self *MockURL) SetContent(content any) {
	self.Content = fixInternalUrlContent(content)
}
