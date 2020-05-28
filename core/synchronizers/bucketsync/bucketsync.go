package bucketsync

import (
	"context"

	"github.com/FleekHQ/space-poc/log"

	tc "github.com/FleekHQ/space-poc/core/textile/client"
	textileHandler "github.com/FleekHQ/space-poc/core/textile/handler"
	tl "github.com/FleekHQ/space-poc/core/textile/listener"
	"github.com/FleekHQ/space-poc/core/watcher"
)

type BucketSynchronizer struct {
	folderWatcher         *watcher.FolderWatcher
	textileClient         *tc.TextileClient
	textileHandler        *textileHandler.TextileHandler
	textileThreadListener *tl.TextileThreadListener
	// textileClient
	// grpcPushNotifier
}

// Creates a new BucketSynchronizer instance
func New(folderWatcher *watcher.FolderWatcher, textileClient *tc.TextileClient, textileThreadListener *tl.TextileThreadListener) *BucketSynchronizer {
	th := textileHandler.New(textileClient)
	return &BucketSynchronizer{
		folderWatcher:         folderWatcher,
		textileClient:         textileClient,
		textileHandler:        th,
		textileThreadListener: textileThreadListener,
		// grpcPushNotifier
	}
}

// Starts the folder watcher and the textile watcher.
func (bs *BucketSynchronizer) Start(ctx context.Context) error {
	bs.folderWatcher.RegisterHandler(bs.textileHandler)
	log.Debug("Starting folder watcher in bucketsync")
	if err := bs.folderWatcher.Watch(ctx); err != nil {
		log.Fatal(err)
		return err
	}

	// TODO: bs.textileThreadListener.RegisterHandler()
	log.Debug("Starting textile thread listener in bucketsync")
	if err := bs.textileThreadListener.Listen(ctx); err != nil {
		log.Fatal(err)
		return err
	}

	return nil
}

func (bs *BucketSynchronizer) Stop() {
	// add shutdown logic here
	log.Debug("shutting down folder watcher in bucketsync")
	bs.folderWatcher.Close()
	log.Debug("shutting down textile thread listener in bucketsync")
	bs.textileThreadListener.Close()
}
