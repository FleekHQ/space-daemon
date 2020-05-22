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

	cfg "github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/log"
)

var (
	ErrFolderPathNotFound = errors.New("could not find a folder path for watcher")
)

type UpdateEvent watcher.Op

const (
	Create = UpdateEvent(watcher.Create)
	Write  = UpdateEvent(watcher.Write)
	Rename = UpdateEvent(watcher.Rename)
	Remove = UpdateEvent(watcher.Remove)
	Chmod  = UpdateEvent(watcher.Chmod)
	Move   = UpdateEvent(watcher.Move)
)

type Handler func(event UpdateEvent, fileInfo os.FileInfo, newPath, oldPath string)

func (e UpdateEvent) String() string {
	return watcher.Op(e).String()
}

type FolderWatcher struct {
	w *watcher.Watcher

	stopWatch chan struct{}
	done      chan struct{}

	lock      sync.Mutex
	watchPath string
	started   bool
	closed    bool
}

// New creates an new instance of folder watcher
// It listens to the path specified in the config space/folderPath
func New(config cfg.Config) (*FolderWatcher, error) {
	path := config.GetString(cfg.SpaceFolderPath, "")
	w := watcher.New()

	if home, err := homedir.Dir(); err == nil {
		// If the root directory contains ~, we replace it with the actual home directory
		path = s.Replace(path, "~", home, -1)
	}

	if path == "" {
		log.Fatal(ErrFolderPathNotFound)
		return nil, ErrFolderPathNotFound
	}

	log.Info("Starting watcher in filePath", fmt.Sprintf("filePath:%s", path))
	err := w.AddRecursive(path)
	if err != nil {
		return nil, err
	}

	return &FolderWatcher{
		w:         w,
		watchPath: path,
		stopWatch: make(chan struct{}),
	}, nil
}

// Watch will start listening of changes on the FolderWatcher path and trigger the handler with any update event
// This is a block operation
func (fw *FolderWatcher) Watch(ctx context.Context, handler Handler) error {
	fw.lock.Lock()
	if fw.started {
		return nil
	}
	fw.started = true
	fw.lock.Unlock()

	go func() {
		for {
			select {
			case <-fw.stopWatch:
				log.Info("graceful shutdown")
				return
			case <-fw.w.Closed:
				fw.Close()
			case <-ctx.Done():
				fw.Close()
			case event, ok := <-fw.w.Event:
				if ok {
					handler(
						UpdateEvent(event.Op),
						event,
						event.Path,
						event.OldPath,
					)
				}
			case err, ok := <-fw.w.Error:
				if !ok {
					return
				}
				log.Fatal(err)
			}
		}
	}()

	// This is blocking
	err := fw.w.Start(time.Millisecond * 100)
	fw.started = false
	if err != nil {
		return err
	}

	return nil
}

// Close will stop the watching operation and unblock watch calls
func (fw *FolderWatcher) Close() {
	log.Info("Closing connection")
	fw.lock.Lock()
	defer fw.lock.Unlock()
	if !fw.started || fw.closed {
		return
	}
	fw.closed = true
	close(fw.stopWatch)
	fw.w.Close()
}
