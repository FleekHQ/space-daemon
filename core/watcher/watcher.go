package watcher

import (
	"context"
	"errors"
	"fmt"
	s "strings"
	"sync"

	"os"
	"time"

	homedir "github.com/mitchellh/go-homedir"

	"github.com/radovskyb/watcher"

	"github.com/FleekHQ/space-poc/log"
)

var (
	ErrFolderPathNotFound = errors.New("could not find a folder path for watcher")
)

type FolderWatcher struct {
	w *watcher.Watcher

	lock        sync.Mutex
	publishLock sync.RWMutex
	options     watcherOptions
	started     bool
	closed      bool
	handlers    []EventHandler
}

// New creates an new instance of folder watcher
// It listens to the path specified in the config space/folderPath
func New(configs ...Option) (*FolderWatcher, error) {
	options := watcherOptions{}
	for _, config := range configs {
		config(&options)
	}

	if len(options.paths) == 0 {
		// default watches current directory
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		options.paths = append(options.paths, cwd)
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

	return &FolderWatcher{
		w:       w,
		options: options,
	}, nil
}

func (fw *FolderWatcher) RegisterHandler(handler EventHandler) {
	fw.publishLock.Lock()
	defer fw.publishLock.Unlock()
	fw.handlers = append(fw.handlers, handler)
}

// Watch will start listening of changes on the FolderWatcher path and trigger the handler with any update events
// This is a block operation
func (fw *FolderWatcher) Watch(ctx context.Context) error {
	fw.setToStarted()

	go func() {
		for {
			select {
			case <-fw.w.Closed:
				log.Info("Watcher graceful shutdown triggered")
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

	log.Info("Starting watcher", fmt.Sprintf("filePath:%s", fw.options.paths))
	// This is blocking
	err := fw.w.Start(time.Millisecond * 100)
	fw.started = false
	if err != nil {
		return err
	}

	return nil
}

func (fw *FolderWatcher) setToStarted() {
	fw.lock.Lock()
	defer fw.lock.Unlock()
	if fw.started {
		return
	}
	fw.started = true
}

func (fw *FolderWatcher) publishEvent(ctx context.Context, event watcher.Event) {
	fw.publishLock.RLock()
	defer fw.publishLock.RUnlock()

	for _, handler := range fw.handlers {
		fw.publishEventToHandler(ctx, handler, event)
	}
}

func (fw *FolderWatcher) publishEventToHandler(
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
func (fw *FolderWatcher) Close() {
	fw.lock.Lock()
	defer fw.lock.Unlock()

	if !fw.started || fw.closed {
		return
	}

	fw.closed = true
	fw.w.Close()
}
