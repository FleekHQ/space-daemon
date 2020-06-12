package sync

import (
	"context"
	"errors"
	"github.com/FleekHQ/space-poc/core/space/domain"
	"github.com/FleekHQ/space-poc/core/textile"
	"golang.org/x/sync/errgroup"
	"sync"

	"github.com/FleekHQ/space-poc/core/events"
	"github.com/FleekHQ/space-poc/log"

	"github.com/FleekHQ/space-poc/core/watcher"
)

var (
	ErrAddFileWatch = errors.New("error adding file to watch")
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

	// lock for openFiles map
	openFilesLock sync.RWMutex
	openFiles     map[string]domain.AddWatchFile
}

// Creates a new bucketSynchronizer instancelistenerEventHandler
func New(
	folderWatcher watcher.FolderWatcher,
	textileClient textile.Client,
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
		openFiles:              make(map[string]domain.AddWatchFile),
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

	bs.openFilesLock.Lock()
	defer bs.openFilesLock.Unlock()
	bs.openFiles[addFileInfo.LocalPath] = addFileInfo

	return nil
}

func (bs *bucketSynchronizer) getOpenFileBucketKey(localPath string) (string, bool) {
	bs.openFilesLock.RLock()
	defer bs.openFilesLock.RUnlock()
	if fi, exists := bs.openFiles[localPath]; exists {
		return fi.BucketKey, true
	}

	return "", false
}
