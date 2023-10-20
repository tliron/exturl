package exturl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tliron/commonlog"
	"github.com/tliron/kutil/util"
)

var log = commonlog.GetLogger("exturl")

func GetFormat(path string) string {
	extension := filepath.Ext(path)
	if extension == "" {
		return ""
	}

	extension = strings.ToLower(extension[1:])

	// Special handling for tarballs
	if pre4start := len(path) - len(extension) - 5; pre4start > 0 {
		pre4 := path[pre4start : pre4start+4]
		if pre4 == ".tar" {
			return "tar." + extension
		}
	}

	// Special handling for alternative extensions
	switch extension {
	case "yml":
		return "yaml"

	case "tgz":
		return "tar.gz"
	}

	return extension
}

func GetTemporaryPathPattern(key string) string {
	return fmt.Sprintf("exturl-%s-*", util.SanitizeFilename(key))
}

func DeleteTemporaryFile(path string) error {
	if err := os.Remove(path); err == nil {
		log.Infof("deleted temporary file %q", path)
		return nil
	} else if os.IsNotExist(err) {
		log.Infof("temporary file already deleted %q", path)
		return nil
	} else {
		log.Errorf("could not delete temporary file %q: %s", path, err.Error())
		return err
	}
}

func DeleteTemporaryDir(path string) error {
	if err := os.RemoveAll(path); err == nil {
		log.Infof("deleted temporary dir %q", path)
		return nil
	} else if os.IsNotExist(err) {
		log.Infof("temporary dir already deleted %q", path)
		return nil
	} else {
		log.Errorf("could not delete temporary dir %q: %s", path, err.Error())
		return err
	}
}
