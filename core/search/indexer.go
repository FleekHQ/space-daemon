package search

import (
	"context"

	"github.com/FleekHQ/space-daemon/core/textile/utils/stack"

	"github.com/FleekHQ/space-daemon/core/textile/bucket"

	synchronizer "github.com/FleekHQ/space-daemon/core/textile/sync"

	"github.com/FleekHQ/space-daemon/core/textile"
	"github.com/FleekHQ/space-daemon/log"
)

type filesSearchIndexer struct {
	sync      synchronizer.Synchronizer
	isRunning bool
}

func NewFilesSearchIndexer() filesSearchIndexer {
	return filesSearchIndexer{
		isRunning: false,
	}
}

func (f *filesSearchIndexer) Start(ctx context.Context, tc textile.Client, sync synchronizer.Synchronizer) {
	if f.isRunning {
		log.Warn("filesSearchIndexer is already running")
		return
	}

	// sync buckets
	go func() {
		bucketsList := stack.New()
		buckets, err := tc.ListBuckets(ctx)
		entries, err := bucket.ListDirectory(ctx, "/")
		if err != nil {
			// something
		}

		stack = append(stack, entries.Item.Path)

		for len(stack) != 0 {
			sync.NotifyIndexItemAdded(bucket.Slug(), entries.Item.Path, "")
			for _, entry := range entries.Item.Items {
				f.sync.NotifyIndexItemAdded(bucket.Slug(), entry.Path, "")
				if entry.IsDir {
					// push children on the stack
					stack = append(stack, entry.Path)
				}
			}
		}
	}()

	// sync shared with me files
	go func() {

	}()
}

func (f *filesSearchIndexer) Shutdown() error {
	f.isRunning = false

	// TODO: Store stack in storage
	return nil
}
