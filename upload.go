package b2

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// Upload uploads a file to a B2 bucket. If mimeType is "", "b2/x-auto" will be used.
// It returns the file ID to be used to stat/delete/download it.
//
// Concurrent calls to Upload will use separate upload URLs, but consequent ones
// will attempt to reuse previously obtained ones to save b2_get_upload_url calls.
//
// Since the B2 API requires a SHA1 header, normally the file will first be read
// entirely into a memory buffer. Two cases avoid the memory copy: if r is a
// bytes.Buffer, the SHA1 will be computed in place; instead if r implements io.Seeker
// the file will be read twice, once to compute the SHA1 and once to upload.
func (b *Bucket) Upload(r io.Reader, name, mimeType string) (string, error) {
	var body io.Reader
	var length int
	h := sha1.New()

	switch r := r.(type) {
	case *bytes.Buffer:
		h.Write(r.Bytes())
		body, length = r, r.Len()
	case io.ReadSeeker:
		n, err := io.Copy(h, r)
		if err != nil {
			return "", err
		}
		if _, err := r.Seek(0, io.SeekStart); err != nil {
			return "", err
		}
		body, length = r, int(n)
	default:
		buf := &bytes.Buffer{}
		if _, err := buf.ReadFrom(io.TeeReader(r, h)); err != nil {
			return "", err
		}
		body, length = buf, buf.Len()
	}

	return b.UploadWithSHA1(body, name, mimeType, hex.EncodeToString(h.Sum(nil)), length)
}

// UploadWithSHA1 is like Upload, but allows the caller to specify previously
// known SHA1 and length of the file. It never does any buffering.
//
// sha1sum should be the hex encoding of the SHA1 sum of what will be read from r.
//
// This is an advanced interface, common clients should use Upload, and consider
// passing it a bytes.Buffer or io.ReadSeeker to avoid buffering.
func (b *Bucket) UploadWithSHA1(r io.Reader, name, mimeType, sha1sum string, length int) (string, error) {
	res, err := b.c.doRequest("b2_get_upload_url", map[string]string{
		"bucketId": b.ID,
	})
	if err != nil {
		return "", err
	}
	defer drainAndClose(res.Body)
	var urlRes struct {
		UploadURL, AuthorizationToken string
	}
	if err := json.NewDecoder(res.Body).Decode(&urlRes); err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", urlRes.UploadURL, r)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", urlRes.AuthorizationToken)
	req.Header.Set("X-Bz-File-Name", url.QueryEscape(name))
	req.Header.Set("Content-Type", mimeType)
	req.Header.Set("Content-Length", strconv.Itoa(length))
	req.Header.Set("X-Bz-Content-Sha1", sha1sum)
	res, err = b.c.hc.Do(req)
	if err != nil {
		return "", err
	}
	var fileRes struct {
		FileID string
	}
	if err := json.NewDecoder(res.Body).Decode(&fileRes); err != nil {
		return "", err
	}
	return fileRes.FileID, nil
}
