package bucketsync

import (
	"context"

	"github.com/FleekHQ/space-poc/log"

	tc "github.com/FleekHQ/space-poc/core/textile/client"
	textileHandler "github.com/FleekHQ/space-poc/core/textile/handler"
	"github.com/FleekHQ/space-poc/core/watcher"
)

type BucketSynchronizer struct {
	folderWatcher  *watcher.FolderWatcher
	textileClient  *tc.TextileClient
	textileHandler *textileHandler.TextileHandler
	// textileWatcher
	// textileClient
	// grpcPushNotifier
}

// Creates a new BucketSynchronizer instance
func New(folderWatcher *watcher.FolderWatcher, textileClient *tc.TextileClient /*textileWatcher, grpcPushNotifier */) *BucketSynchronizer {
	th := textileHandler.New(textileClient)
	return &BucketSynchronizer{
		folderWatcher:  folderWatcher,
		textileClient:  textileClient,
		textileHandler: th,
		// textileWatcher: textileWatcher,
		// grpcPushNotifier
	}
}

// func (bs *BucketSynchronizer) textileBucketEventHandler(event textileWatcher.UpdateEvent, hash, filename string) {
// 	case watcher.Create:
//    // NOTE: We might want to notify the FE that a file got uploaded in the bucket instead of downloading it
//    // That way the user can decide if they want to bring it over or "leave it on the cloud".
// 		bs.grpcPushNotifier.notify("file_creation")
// 	}
// }

// Starts the folder watcher and the textile watcher.
func (bs *BucketSynchronizer) Start(ctx context.Context) error {
	bs.folderWatcher.RegisterHandler(bs.textileHandler)
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
