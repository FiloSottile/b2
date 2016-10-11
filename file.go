package b2

import (
	"encoding/json"
	"errors"
	"time"
)

// DeleteFile deletes a file version.
func (c *Client) DeleteFile(id, name string) error {
	res, err := c.doRequest("b2_delete_file_version", map[string]interface{}{
		"fileId": id, "fileName": name,
	})
	if err != nil {
		return err
	}
	drainAndClose(res.Body)
	return nil
}

// A FileInfo is the metadata associated with a specific file version.
type FileInfo struct {
	ID   string
	Name string

	// Had to remove BucketID since it is not returned by b2_download_file_by_*
	// BucketID string

	ContentSHA1   string
	ContentLength int
	ContentType   string

	CustomMetadata  map[string]interface{}
	UploadTimestamp time.Time

	// If Action is "hide", this ID does not refer to a file version
	// but to an hiding action.
	Action string
}

type fileInfoObj struct {
	AccountID       string                 `json:"accountId"`
	BucketID        string                 `json:"bucketId"`
	ContentLength   int                    `json:"contentLength"`
	ContentSHA1     string                 `json:"contentSha1"`
	ContentType     string                 `json:"contentType"`
	FileID          string                 `json:"fileId"`
	FileInfo        map[string]interface{} `json:"fileInfo"`
	FileName        string                 `json:"fileName"`
	UploadTimestamp int64                  `json:"uploadTimestamp"`
	Action          string                 `json:"action"`
}

func (fi *fileInfoObj) makeFileInfo() *FileInfo {
	return &FileInfo{
		ID:              fi.FileID,
		Name:            fi.FileName,
		ContentLength:   fi.ContentLength,
		ContentSHA1:     fi.ContentSHA1,
		ContentType:     fi.ContentType,
		CustomMetadata:  fi.FileInfo,
		Action:          fi.Action,
		UploadTimestamp: time.Unix(fi.UploadTimestamp/1e3, fi.UploadTimestamp%1e3*1e6),
	}
}

// GetFileInfoByID obtains a FileInfo for a given ID.
//
// The ID can refer to any file version or "hide" action in any bucket.
func (c *Client) GetFileInfoByID(id string) (*FileInfo, error) {
	res, err := c.doRequest("b2_get_file_info", map[string]interface{}{
		"fileId": id,
	})
	if err != nil {
		return nil, err
	}
	defer drainAndClose(res.Body)
	var fi *fileInfoObj
	if err := json.NewDecoder(res.Body).Decode(&fi); err != nil {
		return nil, err
	}
	return fi.makeFileInfo(), nil
}

// GetFileInfoByName obtains a FileInfo for a given name.
//
// If the file doesn't exist, (nil, nil) is returned.
// If multiple versions of the file exist, only the latest is returned.
func (b *Bucket) GetFileInfoByName(name string) (*FileInfo, error) {
	l := b.ListFiles(name, 1)
	if l.Next() {
		if l.FileInfo().Name == name {
			return l.FileInfo(), nil
		}
	}
	return nil, l.Err()
}

// A Listing is the result of (*Bucket).ListFiles[Versions].
// It works like sql.Rows: use Next to advance and then FileInfo.
// Check Err once Next returns false. It handles pagination transparently.
//
//     l := b.ListFiles("", 50)
//     for l.Next() {
//         fi := l.FileInfo()
//         ...
//     }
//     if err := l.Err(); err != nil {
//         ...
//     }
//
type Listing struct {
	b                *Bucket
	versions         bool
	pageCount        int
	nextName, nextID *string
	objects          []*FileInfo // in reverse order
	err              error
}

// Next calls the list API and prepares the FileInfo results.
// It returns true on success, or false if there is no next result
// row or an error happened while preparing it. Err should be
// consulted to distinguish between the two cases.
func (l *Listing) Next() bool {
	if l.err != nil {
		return false
	}
	if len(l.objects) > 0 {
		l.objects = l.objects[:len(l.objects)-1]
	}
	if len(l.objects) > 0 {
		return true
	}
	if l.nextName == nil {
		return false // end of iteration
	}

	data := map[string]interface{}{
		"bucketId":      l.b.ID,
		"startFileName": *l.nextName,
		"maxFileCount":  l.pageCount,
	}
	endpoint := "b2_list_file_names"
	if l.versions {
		endpoint = "b2_list_file_versions"
	}
	if l.nextID != nil && *l.nextID != "" {
		data["startFileId"] = *l.nextID
	}
	r, err := l.b.c.doRequest(endpoint, data)
	if err != nil {
		l.err = err
		return false
	}
	defer drainAndClose(r.Body)

	var x struct {
		Files        []fileInfoObj
		NextFileName *string
		NextFileID   *string
	}
	if l.err = json.NewDecoder(r.Body).Decode(&x); l.err != nil {
		return false
	}

	l.objects = make([]*FileInfo, len(x.Files))
	for i, f := range x.Files {
		l.objects[len(l.objects)-1-i] = f.makeFileInfo()
	}
	l.nextName, l.nextID = x.NextFileName, x.NextFileID
	return len(l.objects) > 0
}

// FileInfo returns the FileInfo object made available by Next.
//
// FileInfo must only be called after a call to Next returned true.
func (l *Listing) FileInfo() *FileInfo {
	return l.objects[len(l.objects)-1]
}

// Err returns the error, if any, that was encountered while listing.
func (l *Listing) Err() error {
	return l.err
}

// ListFiles returns a Listing of files in the Bucket, alphabetically sorted,
// starting from the file named fromName (included if it exists). To start from
// the first file in the bucket, set fileName to "".
//
// ListFiles only returns the most recent version of each (non-hidden) file.
// If you want to fetch all versions, use ListFilesVersions.
func (b *Bucket) ListFiles(fromName string, pageCount int) *Listing {
	return &Listing{
		b:         b,
		nextName:  &fromName,
		pageCount: pageCount,
	}
}

// ListFilesVersions is like ListFiles, but returns all file versions,
// alphabetically sorted first, and by reverse of date/time uploaded then.
//
// If fromID is specified, the name-and-id pair is the starting point.
func (b *Bucket) ListFilesVersions(fromName, fromID string, pageCount int) *Listing {
	if fromName == "" && fromID != "" {
		return &Listing{
			err: errors.New("can't set fromID if fromName is not set"),
		}
	}
	return &Listing{
		b:         b,
		versions:  true,
		nextName:  &fromName,
		nextID:    &fromID,
		pageCount: pageCount,
	}
}
