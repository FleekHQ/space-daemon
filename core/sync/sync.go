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
	folderWatcher         *watcher.FolderWatcher
	textileClient         *tc.TextileClient
	fh                    *fs.Handler
	th                    *textile.Handler
	textileThreadListener *tl.TextileThreadListener

	// NOTE: not sure we need the complete grpc server here, but that could change
	notify func(event events.FileEvent)
}

// Creates a new BucketSynchronizer instance
func New(folderWatcher *watcher.FolderWatcher, textileClient *tc.TextileClient, notify func(event events.FileEvent)) *BucketSynchronizer {
	fh := fs.NewHandler(textileClient)
	th := textile.NewHandler()

	// TODO: Iterate over each of the user buckets and create a listener for each one of them
	textileThreadListener := tl.New(textileClient, tc.DefaultPersonalBucketSlug)

	return &BucketSynchronizer{
		folderWatcher:         folderWatcher,
		textileClient:         textileClient,
		fh:                    fh,
		th:                    th,
		notify:                notify,
		textileThreadListener: textileThreadListener,
	}
}

// Starts the folder watcher and the textile watcher.
func (bs *BucketSynchronizer) Start(ctx context.Context) error {
	bs.folderWatcher.RegisterHandler(bs.fh)

	// TODO: bs.textileThreadListener.RegisterHandler(bs.th)
	// (Needs implementation of bs.th)

	g, newCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		log.Debug("Starting watcher in bucketsync")
		return bs.folderWatcher.Watch(newCtx)
	})

	g.Go(func() error {
		log.Debug("Starting textile thread listener in bucketsync")
		return bs.textileThreadListener.Listen(newCtx)
	})

	return nil
}

func (bs *BucketSynchronizer) Stop() {
	// add shutdown logic here
	log.Debug("shutting down folder watcher in bucketsync")
	bs.folderWatcher.Close()
	log.Debug("shutting down textile thread listener in bucketsync")
	bs.textileThreadListener.Close()
}
