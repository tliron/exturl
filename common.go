package exturl

import (
	contextpkg "context"
	"fmt"
	"io"
	neturlpkg "net/url"
	"os"

	"github.com/tliron/commonlog"
	"github.com/tliron/kutil/util"
)

var log = commonlog.GetLogger("exturl")

func ToNetURL(url URL) (*neturlpkg.URL, error) {
	return neturlpkg.ParseRequestURI(url.String())
}

func GetPath(url URL) (string, error) {
	if url_, err := ToNetURL(url); err == nil {
		if url_.Path != "" {
			return neturlpkg.PathUnescape(url_.Path)
		} else {
			return neturlpkg.PathUnescape(url_.Opaque)
		}
	} else {
		return "", err
	}
}

func ReadBytes(context contextpkg.Context, url URL) ([]byte, error) {
	if reader, err := url.Open(context); err == nil {
		reader = util.NewContextualReadCloser(context, reader)
		defer reader.Close()
		return io.ReadAll(reader)
	} else {
		return nil, err
	}
}

func ReadString(context contextpkg.Context, url URL) (string, error) {
	if bytes, err := ReadBytes(context, url); err == nil {
		return util.BytesToString(bytes), nil
	} else {
		return "", err
	}
}

func Size(context contextpkg.Context, url URL) (int64, error) {
	if reader, err := url.Open(context); err == nil {
		reader = util.NewContextualReadCloser(context, reader)
		defer reader.Close()
		return util.ReaderSize(reader)
	} else {
		return 0, err
	}
}

func DownloadTo(context contextpkg.Context, url URL, path string) error {
	if writer, err := os.Create(path); err == nil {
		if reader, err := url.Open(context); err == nil {
			reader = util.NewContextualReadCloser(context, reader)
			defer reader.Close()
			log.Infof("downloading from %q to file %q", url.String(), path)
			if _, err = io.Copy(writer, reader); err == nil {
				return nil
			} else {
				log.Warningf("failed to download from %q", url.String())
				return err
			}
		} else {
			return err
		}
	} else {
		return err
	}
}

func Download(context contextpkg.Context, url URL, temporaryPathPattern string) (*os.File, error) {
	if file, err := os.CreateTemp("", temporaryPathPattern); err == nil {
		path := file.Name()
		if reader, err := url.Open(context); err == nil {
			reader = util.NewContextualReadCloser(context, reader)
			defer reader.Close()
			log.Infof("downloading from %q to temporary file %q", url.String(), path)
			if _, err = io.Copy(file, reader); err == nil {
				util.OnExitError(func() error {
					return DeleteTemporaryFile(path)
				})
				return file, nil
			} else {
				log.Warningf("failed to download from %q", url.String())
				DeleteTemporaryFile(path)
				return nil, err
			}
		} else {
			DeleteTemporaryFile(path)
			return nil, err
		}
	} else {
		return nil, err
	}
}

func GetTemporaryPathPattern(key string) string {
	return fmt.Sprintf("exturl-%s-*", util.SanitizeFilename(key))
}
