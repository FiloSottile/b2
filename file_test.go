package b2_test

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"testing"
)

func TestFileLifecycle(t *testing.T) {
	c := getClient(t)

	r := make([]byte, 6)
	rand.Read(r)
	name := "test-" + hex.EncodeToString(r)

	b, err := c.CreateBucket(name, false)
	if err != nil {
		t.Fatal(err)
	}

	file := make([]byte, 123456)
	rand.Read(file)
	fileID, err := b.Upload(bytes.NewBuffer(file), name, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(fileID)

	if err := b.Delete(); err != nil {
		t.Fatal(err)
	}
}
