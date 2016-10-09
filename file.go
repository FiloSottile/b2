package b2

import (
	"encoding/json"
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
	ID       string
	Name     string
	BucketID string

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
		BucketID:        fi.BucketID,
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

// ListFiles returns at most maxCount files in the Bucket, alphabetically sorted,
// starting from the file named fromName (included if it exists).
//
// If there are files left in the bucket, next is set to the value you should pass
// as fromName to continue iterating. To iterate all files in a bucket do:
//
// 	for fromName := new(string); fromName != nil; {
//		var res []*FileInfo
//		var err error
// 		res, fromName, err = b2.ListFiles(*fromName, 100)
//		if err != nil {
//			log.Fatal(err)
//		}
//		for _, fi := range res {
//			// ...
//		}
// 	}
//
// ListFiles only returns the most recent version of each file, and does not return
// hidden files. To get those, use ListFilesVersions.
func (b *Bucket) ListFiles(fromName string, maxCount int) (res []*FileInfo, next *string, err error) {
	r, err := b.c.doRequest("b2_list_file_names", map[string]interface{}{
		"bucketId":      b.ID,
		"startFileName": fromName,
		"maxFileCount":  maxCount,
	})
	if err != nil {
		return nil, nil, err
	}
	defer drainAndClose(r.Body)

	var x struct {
		Files        []fileInfoObj
		NextFileName *string
	}
	if err := json.NewDecoder(r.Body).Decode(&x); err != nil {
		return nil, nil, err
	}

	for _, fi := range x.Files {
		res = append(res, fi.makeFileInfo())
	}

	return res, x.NextFileName, nil
}
