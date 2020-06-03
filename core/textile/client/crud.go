package client

import (
	"bytes"
	"context"
	"io"
	"strings"

	"github.com/ipfs/interface-go-ipfs-core/path"
)

// UploadFile uploads a file to path on textile
// path should include the file name as the last path segment
// also nested path not existing yet would be created automatically
func (tc *textileClient) UploadFile(
	ctx context.Context,
	bucketKey string,
	path string,
	reader io.Reader,
) (result path.Resolved, root path.Path, err error) {
	ctx, _, err = tc.GetBucketContext(defaultPersonalBucketSlug)
	if err != nil {
		return nil, nil, err
	}
	return tc.buckets.PushPath(ctx, bucketKey, path, reader)
}

// CreateDirectory creates an empty directory
// Because textile doesn't support empty directory an empty .keep file is created
// in the directory
func (tc *textileClient) CreateDirectory(
	ctx context.Context,
	bucketKey string,
	path string,
) (result path.Resolved, root path.Path, err error) {
	ctx, _, err = tc.GetBucketContext(defaultPersonalBucketSlug)
	if err != nil {
		return nil, nil, err
	}
	// append .keep file to the end of the directory
	emptyDirPath := strings.TrimRight(path, "/") + "/" + keepFileName
	return tc.buckets.PushPath(ctx, bucketKey, emptyDirPath, &bytes.Buffer{})
}

// ListDirectory returns a list of items in a particular directory
func (tc *textileClient) ListDirectory(
	ctx context.Context,
	bucketKey string,
	path string,
) (*TextileDirEntries, error) {
	ctx, _, err := tc.GetBucketContext(defaultPersonalBucketSlug)
	if err != nil {
		return nil, err
	}
	result, err := tc.buckets.ListPath(ctx, bucketKey, path)

	return (*TextileDirEntries)(result), err
}

// DeleteDirOrFile will delete file or directory at path
func (tc *textileClient) DeleteDirOrFile(
	ctx context.Context,
	bucketKey string,
	path string,
) (path.Resolved, error) {
	ctx, _, err := tc.GetBucketContext(defaultPersonalBucketSlug)
	if err != nil {
		return nil, err
	}
	return tc.buckets.RemovePath(ctx, bucketKey, path)
}
