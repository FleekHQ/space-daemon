package sync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/FleekHQ/space-daemon/core/events"
	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/core/space/services"
	"github.com/FleekHQ/space-daemon/core/store"
	"github.com/FleekHQ/space-daemon/core/textile"
	"github.com/FleekHQ/space-daemon/log"
	"golang.org/x/sync/errgroup"

	"github.com/FleekHQ/space-daemon/core/watcher"
)

var (
	ErrAddFileWatch = errors.New("error adding file to watch")
)

const (
	OpenFilesKeyPrefix        = "openFiles#"
	ReverseOpenFilesKeyPrefix = "reverseOpenFiles#"
)

type GrpcNotifier interface {
	SendFileEvent(event events.FileEvent)
	SendTextileEvent(event events.TextileEvent)
}

type BucketSynchronizer interface {
	WaitForReady() chan bool
	Start(ctx context.Context) error
	Shutdown() error
	RegisterNotifier(notifier GrpcNotifier)
	AddFileWatch(addFileInfo domain.AddWatchFile) error
	GetOpenFilePath(bucketSlug string, bucketPath string) (string, bool)
}

type TextileNotifier interface {
	SendTextileEvent(event events.TextileEvent)
}

// Implementation to handle events from FS
type watcherHandler struct {
	client textile.Client
	bs     *bucketSynchronizer
}

// Implementation to handle events from textile
type textileHandler struct {
	notifier TextileNotifier
	bs       *bucketSynchronizer
}

type bucketSynchronizer struct {
	folderWatcher watcher.FolderWatcher
	textileClient textile.Client
	fh            *watcherHandler
	th            *textileHandler
	// textileThreadListeners []textile.ThreadListener
	notifier GrpcNotifier
	store    store.Store
	ready    chan bool
}

// Creates a new bucketSynchronizer instancelistenerEventHandler
func New(
	folderWatcher watcher.FolderWatcher,
	textileClient textile.Client,
	store store.Store,
	notifier GrpcNotifier,
) *bucketSynchronizer {
	// textileThreadListeners := make([]textile.ThreadListener, 0)

	return &bucketSynchronizer{
		folderWatcher: folderWatcher,
		textileClient: textileClient,
		fh:            nil,
		th:            nil,
		// textileThreadListeners: textileThreadListeners,
		notifier: notifier,
		store:    store,
		ready:    make(chan bool),
	}
}

// Starts the folder watcher and the textile watcher.
func (bs *bucketSynchronizer) Start(ctx context.Context) error {
	// Disabling this temporarily due to errors
	//buckets, err := bs.textileClient.ListBuckets(ctx)
	// if err != nil {
	// 	return err
	// }

	if bs.notifier == nil {
		log.Printf("using default notifier to start bucket sync")
		bs.notifier = &defaultNotifier{}
	}

	bs.fh = &watcherHandler{
		client: bs.textileClient,
		bs:     bs,
	}

	bs.th = &textileHandler{
		notifier: bs.notifier,
		bs:       bs,
	}

	// handlers := make([]textile.EventHandler, 0)
	// handlers = append(handlers, bs.th)

	// Disabling this temporarily due to errors
	//for range buckets {
	// bs.textileThreadListeners = append(bs.textileThreadListeners, textile.NewListener(bs.textileClient, bucket.Slug(), handlers))
	//}

	bs.folderWatcher.RegisterHandler(bs.fh)

	// TODO: bs.textileThreadListener.RegisterHandler(bs.th)
	// (Needs implementation of bs.th)

	g, newCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		log.Debug("Starting watcher in bucketsync")
		return bs.folderWatcher.Watch(newCtx)
	})

	// for _, listener := range bs.textileThreadListeners {
	// 	g.Go(func() error {
	// 		log.Debug("Starting textile thread listener in bucketsync")
	// 		return listener.Listen(newCtx)
	// 	})
	// }

	// add open files to watcher
	keys, err := bs.store.KeysWithPrefix(OpenFilesKeyPrefix)
	if err != nil {
		log.Error("error getting keys from store", err)
		return err
	}
	log.Debug("start watching open files ...")
	for _, k := range keys {
		if fi, err := bs.getOpenFileInfo(k); err == nil {
			if services.PathExists(fi.LocalPath) {
				if err := bs.folderWatcher.AddFile(fi.LocalPath); err != nil {
					log.Error(fmt.Sprintf("error opening file at %s", fi.LocalPath), err)
					// remove fileInfo from store for cleanup
					bs.removeFileInfo(fi)
				}
			}
		}
	}

	//go func() {
	bs.ready <- true
	//}()

	err = g.Wait()

	if err != nil {
		return err
	}

	return nil
}

