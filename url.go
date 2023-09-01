package exturl

import (
	contextpkg "context"
	"fmt"
	"io"
	neturlpkg "net/url"
	"path/filepath"
)

// Note: we *must* use the "path" package here rather than "filepath" to ensure consistency with Windows

//
// URL
//

type URL interface {
	// Returns a string representation of the URL that can be used by [NewURL] to
	// recreate this URL.
	String() string

	// Format of the URL content's default representation.
	//
	// Should return "yaml", "json", "xml", etc., or an empty string if the format
	// is unknown.
	Format() string

	// Returns the equivalent of a "base directory" for the URL.
	//
	// The origin can be used in two ways:
	//
	// 1. You can call Relative on it to get a sibling URL to this one (relative to
	//    the same "base directory").
	// 2. You can use it in the origins list argument of [NewValidURL] for the same
	//    purpose.
	//
	// Note that the origin might not be a valid URL in itself, e.g. you might not
	// be able to call Open on it.
	Origin() URL

	// Parses the argument as a path relative to this URL. That means this this
	// URL is treated as a "directory".
	//
	// Returns an absolute URL.
	Relative(path string) URL

	// Returns a string that uniquely identifies this URL.
	//
	// Useful as map keys.
	Key() string

	// Opens the URL for reading.
	//
	// It is the caller's responsibility to Close the reader.
	Open(context contextpkg.Context) (io.ReadCloser, error)

	// The context used to create this URL.
	Context() *Context
}

// Parses the argument as an absolute URL.
//
// To support relative URLs, see [NewValidURL].
//
// If you are expecting either a URL *or* a file path, consider [NewAnyOrFileURL].
func (self *Context) NewURL(url string) (URL, error) {
	if mappedUrl, ok := self.GetMapping(url); ok {
		url = mappedUrl
	}

	if neturl, err := neturlpkg.ParseRequestURI(url); err == nil {
		switch neturl.Scheme {
		case "file":
			filePath := URLPathToFilePath(neturl.Path)
			return self.NewFileURL(filePath), nil

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

		default:
			return nil, fmt.Errorf("unsupported URL scheme: %q for %s", neturl.Scheme, url)
		}
	} else {
		return nil, fmt.Errorf("malformed URL: %s", url)
	}
}

// Parses the argument as *either* an absolute URL *or* a file path.
//
// In essence attempts to parse the URL via [NewURL] and if that fails
// treats the URL as a file path and returns a *[FileURL].
//
// To support relative URLs, see [NewValidAnyOrFileURL].
//
// On Windows note that if there happens to be a drive that has the same
// name as a supported URL scheme (e.g. "http") then callers would have
// to provide a full file URL, e.g. instead of "http:\Dir\file" provide
// "file:///http:/Dir/file", otherwise it will be parsed as a URL of that
// scheme.
func (self *Context) NewAnyOrFileURL(urlOrPath string) URL {
	if url_, err := self.NewURL(urlOrPath); err == nil {
		return url_
	} else {
		return self.NewFileURL(urlOrPath)
	}
}

// Parses the argument as *either* an absolute URL *or* a relative path.
// Relative paths support ".." and ".", with the returned URL path being
// absolute.
//
// The returned URL is "valid", meaning that during this call it was
// possible to call Open on it. Of course this can't guarantee that
// future calls to Open will succeed.
//
// Relative URLs are tested against the origins argument in order. The
// first valid URL will be returned and the remaining origins will be
// ignored.
//
// If you are expecting either a URL *or* a file path, consider
// [NewValidAnyOrFileURL].
func (self *Context) NewValidURL(context contextpkg.Context, urlOrPath string, origins []URL) (URL, error) {
	return self.newValidUrl(context, urlOrPath, origins, false)
}

// Parses the argument as an absolute URL *or* an absolute file path
// *or* a relative path. Relative paths support ".." and ".", with the
// returned URL path being absolute.
//
// The returned URL is "valid", meaning that during this call it was
// possible to call Open on it. Of course this can't guarantee that
// future calls to Open will succeed.
//
// Relative URLs are tested against the origins argument in order. The
// first valid URL will be returned and the remaining origins will be
// ignored.
func (self *Context) NewValidAnyOrFileURL(context contextpkg.Context, urlOrPath string, origins []URL) (URL, error) {
	return self.newValidUrl(context, urlOrPath, origins, true)
}

func (self *Context) newValidUrl(context contextpkg.Context, urlOrPath string, origins []URL, orFile bool) (URL, error) {
	if mappedUrl, ok := self.GetMapping(urlOrPath); ok {
		urlOrPath = mappedUrl
	}

	if neturl, err := neturlpkg.ParseRequestURI(urlOrPath); err == nil {
		switch neturl.Scheme {
		case "file":
			filePath := URLPathToFilePath(neturl.Path)
			return self.NewValidFileURL(filePath)

		case "http", "https":
			// Go's "net/http" only handles "http:" and "https:"
			return self.NewValidNetworkURL(neturl)

		case "tar":
			return self.ParseValidTarballURL(context, urlOrPath)

		case "zip":
			return self.ParseValidZipURL(context, urlOrPath)

		case "git":
			return self.ParseValidGitURL(urlOrPath)

		case "docker":
			return self.NewValidDockerURL(neturl)

		case "internal":
			return self.NewValidInternalURL(urlOrPath[9:])

		case "":

		default:
			return nil, fmt.Errorf("unsupported URL scheme: %q for %s", neturl.Scheme, urlOrPath)
		}
	}

	// Is this an absolute file path?
	filePath := URLPathToFilePath(urlOrPath)
	if orFile && filepath.IsAbs(filePath) {
		return self.NewValidFileURL(filePath)
	}

	// Treat as relative path
	for _, origin := range origins {
		var url URL
		var err error

		switch origin_ := origin.(type) {
		case *FileURL:
			url, err = origin_.NewValidRelativeFileURL(filePath)

		case *NetworkURL:
			url, err = origin_.NewValidRelativeNetworkURL(urlOrPath)

		case *TarballURL:
			url, err = origin_.NewValidRelativeTarballURL(context, urlOrPath)

		case *ZipURL:
			url, err = origin_.NewValidRelativeZipURL(context, urlOrPath)

		case *GitURL:
			url, err = origin_.NewValidRelativeGitURL(urlOrPath)

		case *InternalURL:
			url, err = origin_.NewValidRelativeInternalURL(urlOrPath)
		}

		if err == nil {
			return url, nil
		}
	}

	return nil, fmt.Errorf("invalid URL: %s", urlOrPath)
}
