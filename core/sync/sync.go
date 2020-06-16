package sync

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/FleekHQ/space-poc/core/events"
	"github.com/FleekHQ/space-poc/core/space/domain"
	"github.com/FleekHQ/space-poc/core/store"
	"github.com/FleekHQ/space-poc/core/textile"
	"github.com/FleekHQ/space-poc/log"
	"golang.org/x/sync/errgroup"

	"github.com/FleekHQ/space-poc/core/watcher"
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
	Start(ctx context.Context) error
	Stop()
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
	folderWatcher          watcher.FolderWatcher
	textileClient          textile.Client
	fh                     *watcherHandler
	th                     *textileHandler
	textileThreadListeners []textile.ThreadListener
	notifier               GrpcNotifier
	store                  store.Store
}

// Creates a new bucketSynchronizer instancelistenerEventHandler
func New(
	folderWatcher watcher.FolderWatcher,
	textileClient textile.Client,
	store store.Store,
	notifier GrpcNotifier,
) BucketSynchronizer {
	textileThreadListeners := make([]textile.ThreadListener, 0)

	return &bucketSynchronizer{
		folderWatcher:          folderWatcher,
		textileClient:          textileClient,
		fh:                     nil,
		th:                     nil,
		textileThreadListeners: textileThreadListeners,
		notifier:               notifier,
		store:                  store,
	}
}

// Starts the folder watcher and the textile watcher.
func (bs *bucketSynchronizer) Start(ctx context.Context) error {
	buckets, err := bs.textileClient.ListBuckets(ctx)
	if err != nil {
		return err
	}

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

	handlers := make([]textile.EventHandler, 0)
	handlers = append(handlers, bs.th)

	for _, bucket := range buckets {
		bs.textileThreadListeners = append(bs.textileThreadListeners, textile.NewListener(bs.textileClient, bucket.Slug(), handlers))
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

	// TODO: add files in store to watcher on boot

	err = g.Wait()

	if err != nil {
		return err
	}

	return nil
}

func (bs *bucketSynchronizer) Stop() {
	// add shutdown logic here
	log.Debug("shutting down folder watcher in bucketsync")
	bs.folderWatcher.Close()
	log.Debug("shutting down textile thread listener in bucketsync")
	for _, listener := range bs.textileThreadListeners {
		listener.Close()
	}
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

