package client

import (
	"context"
	"github.com/FleekHQ/space-poc/log"
	"io"
)

// GetFile pulls path from bucket writing it to writer if it's a file.
func (tc *textileClient) GetFile(ctx context.Context, bucketKey string, path string, w io.Writer) error {
	if err := tc.buckets.PullPath(ctx, bucketKey, path, w); err != nil {
		log.Error("error in GetFile from textile client", err)
		return err
	}

	return nil
}

