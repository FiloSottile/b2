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

	fi, err = b.GetFileInfoByName("test-foo")
	if fi.ID != fileID {
		t.Error("Mismatched file ID in GetByName")
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

	for i := 0; i < 2; i++ {
		if _, err := b.Upload(bytes.NewReader(file), "test-3", ""); err != nil {
			t.Fatal(err)
		}
	}

	var fileIDs []string
	for i := 0; i < 5; i++ {
		fileID, err := b.Upload(bytes.NewReader(file), fmt.Sprintf("test-%d", i), "")
		if err != nil {
			t.Fatal(err)
		}
		fileIDs = append(fileIDs, fileID)
	}

	i, l := 0, b.ListFiles("", 0)
	for l.Next() {
		fi := l.FileInfo()
		if fi.ID != fileIDs[i] {
			t.Errorf("wrong file ID number %d: expected %s, got %s", i, fileIDs[i], fi.ID)
		}
		i++
	}
	if err := l.Err(); err != nil {
		t.Fatal(err)
	}
	if i != len(fileIDs) {
		t.Errorf("got %d files, expected %d", i-1, len(fileIDs)-1)
	}

	i, l = 1, b.ListFiles("test-1", 3)
	for l.Next() {
		fi := l.FileInfo()
		if fi.ID != fileIDs[i] {
			t.Errorf("wrong file ID number %d: expected %s, got %s", i, fileIDs[i], fi.ID)
		}
		i++
	}
	if err := l.Err(); err != nil {
		t.Fatal(err)
	}
	if i != len(fileIDs) {
		t.Errorf("got %d files, expected %d", i-1, len(fileIDs)-1)
	}

	i, l = 0, b.ListFilesVersions("", "", 2)
	for l.Next() {
		i++
	}
	if err := l.Err(); err != nil {
		t.Fatal(err)
	}
	if i != len(fileIDs)+2 {
		t.Errorf("got %d files, expected %d", i-1, len(fileIDs)-1+2)
	}
}
