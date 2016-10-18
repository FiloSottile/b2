package b2_test

import (
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"os"
	"testing"
)

func TestUploadError(t *testing.T) {
	c := getClient(t)
	b := getBucket(t, c)
	defer b.Delete()

	file := make([]byte, 123456)
	rand.Read(file)
	_, err := b.Upload(bytes.NewReader(file), "illegal//filename", "")
	if err == nil {
		t.Fatal("Expected an error")
	}
	t.Log(err)
}

func TestUploadFile(t *testing.T) {
	c := getClient(t)
	b := getBucket(t, c)
	defer b.Delete()

	tmpfile, err := ioutil.TempFile("", "b2")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := make([]byte, 123456)
	rand.Read(content)
	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	fi, err := b.Upload(f, "foo-file", "")
	if err != nil {
		t.Fatal(err)
	}
	defer c.DeleteFile(fi.ID, fi.Name)
	if fi.ContentLength != 123456 {
		t.Error("mismatched fi.ContentLength", fi.ContentLength)
	}
}
