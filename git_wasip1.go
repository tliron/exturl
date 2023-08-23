//go:build wasip1

package exturl

//
// GitURL
//

type GitURL struct {
	*MockURL
}

func (self *Context) NewGitURL(path string, repositoryUrl string) *GitURL {
	return &GitURL{self.NewMockURL("git", path, nil)}
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
