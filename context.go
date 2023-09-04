package exturl

import (
	contextpkg "context"
	"net/http"
	"os"
	"sync"

	"github.com/tliron/kutil/util"
)

type Credentials struct {
	Username string
	Password string
	Token    string
}

//
// Context
//

type URLTransformerFunc func(fromUrl string) (string, bool)

type Context struct {
	transformers      []URLTransformerFunc
	mappings          map[string]string
	files             map[string]string
	dirs              map[string]string
	httpRoundTrippers map[string]http.RoundTripper
	credentials       map[string]*Credentials
	lock              sync.Mutex // for files
}

func NewContext() *Context {
	var self Context
	self.transformers = []URLTransformerFunc{self.GetMapping}
	return &self
}

func (self *Context) Transform(fromUrl string) (string, bool) {
	for _, transformer := range self.transformers {
		if toUrl, ok := transformer(fromUrl); ok {
			return toUrl, true
		}
	}
	return "", false
}

func (self *Context) AddTransformer(transformer URLTransformerFunc) {
	self.transformers = append(self.transformers, transformer)
}

// Set toUrl to empty string to delete the mapping.
func (self *Context) Map(fromUrl string, toUrl string) {
	if self.mappings == nil {
		self.mappings = make(map[string]string)
	}

	if toUrl == "" {
		delete(self.mappings, fromUrl)
	} else {
		self.mappings[fromUrl] = toUrl
	}
}

// URLTransformerFunc signature
func (self *Context) GetMapping(fromUrl string) (string, bool) {
	if self.mappings == nil {
		return "", false
	}
	toUrl, ok := self.mappings[fromUrl]
	return toUrl, ok
}

// Not thread-safe
func (self *Context) SetHTTPRoundTripper(host string, httpRoundTripper http.RoundTripper) {
	if self.httpRoundTrippers == nil {
		self.httpRoundTrippers = make(map[string]http.RoundTripper)
	}

	self.httpRoundTrippers[host] = httpRoundTripper
}

// Not thread-safe
func (self *Context) GetHTTPRoundTripper(host string) http.RoundTripper {
	if self.httpRoundTrippers != nil {
		httpRoundTripper, _ := self.httpRoundTrippers[host]
		return httpRoundTripper
	} else {
		return nil
	}
}

// Not thread-safe
func (self *Context) SetCredentials(host string, username string, password string, token string) {
	if self.credentials == nil {
		self.credentials = make(map[string]*Credentials)
	}

	self.credentials[host] = &Credentials{
		Username: username,
		Password: password,
		Token:    token,
	}
}

// Not thread-safe
func (self *Context) GetCredentials(host string) *Credentials {
	if self.credentials != nil {
		credentials, _ := self.credentials[host]
		return credentials
	} else {
		return nil
	}
}

func (self *Context) OpenFile(context contextpkg.Context, url URL) (*os.File, error) {
	if path, err := self.GetLocalPath(context, url); err == nil {
		return os.Open(path)
	} else {
		return nil, err
	}
}

// Will download the file to the local temporary directory if not already locally available
func (self *Context) GetLocalPath(context contextpkg.Context, url URL) (string, error) {
	if fileUrl, ok := url.(*FileURL); ok {
		// No need to download file URLs
		return fileUrl.Path, nil
	}

	key := url.Key()

	self.lock.Lock()
	defer self.lock.Unlock()

	if self.files != nil {
		if path, ok := self.files[key]; ok {
			if ok, err := util.DoesFileExist(path); err == nil {
				if ok {
					return path, nil
				} else {
					delete(self.files, key)
				}
			} else {
				return "", err
			}
		}
	}

	if file, err := Download(context, url, GetTemporaryPathPattern(key)); err == nil {
		if self.files == nil {
			self.files = make(map[string]string)
		}
		path := file.Name()
		self.files[key] = path
		return path, nil
	} else {
		return "", err
	}
}

func (self *Context) Release() error {
	self.lock.Lock()
	defer self.lock.Unlock()

	var err error

	if self.files != nil {
		for _, path := range self.files {
			if err_ := DeleteTemporaryFile(path); err_ != nil {
				err = err_
			}
		}

		self.files = nil
	}

	if self.dirs != nil {
		for _, path := range self.dirs {
			if err_ := DeleteTemporaryDir(path); err_ != nil {
				err = err_
			}
		}

		self.files = nil
	}

	return err
}
