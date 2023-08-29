package exturl

import (
	contextpkg "context"
	"errors"
	"fmt"
	"io"
	neturlpkg "net/url"
	pathpkg "path"
)

// Note: we *must* use the "path" package rather than "filepath" to ensure consistency with Windows

//
// URL
//

type URL interface {
	String() string
	Format() string // yaml, json, xml etc.
	Origin() URL    // base dir, is not necessarily a valid URL
	Relative(path string) URL
	Key() string // for maps
	Open(context contextpkg.Context) (io.ReadCloser, error)
	Context() *Context
}

func (self *Context) NewURL(url string) (URL, error) {
	if url_, ok := self.GetMapping(url); ok {
		url = url_
	}

	if neturl, err := neturlpkg.ParseRequestURI(url); err == nil {
		switch neturl.Scheme {
		case "file":
			return self.NewFileURL(URLPathToFilePath(neturl.Path)), nil

		case "http", "https":
			// Go's "net/http" only handles "http:" and "https:"
			return self.NewNetworkURL(neturl), nil

		case "tar":
			return self.ParseTarballURL(url)

		case "zip":
			return self.ParseZipURL(url)

		case "git":
			return self.ParseGitURL(url)

		case "docker":
			return self.NewDockerURL(neturl), nil

		case "internal":
			return self.NewInternalURL(url[9:]), nil

		case "":
			return self.NewFileURL(url), nil
		}
	}

	return nil, fmt.Errorf("unsupported URL format: %s", url)
}

func (self *Context) NewValidURL(context contextpkg.Context, url string, origins []URL) (URL, error) {
	if url_, ok := self.GetMapping(url); ok {
		url = url_
	}

	if neturl, err := neturlpkg.ParseRequestURI(url); err == nil {
		switch neturl.Scheme {
		case "file":
			return self.newValidRelativeURL(context, URLPathToFilePath(neturl.Path), origins, true)

		case "http", "https":
			// Go's "net/http" package only handles "http:" and "https:"
			return self.NewValidNetworkURL(neturl)

		case "tar":
			return self.ParseValidTarballURL(context, url)

		case "zip":
			return self.ParseValidZipURL(context, url)

		case "git":
			return self.ParseValidGitURL(url)

		case "docker":
			return self.NewValidDockerURL(neturl)

		case "internal":
			return self.NewValidInternalURL(url[9:])

		case "":
			return self.newValidRelativeURL(context, url, origins, false)
		}
	} else {
		// Malformed net URL, so it might be a relative path
		return self.newValidRelativeURL(context, url, origins, false)
	}

	return nil, fmt.Errorf("unsupported URL format: %s", url)
}

func (self *Context) newValidRelativeURL(context contextpkg.Context, path string, origins []URL, onlyFileURLs bool) (URL, error) {
	// Absolute file path?
	if pathpkg.IsAbs(path) {
		url, err := self.NewValidFileURL(path)
		if err != nil {
			return nil, err
		}
		return url, nil
	} else {
		// Try relative to origins
		for _, origin := range origins {
			var url URL
			err := errors.New("")

			switch origin_ := origin.(type) {
			case *FileURL:
				url, err = origin_.NewValidRelativeFileURL(path)

			case *NetworkURL:
				if !onlyFileURLs {
					url, err = origin_.NewValidRelativeNetworkURL(path)
				}

			case *TarballURL:
				if !onlyFileURLs {
					url, err = origin_.NewValidRelativeTarballURL(context, path)
				}

			case *ZipURL:
				if !onlyFileURLs {
					url, err = origin_.NewValidRelativeZipURL(context, path)
				}

			case *GitURL:
				if !onlyFileURLs {
					url, err = origin_.NewValidRelativeGitURL(path)
				}

			case *InternalURL:
				if !onlyFileURLs {
					url, err = origin_.NewValidRelativeInternalURL(path)
				}
			}

			if err == nil {
				return url, nil
			}
		}

		/* Security problem!
		// Try file relative to current directory
		url, err := self.NewValidFileURL(path)
		if err != nil {
			return nil, NewNotFoundf("URL not found: %s", path)
		}

		return url, nil
		*/

		return nil, fmt.Errorf("invalid URL: %s", path)
	}
}
