//go:build wasip1

package exturl

import (
	contextpkg "context"
	"io"
	neturlpkg "net/url"
)

//
// DockerURL
//

type DockerURL struct {
	urlContext *Context
}

func (self *Context) NewDockerURL(neturl *neturlpkg.URL) *DockerURL {
	return &DockerURL{self}
}

func (self *Context) NewValidDockerURL(neturl *neturlpkg.URL) (*DockerURL, error) {
	return nil, NewNotImplemented("NewValidDockerURL")
}

// URL interface
func (self *DockerURL) String() string {
	return ""
}

// URL interface
func (self *DockerURL) Format() string {
	return ""
}

// URL interface
func (self *DockerURL) Origin() URL {
	return self
}

// URL interface
func (self *DockerURL) Relative(path string) URL {
	return self
}

// URL interface
func (self *DockerURL) Key() string {
	return ""
}

// URL interface
func (self *DockerURL) Open(context contextpkg.Context) (io.ReadCloser, error) {
	return nil, NewNotImplemented("DockerURL.Open")
}

// URL interface
func (self *DockerURL) Context() *Context {
	return self.urlContext
}
