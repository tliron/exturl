package exturl

import (
	"testing"
)

func TestFile(t *testing.T) {
	context := NewContext()
	defer context.Release()

	var absoluteFilePath, absoluteUrlString string
	var relativeFilePath, relativeUrlString string

	relativeFilePath = "../rel/path"
	if PathSeparator == `\` {
		// Windows
		absoluteFilePath = `C:\abs\path`
		absoluteUrlString = "file:///C:/abs/path"
		relativeUrlString = "file:///C:/abs/rel/path"
	} else {
		absoluteFilePath = "/abs/path"
		absoluteUrlString = "file:///abs/path"
		relativeUrlString = "file:///abs/rel/path"
	}

	absoluteUrl, _ := context.NewURL(absoluteUrlString)
	relativeFileUrl := context.NewFileURL(relativeFilePath)

	if url_ := context.NewAnyOrFileURL(absoluteFilePath).String(); url_ != absoluteUrlString {
		t.Errorf("absolute file path: %s", url_)
		return
	}

	if url_ := absoluteUrl.(*FileURL).Path; url_ != absoluteFilePath {
		t.Errorf("absolute URL: %s", url_)
		return
	}

	if url_ := relativeFileUrl.String(); url_ != relativeFilePath {
		t.Errorf("relative file path: %s", url_)
		return
	}

	if url_ := absoluteUrl.Relative(relativeFilePath).String(); url_ != relativeUrlString {
		t.Errorf("file path relative to base: %s", url_)
		return
	}

	if _, err := context.NewWorkingDirFileURL(); err != nil {
		t.Errorf("NewWorkingDirFileURL: %s", err.Error())
		return
	}
}