func (bs *bucketSynchronizer) WaitForReady() chan bool {
	return bs.ready
}

func (bs *bucketSynchronizer) Shutdown() error {
	// add shutdown logic here
	log.Debug("shutting down folder watcher in bucketsync")
	bs.folderWatcher.Close()
	log.Debug("shutting down textile thread listener in bucketsync")
	// for _, listener := range bs.textileThreadListeners {
	// 	listener.Close()
	// }

	close(bs.ready)
	return nil
}

func (bs *bucketSynchronizer) RegisterNotifier(notifier GrpcNotifier) {
	bs.notifier = notifier
}

// TODO: add GC code logic to open files to cleanup
// Adds a file to watcher list to keep track of
func (bs *bucketSynchronizer) AddFileWatch(addFileInfo domain.AddWatchFile) error {
	if addFileInfo.LocalPath == "" {
		return ErrAddFileWatch
	}
	if addFileInfo.BucketKey == "" {
		return ErrAddFileWatch
	}

	if addFileInfo.BucketPath == "" {
		return ErrAddFileWatch
	}

	err := bs.addFileInfoToStore(addFileInfo)
	if err != nil {
		return err
	}

	err = bs.folderWatcher.AddFile(addFileInfo.LocalPath)
	if err != nil {
		return err
	}

	return nil
}

func (bs *bucketSynchronizer) GetOpenFilePath(bucketSlug string, bucketPath string) (string, bool) {
	var fi domain.AddWatchFile
	var err error
	reversKey := getOpenFileReverseKey(bucketSlug, bucketPath)

	if fi, err = bs.getOpenFileInfo(reversKey); err != nil {
		return "", false
	}

	if fi.LocalPath == "" {
		return "", false
	}

	return fi.LocalPath, true
}

func getOpenFileKey(localPath string) string {
	return OpenFilesKeyPrefix + localPath
}

func getOpenFileReverseKey(bucketSlug string, bucketPath string) string {
	return ReverseOpenFilesKeyPrefix + bucketSlug + ":" + bucketPath
}

func (bs *bucketSynchronizer) getOpenFileBucketKey(localPath string) (string, bool) {
	var fi domain.AddWatchFile
	var err error
	if fi, err = bs.getOpenFileInfo(getOpenFileKey(localPath)); err != nil {
		return "", false
	}

	if fi.BucketKey == "" {
		return "", false
	}

	return fi.BucketKey, true
}

// Helper function to set open file info in the store
func (bs *bucketSynchronizer) addFileInfoToStore(addFileInfo domain.AddWatchFile) error {
	out, err := json.Marshal(addFileInfo)
	if err != nil {
		return err
	}
	if err := bs.store.SetString(getOpenFileKey(addFileInfo.LocalPath), string(out)); err != nil {
		return err
	}
	reverseKey := getOpenFileReverseKey(addFileInfo.BucketKey, addFileInfo.BucketPath)
	if err := bs.store.SetString(reverseKey, string(out)); err != nil {
		return err
	}
	return nil
}

// Helper function to remove file information from store
func (bs *bucketSynchronizer) removeFileInfo(addFileInfo domain.AddWatchFile) error {
	if err := bs.store.Remove([]byte(getOpenFileKey(addFileInfo.LocalPath))); err != nil {
		return err
	}
	reverseKey := getOpenFileReverseKey(addFileInfo.BucketKey, addFileInfo.BucketPath)
	if err := bs.store.Remove([]byte(reverseKey)); err != nil {
		return err
	}
	return nil
}

// Helper function to retrieve open file info from store
func (bs *bucketSynchronizer) getOpenFileInfo(key string) (domain.AddWatchFile, error) {
	var fi []byte
	var err error

	if fi, err = bs.store.Get([]byte(key)); err != nil {
		return domain.AddWatchFile{}, err
	}

	var fileInfo domain.AddWatchFile

	if err := json.Unmarshal(fi, &fileInfo); err != nil {
		return domain.AddWatchFile{}, err
	}

	return fileInfo, nil
}
