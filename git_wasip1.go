//go:build wasip1

package exturl

import (
	contextpkg "context"
	"io"
)

//
// GitURL
//

type GitURL struct {
	urlContext *Context
}

func (self *Context) NewGitURL(path string, repositoryUrl string) *GitURL {
	return &GitURL{self}
}

func (self *Context) NewValidGitURL(path string, repositoryUrl string) (*GitURL, error) {
	return nil, NewNotImplemented("NewValidGitURL")
}

func (self *GitURL) NewValidRelativeGitURL(path string) (*GitURL, error) {
	return nil, NewNotImplemented("NewValidRelativeGitURL")
}

func (self *Context) ParseGitURL(url string) (*GitURL, error) {
	return nil, NewNotImplemented("ParseGitURL")
}

func (self *Context) ParseValidGitURL(url string) (*GitURL, error) {
	return nil, NewNotImplemented("ParseValidGitURL")
}

// URL interface
func (self *GitURL) String() string {
	return ""
}

// URL interface
func (self *GitURL) Format() string {
	return ""
}

// URL interface
func (self *GitURL) Origin() URL {
	return self
}

// URL interface
func (self *GitURL) Relative(path string) URL {
	return self
}

// URL interface
func (self *GitURL) Key() string {
	return ""
}

// URL interface
func (self *GitURL) Open(context contextpkg.Context) (io.ReadCloser, error) {
	return nil, NewNotImplemented("GitURL.Open")
}

// URL interface
func (self *GitURL) Context() *Context {
	return self.urlContext
}
