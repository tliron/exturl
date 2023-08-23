//go:build wasip1

package exturl

import (
	neturlpkg "net/url"
)

//
// DockerURL
//

type DockerURL struct {
	*MockURL
}

func (self *Context) NewDockerURL(neturl *neturlpkg.URL) *DockerURL {
	return &DockerURL{self.NewMockURL("docker", neturl.Path, nil)}
}

func (self *Context) NewValidDockerURL(neturl *neturlpkg.URL) (*DockerURL, error) {
	return nil, NewNotImplemented("NewValidDockerURL")
}
