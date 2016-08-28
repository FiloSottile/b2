package b2_test

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"sync"
	"testing"

	"github.com/FiloSottile/b2"
)

var client *b2.Client
var clientMu sync.Mutex

func getClient(t *testing.T) *b2.Client {
	accountID := os.Getenv("ACCOUNT_ID")
	applicationKey := os.Getenv("APPLICATION_KEY")
	if accountID == "" || applicationKey == "" {
		t.Fatal("Missing ACCOUNT_ID or APPLICATION_KEY")
	}
	clientMu.Lock()
	defer clientMu.Unlock()
	if client != nil {
		return client
	}
	c, err := b2.NewClient(accountID, applicationKey, nil)
	if err != nil {
		t.Fatal("While authenticating:", err)
	}
	client = c
	return c
}

func TestBucketLifecycle(t *testing.T) {
	c := getClient(t)

	r := make([]byte, 6)
	rand.Read(r)
	name := "test-" + hex.EncodeToString(r)

	b, err := c.CreateBucket(name, false)
	if err != nil {
		t.Fatal(err)
	}
	buckets, err := c.Buckets()
	if err != nil {
		t.Fatal(err)
	}
	if bb, ok := buckets[name]; !ok {
		t.Fatal("Bucket did not appear in Buckets()")
	} else if bb.ID != b.ID {
		t.Fatal("Bucket ID mismatch:", b.ID, bb.ID)
	}
	if err := b.Delete(); err != nil {
		t.Fatal(err)
	}
	buckets, err = c.Buckets()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := buckets[name]; ok {
		t.Fatal("Bucket did not disappear from Buckets()")
	}
}
