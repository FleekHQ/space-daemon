package sync

import (
	"context"

	"github.com/FleekHQ/space-daemon/core/events"
)

type EventNotifier interface {
	SendFileEvent(event events.FileEvent)
}

type Synchronizer interface {
	NotifyItemAdded(bucket, path string)
	NotifyItemRemoved(bucket, path string)
	NotifyBucketCreated(bucket string, enckey []byte)
	NotifyBucketBackupOn(bucket string)
	NotifyBucketBackupOff(bucket string)
	NotifyBucketRestore(bucket string)
	NotifyFileRestore(bucket, path string)
	NotifyBucketStartup(bucket string)
	NotifyIndexItemAdded(bucket, path, dbId string)
	Start(ctx context.Context)
	RestoreQueue() error
	Shutdown()
	String() string
	AttachNotifier(EventNotifier)
}
