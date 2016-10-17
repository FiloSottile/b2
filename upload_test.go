package b2_test

import (
	"bytes"
	"crypto/rand"
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
