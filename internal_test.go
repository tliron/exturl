package exturl

import (
	"bytes"
	"testing"
)

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
