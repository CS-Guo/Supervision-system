package storage

import (
	"bytes"
	"io"
	"mime/multipart"
	"os"
	"testing"
)

type testFile struct{ *bytes.Reader }

func (t testFile) Close() error { return nil }

var _ multipart.File = testFile{}

func TestSaveAndOpen(t *testing.T) {
	l := Local{Root: t.TempDir()}
	key, n, e := l.Save(testFile{bytes.NewReader([]byte("material"))})
	if e != nil {
		t.Fatal(e)
	}
	if n != 8 {
		t.Fatalf("size=%d", n)
	}
	f, e := l.Open(key)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	b, _ := io.ReadAll(f)
	if string(b) != "material" {
		t.Fatalf("content=%q", b)
	}
	l.Delete(key)
	if _, e = os.Stat(l.Root + "/" + key); !os.IsNotExist(e) {
		t.Fatal("file was not deleted")
	}
}
