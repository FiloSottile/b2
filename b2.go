package b2

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

// B2Error is the decoded B2 JSON error return value. It's not the only type of
// error returned by this package, you can check for it with a type assertion.
type B2Error struct {
	Code    string
	Message string
	Status  int
}

func (e *B2Error) Error() string {
	return fmt.Sprintf("b2 remote error [%s]: %s", e.Code, e.Message)
}

const (
	defaultApiUrl = "https://api.backblaze.com"
	apiPath       = "/b2api/v1/"
)

// A Client is an authenticated API client. It is safe for concurrent use and should
// be reused to take advantage of connection and URL reuse.
type Client struct {
	AccountID          string
	AuthorizationToken string
	ApiURL             string
	DownloadURL        string

	hc *http.Client
}

// NewClient calls b2_authorize_account and returns an authenticated Client.
// httpClient can be nil, in which case http.DefaultClient will be used.
func NewClient(accountID, applicationKey string, httpClient *http.Client) (*Client, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	r, err := http.NewRequest("GET", defaultApiUrl+apiPath+"b2_authorize_account", nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString(
		[]byte(accountID+":"+applicationKey)))

	res, err := httpClient.Do(r)
	if err != nil {
		return nil, err
	}
	defer drainAndClose(res.Body)

	if res.StatusCode != 200 {
		b2Err := &B2Error{}
		if err := json.NewDecoder(res.Body).Decode(b2Err); err != nil {
			return nil, fmt.Errorf("unknown error during b2_authorize_account: %d", res.StatusCode)
		}
		return nil, b2Err
	}

	c := &Client{hc: httpClient}
	if err := json.NewDecoder(res.Body).Decode(c); err != nil {
		return nil, fmt.Errorf("failed to decode b2_authorize_account answer: %s", err)
	}
	switch {
	case c.AccountID == "":
		return nil, errors.New("b2_authorize_account answer missing accountId")
	case c.AuthorizationToken == "":
		return nil, errors.New("b2_authorize_account answer missing authorizationToken")
	case c.ApiURL == "":
		return nil, errors.New("b2_authorize_account answer missing apiUrl")
	case c.DownloadURL == "":
		return nil, errors.New("b2_authorize_account answer missing downloadUrl")
	}
	return c, nil
}

func (c *Client) doRequest(endpoint string, params map[string]string) (*http.Response, error) {
	if params == nil {
		params = make(map[string]string)
	}
	params["accountId"] = c.AccountID
	body, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequest("POST", c.ApiURL+apiPath+endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	r.Header.Set("Authorization", c.AuthorizationToken)

	res, err := c.hc.Do(r)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		defer drainAndClose(res.Body)
		b2Err := &B2Error{}
		if err := json.NewDecoder(res.Body).Decode(b2Err); err != nil {
			return nil, fmt.Errorf("unknown error during b2_authorize_account: %d", res.StatusCode)
		}
		return nil, b2Err
	}

	return res, nil
}

// drainAndClose will make an attempt at flushing and closing the body so that the
// underlying connection can be reused.  It will not read more than 10KB.
func drainAndClose(body io.ReadCloser) {
	io.CopyN(ioutil.Discard, body, 10*1024)
	body.Close()
}

// A Bucket is bound to the Client that created it. It is safe for concurrent use and
// should be reused to take advantage of connection and URL reuse.
type Bucket struct {
	ID string `json:"bucketId"`
	c  *Client
}

// BucketByID returns a Bucket bound to the Client. It does NOT check that the
// bucket actually exists, or perform any network operation.
func (c *Client) BucketByID(bucketID string) *Bucket {
	return &Bucket{
		ID: bucketID,
		c:  c,
	}
}

// Buckets returns a map of bucket names to Bucket objects (b2_list_buckets).
func (c *Client) Buckets() (map[string]*Bucket, error) {
	res, err := c.doRequest("b2_list_buckets", nil)
	if err != nil {
		return nil, err
	}
	defer drainAndClose(res.Body)
	var buckets struct {
		Buckets []struct {
			BucketID, BucketName string
		}
	}
	if err := json.NewDecoder(res.Body).Decode(&buckets); err != nil {
		return nil, err
	}
	m := make(map[string]*Bucket)
	for _, b := range buckets.Buckets {
		m[b.BucketName] = &Bucket{
			ID: b.BucketID,
			c:  c,
		}
	}
	return m, nil
}

// CreateBucket creates a bucket with b2_create_bucket. If allPublic is true,
// files in this bucket can be downloaded by anybody.
func (c *Client) CreateBucket(name string, allPublic bool) (*Bucket, error) {
	bucketType := "allPrivate"
	if allPublic {
		bucketType = "allPublic"
	}
	res, err := c.doRequest("b2_create_bucket", map[string]string{
		"bucketName": name,
		"bucketType": bucketType,
	})
	if err != nil {
		return nil, err
	}
	defer drainAndClose(res.Body)
	bucket := &Bucket{c: c}
	if err := json.NewDecoder(res.Body).Decode(bucket); err != nil {
		return nil, err
	}
	return bucket, nil
}

// Delete calls b2_delete_bucket. After this call succeeds the Bucket object
// becomes invalid and any other calls will fail.
func (b *Bucket) Delete() error {
	res, err := b.c.doRequest("b2_delete_bucket", map[string]string{
		"bucketId": b.ID,
	})
	if err != nil {
		return err
	}
	drainAndClose(res.Body)
	return nil
}
