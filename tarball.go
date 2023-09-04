package exturl

import (
	"archive/tar"
	contextpkg "context"
	"fmt"
	"io"
	pathpkg "path"
	"strings"

	"github.com/klauspost/pgzip"
)

// Note: we must use the "path" package rather than "filepath" to ensure consistency with Windows

// TODO: xz support, consider: https://github.com/ulikunitz/xz

var TARBALL_ARCHIVE_FORMATS = []string{"tar", "tar.gz"}

func IsValidTarballArchiveFormat(archiveFormat string) bool {
	for _, archiveFormat_ := range TARBALL_ARCHIVE_FORMATS {
		if archiveFormat_ == archiveFormat {
			return true
		}
	}
	return false
}

//
// TarballURL
//
// Inspired by Java's JarURLConnection:
// https://docs.oracle.com/javase/8/docs/api/java/net/JarURLConnection.html
//

type TarballURL struct {
	Path          string
	ArchiveURL    URL
	ArchiveFormat string
}

func NewTarballURL(path string, archiveUrl URL, archiveFormat string) *TarballURL {
	path = strings.TrimLeft(path, "/")

	if archiveFormat == "" {
		archiveFormat = archiveUrl.Format()
	}

	return &TarballURL{
		Path:          path,
		ArchiveURL:    archiveUrl,
		ArchiveFormat: archiveFormat,
	}
}

func NewValidTarballURL(context contextpkg.Context, path string, archiveUrl URL, archiveFormat string) (*TarballURL, error) {
	self := NewTarballURL(path, archiveUrl, archiveFormat)
	if tarballReader, err := self.OpenArchive(context); err == nil {
		defer tarballReader.Close()

		for {
			if header, err := tarballReader.TarReader.Next(); err == nil {
				if self.Path == fixTarballEntryPath(header.Name) {
					return self, nil
				}
			} else if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}

		return nil, NewNotFoundf("path %q not found in tarball: %s", path, archiveUrl.String())
	} else {
		return nil, err
	}
}

func (self *Context) ParseTarballURL(url string) (*TarballURL, error) {
	if archiveUrl, path, err := parseTarballURL(url); err == nil {
		archiveUrl_ := self.NewAnyOrFileURL(archiveUrl)
		return NewTarballURL(path, archiveUrl_, ""), nil
	} else {
		return nil, err
	}
}

func (self *Context) ParseValidTarballURL(context contextpkg.Context, url string) (*TarballURL, error) {
	if archiveUrl, path, err := parseTarballURL(url); err == nil {
		archiveUrl_ := self.NewAnyOrFileURL(archiveUrl)
		return NewValidTarballURL(context, path, archiveUrl_, "")
	} else {
		return nil, err
	}
}

// ([URL] interface, [fmt.Stringer] interface)
func (self *TarballURL) String() string {
	return self.Key()
}

// ([URL] interface)
func (self *TarballURL) Format() string {
	return GetFormat(self.Path)
}

// ([URL] interface)
func (self *TarballURL) Base() URL {
	path := pathpkg.Dir(self.Path)
	if path != "/" {
		path += "/"
	}

	return &TarballURL{
		Path:          path,
		ArchiveURL:    self.ArchiveURL,
		ArchiveFormat: self.ArchiveFormat,
	}
}

// ([URL] interface)
func (self *TarballURL) Relative(path string) URL {
	return &TarballURL{
		Path:          pathpkg.Join(self.Path, path),
		ArchiveURL:    self.ArchiveURL,
		ArchiveFormat: self.ArchiveFormat,
	}
}

// ([URL] interface)
func (self *TarballURL) ValidRelative(context contextpkg.Context, path string) (URL, error) {
	tarballUrl := self.Relative(path).(*TarballURL)
	if tarballReader, err := tarballUrl.OpenArchive(context); err == nil {
		defer tarballReader.Close()

		for {
			if header, err := tarballReader.TarReader.Next(); err == nil {
				if tarballUrl.Path == fixTarballEntryPath(header.Name) {
					return tarballUrl, nil
				}
			} else if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}

		return nil, NewNotFoundf("path %q not found in tarball: %s", tarballUrl.Path, tarballUrl.ArchiveURL.String())
	} else {
		return nil, err
	}
}

// ([URL] interface)
func (self *TarballURL) Key() string {
	return fmt.Sprintf("tar:%s!/%s", self.ArchiveURL.String(), self.Path)
}

// ([URL] interface)
func (self *TarballURL) Open(context contextpkg.Context) (io.ReadCloser, error) {
	if tarballReader, err := self.OpenArchive(context); err == nil {
		if tarballEntryReader, err := tarballReader.Open(self.Path); err == nil {
			if tarballEntryReader != nil {
				return tarballEntryReader, nil
			} else {
				tarballReader.Close()
				return nil, NewNotFoundf("path %q not found in archive: %s", self.Path, self.ArchiveURL.String())
			}
		} else {
			tarballReader.Close()
			return nil, err
		}
	} else {
		return nil, err
	}
}

// ([URL] interface)
func (self *TarballURL) Context() *Context {
	return self.ArchiveURL.Context()
}

func (self *TarballURL) OpenArchive(context contextpkg.Context) (*TarballReader, error) {
	if !IsValidTarballArchiveFormat(self.ArchiveFormat) {
		return nil, fmt.Errorf("unsupported tarball archive format: %q", self.ArchiveFormat)
	}

	if archiveReader, err := self.ArchiveURL.Open(context); err == nil {
		switch self.ArchiveFormat {
		case "tar":
			return NewTarballReader(tar.NewReader(archiveReader), archiveReader, nil), nil

		case "tar.gz":
			if gzipReader, err := pgzip.NewReader(archiveReader); err == nil {
				return NewTarballReader(tar.NewReader(gzipReader), archiveReader, gzipReader), nil
			} else {
				archiveReader.Close()
				return nil, err
			}

		default:
			return nil, fmt.Errorf("unsupported tarball format: %s", self.ArchiveFormat)
		}
	} else {
		return nil, err
	}
}

// Utils

func parseTarballURL(url string) (string, string, error) {
	if strings.HasPrefix(url, "tar:") {
		if split := strings.Split(url[4:], "!"); len(split) == 2 {
			return split[0], split[1], nil
		} else {
			return "", "", fmt.Errorf("malformed \"tar:\" URL: %s", url)
		}
	} else {
		return "", "", fmt.Errorf("not a \"tar:\" URL: %s", url)
	}
}

func fixTarballEntryPath(path string) string {
	if strings.HasPrefix(path, "./") {
		return path[3:]
	}
	return path
}
