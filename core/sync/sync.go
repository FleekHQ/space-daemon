package sync

import (
	"context"
	"github.com/FleekHQ/space-poc/core/events"
	"github.com/FleekHQ/space-poc/core/sync/fs"
	"github.com/FleekHQ/space-poc/core/sync/textile"

	"github.com/FleekHQ/space-poc/log"

	tc "github.com/FleekHQ/space-poc/core/textile/client"
	"github.com/FleekHQ/space-poc/core/watcher"
)

type BucketSynchronizer struct {
	folderWatcher *watcher.FolderWatcher
	textileClient *tc.TextileClient
	fh            *fs.Handler
	th            *textile.Handler
	// textileWatcher
	// th textileHandler
	// NOTE: not sure we need the complete grpc server here, but that could change
	notify func(event events.FileEvent)
}

// Creates a new BucketSynchronizer instance
func New(folderWatcher *watcher.FolderWatcher, textileClient *tc.TextileClient, notify func(event events.FileEvent)) *BucketSynchronizer {
	fh := fs.NewHandler(textileClient)
	th := textile.NewHandler()
	return &BucketSynchronizer{
		folderWatcher: folderWatcher,
		textileClient: textileClient,
		fh:            fh,
		th:            th,
		notify:        notify,
		// textileWatcher: textileWatcher,
	}
}

// func (bs *BucketSynchronizer) textileBucketEventHandler(events textileWatcher.UpdateEvent, hash, filename string) {
// 	case watcher.Create:
//    // NOTE: We might want to notify the FE that a file got uploaded in the bucket instead of downloading it
//    // That way the user can decide if they want to bring it over or "leave it on the cloud".
// 		bs.grpcPushNotifier.notify("file_creation")
// 	}
// }

// Starts the folder watcher and the textile watcher.
func (bs *BucketSynchronizer) Start(ctx context.Context) error {
	bs.folderWatcher.RegisterHandler(bs.fh)
	if err := bs.folderWatcher.Watch(ctx); err != nil {
		log.Fatal(err)
		return err
	}

	// if err := bs.textileWatcher.Watch(ctx, textileBucketEventHandler); err != nil {
	// 	log.Fatal(err)
	// 	return err
	// }

	return nil
}

func (bs *BucketSynchronizer) Stop() {
	// add shutdown logic here
	bs.folderWatcher.Close()
}
