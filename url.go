package exturl

import (
	contextpkg "context"
	"fmt"
	"io"
	neturlpkg "net/url"
	"path/filepath"
)

//
// URL
//

type URL interface {
	// Returns a string representation of the URL that can be used by
	// Context.NewURL to recreate this URL.
	//
	// (fmt.Stringer interface)
	String() string

	// Format of the URL content's default representation.
	//
	// Can return "yaml", "json", "xml", etc., or an empty string if the format
	// is unknown.
	//
	// The format is often derived from a file extension if available, otherwise
	// it might be retrieved from metadata.
	//
	// An attempt is made to standardize the return values, e.g. a "yml" file
	// extension is always returned as "yaml", and a "tar.gz" file extension is
	// always returned as "tgz".
	Format() string

	// Returns a URL that is the equivalent of a "base directory" for this URL.
	//
	// Base URLs always often have a trailing slash to signify that they are
	// "directories" rather than "files". One notable exception is "file:" URLs
	// when compiled on Windows, in which case a trailing backslash is used
	// instead.
	//
	// The base is often used in two ways:
	//
	// 1. You can call URL.Relative on it to get a sibling URL to this one (relative
	//    to the same "base directory").
	// 2. You can use it in the "bases" list argument of Context.NewValidURL for the
	//    same purpose.
	//
	// Note that the base might not be a valid URL in itself, e.g. you might not
	// be able to call Open on it.
	Base() URL

	// Parses the argument as a path relative to the URL. That means that this
	// URL is treated as a "base directory" (see URL.Base). The argument supports
	// ".." and ".", with the returned URL path always being absolute.
	Relative(path string) URL

	// As URL.Relative but returns a valid URL.
	ValidRelative(context contextpkg.Context, path string) (URL, error)

	// Returns a string that uniquely identifies the URL.
	//
	// Useful for map and cache keys.
	Key() string

	// Opens the URL for reading.
	//
	// Note that for some URLs it can involve lengthy operations, e.g. cloning a
	// remote repository or downloading an archive. For this reason a cancellable
	// context can be provided as an argument.
	//
	// An effort is made to not repeat these lengthy operations by caching related
	// state in the URL's exturl Context (caching is deliberately not done globally).
	// For example, when accessing a "git:" URL on a remote git repository then that
	// repository will be cloned locally only if it's the first the repository has been
	// referred to for the exturl Context. Subsequent Open calls for URLs that refer
	// to the same git repository will reuse the existing clone.
	//
	// It is the caller's responsibility to call Close on the reader.
	Open(context contextpkg.Context) (io.ReadCloser, error)

	// The exturl context used to create this URL.
	Context() *Context
}

// Parses the argument as an absolute URL.
//
// To support relative URLs, see [Context.NewValidURL].
//
// If you are expecting either a URL or a file path, consider [Context.NewAnyOrFileURL].
func (self *Context) NewURL(url string) (URL, error) {
	if mappedUrl, ok := self.GetMapping(url); ok {
		url = mappedUrl
	}

	if neturl, err := neturlpkg.ParseRequestURI(url); err == nil {
		switch neturl.Scheme {
		case "http", "https":
			// Go's "net/http" only handles "http:" and "https:"
			return self.NewNetworkURL(neturl), nil

		case "file":
			filePath := URLPathToFilePath(neturl.Path)
			return self.NewFileURL(filePath), nil

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

// Parses the argument as either an absolute URL or a file path.
//
// In essence attempts to parse the URL via [Context.NewURL] and if that fails
// treats the URL as a file path and returns a [*FileURL].
//
// To support relative URLs, see [Context.NewValidAnyOrFileURL].
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

// Parses the argument as either an absolute URL or a relative path.
// Relative paths support ".." and ".", with the returned URL path always
// being absolute.
//
// The returned URL is "valid", meaning that during this call it was
// possible to call Open on it. Of course this can't guarantee that
// future calls to Open will succeed.
//
// Relative URLs are tested against the "bases" argument in order. The
// first valid URL will be returned and the remaining bases will be
// ignored. Note that bases can be any of any URL type.
//
// If you are expecting either a URL or a file path, consider
// [Context.NewValidAnyOrFileURL].
func (self *Context) NewValidURL(context contextpkg.Context, urlOrPath string, bases []URL) (URL, error) {
	return self.newValidUrl(context, urlOrPath, bases, false)
}

// Parses the argument as an absolute URL or an absolute file path
// or a relative path. Relative paths support ".." and ".", with the
// returned URL path always being absolute.
//
// The returned URL is "valid", meaning that during this call it was
// possible to call Open on it. Of course this can't guarantee that
// future calls to Open will succeed.
//
// Relative URLs are tested against the "bases" argument in order. The
// first valid URL will be returned and the remaining bases will be
// ignored. Note that bases can be any of any URL type.
func (self *Context) NewValidAnyOrFileURL(context contextpkg.Context, urlOrPath string, bases []URL) (URL, error) {
	return self.newValidUrl(context, urlOrPath, bases, true)
}

func (self *Context) newValidUrl(context contextpkg.Context, urlOrPath string, bases []URL, orFile bool) (URL, error) {
	if mappedUrl, ok := self.GetMapping(urlOrPath); ok {
		urlOrPath = mappedUrl
	}

	if neturl, err := neturlpkg.ParseRequestURI(urlOrPath); err == nil {
		switch neturl.Scheme {
		case "http", "https":
			// Go's "net/http" only handles "http:" and "https:"
			return self.NewValidNetworkURL(neturl)

		case "file":
			filePath := URLPathToFilePath(neturl.Path)
			return self.NewValidFileURL(filePath)

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
	for _, base := range bases {
		var url URL
		var err error

		switch base_ := base.(type) {
		case *FileURL:
			url, err = base_.ValidRelative(context, filePath)

		default:
			url, err = base_.ValidRelative(context, urlOrPath)
		}

		if err == nil {
			return url, nil
		}
	}

	return nil, fmt.Errorf("invalid URL: %s", urlOrPath)
}
