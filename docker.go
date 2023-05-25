package exturl

import (
	contextpkg "context"
	"fmt"
	"io"
	neturlpkg "net/url"
	"path"

	"github.com/google/go-containerregistry/pkg/authn"
	namepkg "github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

//
// DockerURL
//

type DockerURL struct {
	URL *neturlpkg.URL

	string_    string
	urlContext *Context
}

func (self *Context) NewDockerURL(neturl *neturlpkg.URL) *DockerURL {
	if self == nil {
		self = NewContext()
	}

	return &DockerURL{
		URL:        neturl,
		string_:    neturl.String(),
		urlContext: self,
	}
}

func (self *Context) NewValidDockerURL(neturl *neturlpkg.URL) (*DockerURL, error) {
	if (neturl.Scheme != "docker") && (neturl.Scheme != "") {
		return nil, fmt.Errorf("not a docker URL: %s", neturl.String())
	}

	// TODO

	return self.NewDockerURL(neturl), nil
}

// URL interface
// fmt.Stringer interface
func (self *DockerURL) String() string {
	return self.Key()
}

// URL interface
func (self *DockerURL) Format() string {
	format := self.URL.Query().Get("format")
	if format != "" {
		return format
	} else {
		return GetFormat(self.URL.Path)
	}
}

// URL interface
func (self *DockerURL) Origin() URL {
	url := *self
	url.URL.Path = path.Dir(url.URL.Path)
	if url.URL.Path != "/" {
		url.URL.Path += "/"
	}
	// TODO: url.URL.RawPath?
	return &url
}

// URL interface
func (self *DockerURL) Relative(path string) URL {
	if neturl, err := neturlpkg.Parse(path); err == nil {
		return self.urlContext.NewDockerURL(self.URL.ResolveReference(neturl))
	} else {
		return nil
	}
}

// URL interface
func (self *DockerURL) Key() string {
	return self.string_
}

// URL interface
func (self *DockerURL) Open(context contextpkg.Context) (io.ReadCloser, error) {
	pipeReader, pipeWriter := io.Pipe()

	go func() {
		if err := self.WriteLayer(pipeWriter); err == nil {
			pipeWriter.Close()
		} else {
			pipeWriter.CloseWithError(err)
		}
	}()

	return pipeReader, nil
}

// URL interface
func (self *DockerURL) Context() *Context {
	return self.urlContext
}

func (self *DockerURL) WriteTarball(writer io.Writer) error {
	url := fmt.Sprintf("%s%s", self.URL.Host, self.URL.Path)
	if tag, err := namepkg.NewTag(url); err == nil {
		if image, err := remote.Image(tag, self.RemoteOptions()...); err == nil {
			return tarball.Write(tag, image, writer)
		} else {
			return err
		}
	} else {
		return err
	}
}

func (self *DockerURL) WriteLayer(writer io.Writer) error {
	pipeReader, pipeWriter := io.Pipe()

	go func() {
		if err := self.WriteTarball(pipeWriter); err == nil {
			pipeWriter.Close()
		} else {
			pipeWriter.CloseWithError(err)
		}
	}()

	decoder := NewContainerImageLayerDecoder(pipeReader)
	if _, err := io.Copy(writer, decoder.Decode()); err == nil {
		return nil
	} else {
		return err
	}
}

func (self *DockerURL) RemoteOptions() []remote.Option {
	var options []remote.Option

	if httpRoundTripper := self.urlContext.GetHTTPRoundTripper(self.URL.Host); httpRoundTripper != nil {
		options = append(options, remote.WithTransport(httpRoundTripper))
	}

	if credentials := self.urlContext.GetCredentials(self.URL.Host); credentials != nil {
		authenticator := authn.FromConfig(authn.AuthConfig{
			Username:      credentials.Username,
			Password:      credentials.Password,
			RegistryToken: credentials.Token,
		})
		options = append(options, remote.WithAuth(authenticator))
	}

	return options
}
