package b2_test

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/FiloSottile/b2"
)

func getBucket(t *testing.T, c *b2.Client) *b2.Bucket {
	r := make([]byte, 6)
	rand.Read(r)
	name := "test-" + hex.EncodeToString(r)

	b, err := c.CreateBucket(name, false)
	if err != nil {
		t.Fatal(err)
	}

	return &b.Bucket
}

func TestFileLifecycle(t *testing.T) {
	c := getClient(t)
	b := getBucket(t, c)
	defer b.Delete()

	file := make([]byte, 123456)
	rand.Read(file)
	fileID, err := b.Upload(bytes.NewReader(file), "test-foo", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(fileID)

	fi, err := c.GetFileInfoByID(fileID)
	if err != nil {
		t.Fatal(err)
	}
	if fi.ID != fileID {
		t.Error("Mismatched file ID")
	}
	if fi.ContentLength != 123456 {
		t.Error("Mismatched file length")
	}
	if fi.Name != "test-foo" {
		t.Error("Mismatched file name")
	}
	if fi.UploadTimestamp.After(time.Now()) || fi.UploadTimestamp.Before(time.Now().Add(-time.Hour)) {
		t.Error("Wrong UploadTimestamp")
	}

	if err := c.DeleteFile(fileID, "test-foo"); err != nil {
		t.Fatal(err)
	}
}

func TestFileListing(t *testing.T) {
	c := getClient(t)
	b := getBucket(t, c)
	defer b.Delete()

	file := make([]byte, 1234)
	rand.Read(file)

	var fileIDs []string
	for i := 0; i < 5; i++ {
		fileID, err := b.Upload(bytes.NewReader(file), fmt.Sprintf("test-%d", i), "")
		if err != nil {
			t.Fatal(err)
		}
		fileIDs = append(fileIDs, fileID)
	}

	i := 1
	fromName := "test-1"
	for fromName := &fromName; fromName != nil; {
		var res []*b2.FileInfo
		var err error
		res, fromName, err = b.ListFiles(*fromName, 3)
		if err != nil {
			t.Fatal(err)
		}
		if len(res) > 3 {
			t.Errorf("too many returned values: %d", len(res))
		}
		for _, fi := range res {
			if fi.ID != fileIDs[i] {
				t.Errorf("wrong file ID number %d: expected %s, got %s", i, fileIDs[i], fi.ID)
			}
			i++
		}
	}
	if i != len(fileIDs) {
		t.Errorf("got %d files, expected %d", i-1, len(fileIDs)-1)
	}
}
