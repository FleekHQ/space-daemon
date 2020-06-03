package sync

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/FleekHQ/space-poc/core/events"
	"github.com/FleekHQ/space-poc/core/sync/fs"
	"github.com/FleekHQ/space-poc/core/sync/textile"

	"github.com/FleekHQ/space-poc/log"

	tc "github.com/FleekHQ/space-poc/core/textile/client"
	tl "github.com/FleekHQ/space-poc/core/textile/listener"
	"github.com/FleekHQ/space-poc/core/watcher"
)

type BucketSynchronizer struct {
	folderWatcher          *watcher.FolderWatcher
	textileClient          tc.Client
	fh                     *fs.Handler
	th                     *textile.Handler
	textileThreadListeners []tl.ThreadListener

	// NOTE: not sure we need the complete grpc server here, but that could change
	notify func(event events.FileEvent)
}

// Creates a new BucketSynchronizer instance
func New(
	folderWatcher *watcher.FolderWatcher,
	textileClient tc.Client,
	notify func(event events.FileEvent),
) *BucketSynchronizer {
	textileThreadListeners := make([]tl.ThreadListener, 0)

	return &BucketSynchronizer{
		folderWatcher:          folderWatcher,
		textileClient:          textileClient,
		fh:                     nil,
		th:                     nil,
		notify:                 notify,
		textileThreadListeners: textileThreadListeners,
	}
}

// Starts the folder watcher and the textile watcher.
func (bs *BucketSynchronizer) Start(ctx context.Context) error {
	buckets, err := bs.textileClient.ListBuckets()
	if err != nil {
		return err
	}

	// TODO: Generalize this to one per bucket
	bs.fh = fs.NewHandler(bs.textileClient, buckets[0])
	bs.th = textile.NewHandler()

	for _, bucket := range buckets {
		bs.textileThreadListeners = append(bs.textileThreadListeners, tl.New(bs.textileClient, bucket.Name))
	}

	bs.folderWatcher.RegisterHandler(bs.fh)

	// TODO: bs.textileThreadListener.RegisterHandler(bs.th)
	// (Needs implementation of bs.th)

	g, newCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		log.Debug("Starting watcher in bucketsync")
		return bs.folderWatcher.Watch(newCtx)
	})

	for _, listener := range bs.textileThreadListeners {
		g.Go(func() error {
			log.Debug("Starting textile thread listener in bucketsync")
			return listener.Listen(newCtx)
		})
	}

	err = g.Wait()

	if err != nil {
		return err
	}

	return nil
}

func (bs *BucketSynchronizer) Stop() {
	// add shutdown logic here
	log.Debug("shutting down folder watcher in bucketsync")
	bs.folderWatcher.Close()
	log.Debug("shutting down textile thread listener in bucketsync")
	for _, listener := range bs.textileThreadListeners {
		listener.Close()
	}
}
