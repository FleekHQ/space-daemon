package watcher

import (
	"context"
	"errors"
	"fmt"
	s "strings"
	"sync"

	fsutils "github.com/FleekHQ/space-daemon/core/space/services"
	"github.com/mitchellh/go-homedir"

	"time"

	"github.com/radovskyb/watcher"

	"github.com/FleekHQ/space-daemon/log"
)

var (
	ErrFolderPathNotFound = errors.New("could not find a folder path for watcher")
)

type FolderWatcher interface {
	RegisterHandler(handler EventHandler)
	AddFile(path string) error
	Watch(ctx context.Context) error
	Close()
}

type folderWatcher struct {
	w *watcher.Watcher

	lock        sync.Mutex
	publishLock sync.RWMutex
	options     watcherOptions
	started     bool
	closed      bool
	handlers    []EventHandler
}

// New creates an new instance of folder watcher
func New(configs ...Option) (*folderWatcher, error) {
	options := watcherOptions{}
	for _, config := range configs {
		config(&options)
	}

	w := watcher.New()

	for _, path := range options.paths {
		if home, err := homedir.Dir(); err == nil {
			// If the root directory contains ~, we replace it with the actual home directory
			path = s.Replace(path, "~", home, -1)
		}

		if path == "" {
			log.Fatal(ErrFolderPathNotFound)
			return nil, ErrFolderPathNotFound
		}

		err := w.AddRecursive(path)
		if err != nil {
			return nil, err
		}
	}

	return &folderWatcher{
		w:       w,
		options: options,
	}, nil
}

func (fw *folderWatcher) RegisterHandler(handler EventHandler) {
	fw.publishLock.Lock()
	defer fw.publishLock.Unlock()
	fw.handlers = append(fw.handlers, handler)
}

func (fw *folderWatcher) AddFile(path string) error {
	if fsutils.IsPathDir(path) {
		return errors.New(fmt.Sprintf("unable to watch path %s folder is not supporter", path))
	}
	err := fw.w.Add(path)
	if err != nil {
		return err
	}

	return err
}

// Watch will start listening of changes on the folderWatcher path and trigger the handler with any update events
// This is a block operation
func (fw *folderWatcher) Watch(ctx context.Context) error {
	fw.setToStarted()

	go func() {
		for {
			select {
			case <-fw.w.Closed:
				log.Debug("Watcher graceful shutdown triggered")
				return
			case <-ctx.Done():
				fw.Close()
			case event, ok := <-fw.w.Event:
				if ok {
					if len(fw.handlers) == 0 {
						fw.publishEventToHandler(ctx, &defaultWatcherHandler{}, event)
					} else {
						fw.publishEvent(ctx, event)
					}
				}
			case err, ok := <-fw.w.Error:
				if !ok {
					return
				}
				log.Fatal(err)
			}
		}
	}()

	log.Debug("Starting watcher", fmt.Sprintf("filePath:%s", fw.options.paths))
	// This is blocking
	err := fw.w.Start(time.Millisecond * 100)
	fw.started = false
	if err != nil {
		return err
	}

	return nil
}

func (fw *folderWatcher) setToStarted() {
	fw.lock.Lock()
	defer fw.lock.Unlock()
	if fw.started {
		return
	}
	fw.started = true
}

func (fw *folderWatcher) publishEvent(ctx context.Context, event watcher.Event) {
	fw.publishLock.RLock()
	defer fw.publishLock.RUnlock()

	for _, handler := range fw.handlers {
		fw.publishEventToHandler(ctx, handler, event)
	}
}

func (fw *folderWatcher) publishEventToHandler(
	ctx context.Context,
	handler EventHandler,
	event watcher.Event,
) {
	if isBlacklisted(event.Path, event.FileInfo) {
		log.Debug("Skipping blacklisted file/folder event")
		return
	}

	switch event.Op {
	case watcher.Create:
		handler.OnCreate(ctx, event.Path, event.FileInfo)
	case watcher.Remove:
		handler.OnRemove(ctx, event.Path, event.FileInfo)
	case watcher.Write:
		handler.OnWrite(ctx, event.Path, event.FileInfo)
	case watcher.Rename:
		handler.OnRename(ctx, event.Path, event.FileInfo, event.OldPath)
	case watcher.Move:
		handler.OnMove(ctx, event.Path, event.FileInfo, event.OldPath)
	}
}

// Close will stop the watching operation and unblock watch calls
func (fw *folderWatcher) Close() {
	fw.lock.Lock()
	defer fw.lock.Unlock()

	if !fw.started || fw.closed {
		return
	}

	fw.closed = true
	fw.w.Close()
}

func (fw *folderWatcher) Shutdown() error {
	fw.Close()
	return nil
}
