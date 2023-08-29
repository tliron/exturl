package exturl

import (
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

	if _, err := context.NewWorkingDirFileURL(); err != nil {
		t.Errorf("NewWorkingDirFileURL: %s", err.Error())
		return
	}
}
