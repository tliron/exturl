package exturl

import (
	contextpkg "context"
	"fmt"
	"io"
	"net/http"
	neturlpkg "net/url"
	"path"
)

// Note: we *must* use the "path" package rather than "filepath" to ensure consistency with Windows

//
// NetworkURL
//

type NetworkURL struct {
	URL *neturlpkg.URL

	string_    string
	urlContext *Context
}

func (self *Context) NewNetworkURL(neturl *neturlpkg.URL) *NetworkURL {
	return &NetworkURL{
		URL:        neturl,
		string_:    neturl.String(),
		urlContext: self,
	}
}

func (self *Context) NewValidNetworkURL(neturl *neturlpkg.URL) (*NetworkURL, error) {
	string_ := neturl.String()
	if response, err := http.Head(string_); err == nil {
		response.Body.Close()
		if response.StatusCode == http.StatusOK {
			return &NetworkURL{
				URL:        neturl,
				string_:    string_,
				urlContext: self,
			}, nil
		} else {
			return nil, fmt.Errorf("HTTP status: %s", response.Status)
		}
	} else {
		return nil, err
	}
}

func (self *NetworkURL) NewValidRelativeNetworkURL(path string) (*NetworkURL, error) {
	if neturl, err := neturlpkg.Parse(path); err == nil {
		neturl = self.URL.ResolveReference(neturl)
		return self.urlContext.NewValidNetworkURL(neturl)
	} else {
		return nil, err
	}
}

// URL interface
// fmt.Stringer interface
func (self *NetworkURL) String() string {
	return self.Key()
}

// URL interface
func (self *NetworkURL) Format() string {
	format := self.URL.Query().Get("format")
	if format != "" {
		return format
	} else {
		return GetFormat(self.URL.Path)
	}
}

// URL interface
func (self *NetworkURL) Origin() URL {
	url := *self
	url.URL.Path = path.Dir(url.URL.Path)
	if url.URL.Path != "/" {
		url.URL.Path += "/"
	}
	// TODO: url.URL.RawPath?
	return &url
}

// URL interface
func (self *NetworkURL) Relative(path string) URL {
	if neturl, err := neturlpkg.Parse(path); err == nil {
		return self.urlContext.NewNetworkURL(self.URL.ResolveReference(neturl))
	} else {
		return nil
	}
}

// URL interface
func (self *NetworkURL) Key() string {
	return self.string_
}

// URL interface
func (self *NetworkURL) Open(context contextpkg.Context) (io.ReadCloser, error) {
	if response, err := http.Get(self.string_); err == nil {
		if response.StatusCode == http.StatusOK {
			return response.Body, nil
		} else {
			response.Body.Close()
			return nil, fmt.Errorf("HTTP status: %s", response.Status)
		}
	} else {
		return nil, err
	}
}

// URL interface
func (self *NetworkURL) Context() *Context {
	return self.urlContext
}
