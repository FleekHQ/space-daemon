package sync

import (
	"context"
)

type Synchronizer interface {
	NotifyItemAdded(bucket, path string)
	NotifyItemRemoved(bucket, path string)
	NotifyBucketCreated(bucket string, enckey []byte)
	Start(ctx context.Context)
	RestoreQueue() error
	Shutdown()
}
