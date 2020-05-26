package bucketsync

import (
	"context"
	"log"
	"os"

	tc "github.com/FleekHQ/space-poc/core/textile/client"
	"github.com/FleekHQ/space-poc/core/watcher"
)

type BucketSynchronizer struct {
	folderWatcher *watcher.FolderWatcher
	textileClient *tc.TextileClient
	// textileWatcher
	// textileClient
	// grpcPushNotifier
}

// Creates a new BucketSynchronizer instance
func New(folderWatcher *watcher.FolderWatcher, textileClient *tc.TextileClient /*textileWatcher, grpcPushNotifier */) *BucketSynchronizer {
	return &BucketSynchronizer{
		folderWatcher: folderWatcher,
		textileClient: textileClient,
		// textileWatcher: textileWatcher,
		// grpcPushNotifier
	}
}

func (bs *BucketSynchronizer) folderEventHandler(event watcher.UpdateEvent, fileInfo os.FileInfo, newPath, oldPath string) {
	log.Printf(
		"Event: %s\nNewPath: %s\nOldPath: %s\nFile Name: %s\n",
		event.String(),
		newPath,
		oldPath,
		fileInfo.Name(),
	)

	switch event {
	case watcher.Create:
		// bs.textileClient.create(...)
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
	if err := bs.folderWatcher.Watch(ctx, bs.folderEventHandler); err != nil {
		log.Fatal(err)
		return err
	}

	// if err := bs.textileWatcher.Watch(ctx, textileBucketEventHandler); err != nil {
	// 	log.Fatal(err)
	// 	return err
	// }

	return nil
}
