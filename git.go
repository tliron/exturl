package exturl

import (
	contextpkg "context"
	"fmt"
	"io"
	neturlpkg "net/url"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

//
// GitURL
//

type GitURL struct {
	Path          string
	RepositoryURL string
	Reference     string
	Username      string
	Password      string

	clonePath  string
	urlContext *Context
}

func (self *Context) NewGitURL(path string, repositoryUrl string) *GitURL {
	// Must be absolute
	path = strings.TrimLeft(path, "/")

	var gitUrl = GitURL{
		Path:       path,
		urlContext: self,
	}

	if neturl, err := neturlpkg.Parse(repositoryUrl); err == nil {
		if neturl.User != nil {
			gitUrl.Username = neturl.User.Username()
			if password, ok := neturl.User.Password(); ok {
				gitUrl.Password = password
			}
			// Don't store user info
			neturl.User = nil
		}
		gitUrl.Reference = neturl.Fragment
		neturl.Fragment = ""
		gitUrl.RepositoryURL = neturl.String()
	} else {
		gitUrl.RepositoryURL = repositoryUrl
	}

	return &gitUrl
}

func (self *Context) NewValidGitURL(path string, repositoryUrl string) (*GitURL, error) {
	gitUrl := self.NewGitURL(path, repositoryUrl)
	if _, err := gitUrl.OpenRepository(); err == nil {
		path := filepath.Join(gitUrl.clonePath, gitUrl.Path)
		if _, err := os.Stat(path); err == nil {
			return gitUrl, nil
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}

func (self *GitURL) NewValidRelativeGitURL(path string) (*GitURL, error) {
	gitUrl := self.Relative(path).(*GitURL)
	if _, err := gitUrl.OpenRepository(); err == nil {
		path_ := filepath.Join(gitUrl.clonePath, gitUrl.Path)
		if _, err := os.Stat(path_); err == nil {
			return gitUrl, nil
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}

func (self *Context) ParseGitURL(url string) (*GitURL, error) {
	if repositoryUrl, path, err := parseGitURL(url); err == nil {
		return self.NewGitURL(path, repositoryUrl), nil
	} else {
		return nil, err
	}
}

func (self *Context) ParseValidGitURL(url string) (*GitURL, error) {
	if repositoryUrl, path, err := parseGitURL(url); err == nil {
		return self.NewValidGitURL(path, repositoryUrl)
	} else {
		return nil, err
	}
}

// URL interface
// fmt.Stringer interface
func (self *GitURL) String() string {
	return self.Key()
}

// URL interface
func (self *GitURL) Format() string {
	return GetFormat(self.Path)
}

// URL interface
func (self *GitURL) Origin() URL {
	path := pathpkg.Dir(self.Path)
	if path != "/" {
		path += "/"
	}

	return &GitURL{
		Path:          path,
		RepositoryURL: self.RepositoryURL,
		clonePath:     self.clonePath,
		urlContext:    self.urlContext,
	}
}

// URL interface
func (self *GitURL) Relative(path string) URL {
	return &GitURL{
		Path:          pathpkg.Join(self.Path, path),
		RepositoryURL: self.RepositoryURL,
		clonePath:     self.clonePath,
		urlContext:    self.urlContext,
	}
}

// URL interface
func (self *GitURL) Key() string {
	return fmt.Sprintf("git:%s!/%s", self.RepositoryURL, self.Path)
}

// URL interface
func (self *GitURL) Open(context contextpkg.Context) (io.ReadCloser, error) {
	if _, err := self.OpenRepository(); err == nil {
		path := filepath.Join(self.clonePath, self.Path)
		if reader, err := os.Open(path); err == nil {
			return reader, nil
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}

// URL interface
func (self *GitURL) Context() *Context {
	return self.urlContext
}

func (self *GitURL) OpenRepository() (*git.Repository, error) {
	if self.clonePath != "" {
		return self.openRepository(false)
	} else {
		key := self.Key()

		// Note: this will lock for the entire clone duration!
		self.urlContext.lock.Lock()
		defer self.urlContext.lock.Unlock()

		if self.urlContext.dirs != nil {
			// Already cloned?
			if clonePath, ok := self.urlContext.dirs[key]; ok {
				self.clonePath = clonePath
				return self.openRepository(false)
			}
		}

		if clonePath, err := os.MkdirTemp("", GetTemporaryPathPattern(key)); err == nil {
			if self.urlContext.dirs == nil {
				self.urlContext.dirs = make(map[string]string)
			}
			self.urlContext.dirs[key] = clonePath

			// Clone
			fmt.Println(self.RepositoryURL)
			if repository, err := git.PlainClone(clonePath, false, &git.CloneOptions{
				URL:   self.RepositoryURL,
				Auth:  self.getAuth(),
				Depth: 1,
				Tags:  git.NoTags,
			}); err == nil {
				if reference, err := self.findReference(repository); err == nil {
					if reference != nil {
						// Checkout
						if workTree, err := repository.Worktree(); err == nil {
							if err := workTree.Checkout(&git.CheckoutOptions{
								Branch: reference.Name(),
							}); err != nil {
								os.RemoveAll(clonePath)
								return nil, err
							}
						} else {
							os.RemoveAll(clonePath)
							return nil, err
						}
					}
				} else {
					os.RemoveAll(clonePath)
					return nil, err
				}

				self.clonePath = clonePath
				return repository, nil
			} else {
				os.RemoveAll(clonePath)
				return nil, err
			}
		} else {
			return nil, err
		}
	}
}

func (self *GitURL) openRepository(pull bool) (*git.Repository, error) {
	if repository, err := git.PlainOpen(self.clonePath); err == nil {
		if pull {
			if err := self.pullRepository(repository); err != nil {
				return nil, err
			}
		}

		return repository, nil
	} else {
		return nil, err
	}
}

func (self *GitURL) pullRepository(repository *git.Repository) error {
	if workTree, err := repository.Worktree(); err == nil {
		if err := workTree.Pull(&git.PullOptions{
			Auth: self.getAuth(),
		}); (err == nil) || (err == git.NoErrAlreadyUpToDate) {
			return nil
		} else {
			return err
		}
	} else {
		return err
	}
}

func (self *GitURL) findReference(repository *git.Repository) (*plumbing.Reference, error) {
	if self.Reference != "" {
		if iter, err := repository.References(); err == nil {
			defer iter.Close()
			for {
				if reference, err := iter.Next(); err == nil {
					name := reference.Name()
					if name.Short() == self.Reference {
						return reference, nil
					} else if name.String() == self.Reference {
						return reference, nil
					}
				} else if err == io.EOF {
					return nil, NewNotFoundf("reference %q not found in git repository: %s", self.Reference, self.RepositoryURL)
				} else {
					return nil, err
				}
			}
		} else {
			return nil, err
		}
	} else {
		return nil, nil
	}
}

func (self *GitURL) getAuth() transport.AuthMethod {
	// TODO: what about non-HTTP transports, like ssh?
	if self.Username != "" {
		return &http.BasicAuth{
			Username: self.Username,
			Password: self.Password,
		}
	} else {
		return nil
	}
}

func parseGitURL(url string) (string, string, error) {
	if strings.HasPrefix(url, "git:") {
		if split := strings.Split(url[4:], "!"); len(split) == 2 {
			return split[0], split[1], nil
		} else {
			return "", "", fmt.Errorf("malformed \"git:\" URL: %s", url)
		}
	} else {
		return "", "", fmt.Errorf("not a \"git:\" URL: %s", url)
	}
}
