package exturl

import (
	"bytes"
	contextpkg "context"
	"io"
)

func testRead(context *Context, url string) ([]byte, error) {
	url_, err := context.NewURL(url)
	if err != nil {
		return nil, err
	}
	reader, err := url_.Open(contextpkg.TODO())
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

type testProvider struct {
	content []byte
}

// ([InternalURLProvider] interface)
func (self *testProvider) OpenPath(context contextpkg.Context, path string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(self.content)), nil
}
