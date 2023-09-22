package exturl

import (
	contextpkg "context"
	"fmt"
	"io"
	pathpkg "path"
	"strings"
)

// Note: we must use the "path" package rather than "filepath" to ensure consistency with Windows

//
// ZipURL
//
// Inspired by Java's JarURLConnection:
// https://docs.oracle.com/javase/8/docs/api/java/net/JarURLConnection.html
//

type ZipURL struct {
	Path       string
	ArchiveURL URL
}

func NewZipURL(path string, archiveUrl URL) *ZipURL {
	path = strings.TrimLeft(path, "/")

	return &ZipURL{
		Path:       path,
		ArchiveURL: archiveUrl,
	}
}

func NewValidZipURL(context contextpkg.Context, path string, archiveUrl URL) (*ZipURL, error) {
	self := NewZipURL(path, archiveUrl)
	if zipReader, err := self.OpenArchive(context); err == nil {
		defer zipReader.Close()

		for _, file := range zipReader.ZipReader.File {
			if self.Path == file.Name {
				return self, nil
			}
		}

		return nil, NewNotFoundf("path %q not found in zip: %s", path, archiveUrl.String())
	} else {
		return nil, err
	}
}

func (self *Context) ParseZipURL(url string) (*ZipURL, error) {
	if archiveUrl, path, err := parseZipURL(url); err == nil {
		archiveUrl_ := self.NewAnyOrFileURL(archiveUrl)
		return NewZipURL(path, archiveUrl_), nil
	} else {
		return nil, err
	}
}

func (self *Context) ParseValidZipURL(context contextpkg.Context, url string) (*ZipURL, error) {
	if archiveUrl, path, err := parseZipURL(url); err == nil {
		archiveUrl_ := self.NewAnyOrFileURL(archiveUrl)
		return NewValidZipURL(context, path, archiveUrl_)
	} else {
		return nil, err
	}
}

// ([fmt.Stringer] interface)
func (self *ZipURL) String() string {
	return self.Key()
}

// ([URL] interface)
func (self *ZipURL) Format() string {
	return GetFormat(self.Path)
}

// ([URL] interface)
func (self *ZipURL) Base() URL {
	path := pathpkg.Dir(self.Path)
	if path != "/" {
		path += "/"
	}

	return &ZipURL{
		Path:       path,
		ArchiveURL: self.ArchiveURL,
	}
}

// ([URL] interface)
func (self *ZipURL) Relative(path string) URL {
	return &ZipURL{
		Path:       pathpkg.Join(self.Path, path),
		ArchiveURL: self.ArchiveURL,
	}
}

// ([URL] interface)
func (self *ZipURL) ValidRelative(context contextpkg.Context, path string) (URL, error) {
	zipUrl := self.Relative(path).(*ZipURL)
	if zipReader, err := zipUrl.OpenArchive(context); err == nil {
		defer zipReader.Close()

		for _, file := range zipReader.ZipReader.File {
			if zipUrl.Path == file.Name {
				return zipUrl, nil
			}
		}

		return nil, NewNotFoundf("path %q not found in zip: %s", zipUrl.Path, zipUrl.ArchiveURL.String())
	} else {
		return nil, err
	}
}

// ([URL] interface)
func (self *ZipURL) Key() string {
	return fmt.Sprintf("zip:%s!/%s", self.ArchiveURL.String(), self.Path)
}

// ([URL] interface)
func (self *ZipURL) Open(context contextpkg.Context) (io.ReadCloser, error) {
	if zipReader, err := self.OpenArchive(context); err == nil {
		if zipEntryReader, err := zipReader.Open(self.Path); err == nil {
			if zipEntryReader != nil {
				return zipEntryReader, nil
			} else {
				zipReader.Close()
				return nil, NewNotFoundf("path %q not found in archive: %s", self.Path, self.ArchiveURL.String())
			}
		} else {
			zipReader.Close()
			return nil, err
		}
	} else {
		return nil, err
	}
}

// ([URL] interface)
func (self *ZipURL) Context() *Context {
	return self.ArchiveURL.Context()
}

func (self *ZipURL) OpenArchive(context contextpkg.Context) (*ZipReader, error) {
	if file, err := self.ArchiveURL.Context().OpenFile(context, self.ArchiveURL); err == nil {
		return NewZipReaderForFile(file)
	} else {
		return nil, err
	}
}

// Utils

func parseZipURL(url string) (string, string, error) {
	if strings.HasPrefix(url, "zip:") {
		if split := strings.Split(url[4:], "!"); len(split) == 2 {
			return split[0], split[1], nil
		} else {
			return "", "", fmt.Errorf("malformed \"zip:\" URL: %s", url)
		}
	} else {
		return "", "", fmt.Errorf("not a \"zip:\" URL: %s", url)
	}
}
