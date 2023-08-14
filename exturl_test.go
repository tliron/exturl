package exturl

import (
	"bytes"
	contextpkg "context"
	"io"
	"testing"
)

func TestFile(t *testing.T) {
	context := NewContext()
	defer context.Release()

	url, _ := context.NewURL("/abs/path")
	if url.String() != "file:///abs/path" {
		t.Error("absolute file path")
		return
	}

	fileUrl := context.NewFileURL("rel/path")
	if fileUrl.String() != "rel/path" {
		t.Error("relative file path")
		return
	}
}

func TestInternal(t *testing.T) {
	context := NewContext()
	defer context.Release()

	RegisterInternalURL("bytes", []byte{1, 2, 3, 4, 5})
	b, _ := testRead(context, "internal:bytes")
	if !bytes.Equal(b, []byte{1, 2, 3, 4, 5}) {
		t.Error("internal bytes")
		return
	}

	RegisterInternalURL("string", "a string")
	b, _ = testRead(context, "internal:string")
	if string(b) != "a string" {
		t.Error("internal string")
		return
	}

	RegisterInternalURL("int", 12345)
	b, _ = testRead(context, "internal:int")
	if string(b) != "12345" {
		t.Error("internal int")
		return
	}

	RegisterInternalURL("provider", &testProvider{[]byte{6, 7, 8, 9, 10}})
	b, _ = testRead(context, "internal:provider")
	if !bytes.Equal(b, []byte{6, 7, 8, 9, 10}) {
		t.Error("internal provider")
		return
	}
}

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

func (self *testProvider) OpenPath(context contextpkg.Context, path string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(self.content)), nil
}
