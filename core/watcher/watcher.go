package watcher

import (
	"context"
	"errors"
	"fmt"
	cfg "github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/log"
	"github.com/fsnotify/fsnotify"
	"sync"
)

var (
	ErrFolderPathNotFound = errors.New("could not find a folder path found for watcher")
)

type Handler func(fileName string) error

type FolderWatcher struct {
	w        *fsnotify.Watcher
	onCreate Handler

	stopWatch chan struct{}
	done      chan struct{}

	lock    sync.Mutex
	started bool
	closed  bool
}

func New(path string, onCreate Handler) (*FolderWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	if err = watcher.Add(path); err != nil {
		return nil, err
	}

	return &FolderWatcher{
		w:         watcher,
		onCreate:  onCreate,
		stopWatch: make(chan struct{}),
		done:      make(chan struct{}),
	}, nil
}

func (fw *FolderWatcher) Close() {
	fw.lock.Lock()
	defer fw.lock.Unlock()
	if !fw.started || fw.closed {

		return
	}
	fw.closed = true

	close(fw.stopWatch)
	<-fw.done
	fw.w.Close()
}


func Start(ctx context.Context, config cfg.Config) {
	path := config.GetString(cfg.SpaceFolderPath, "")

	if path == "" {
		log.Fatal(ErrFolderPathNotFound)
		panic(ErrFolderPathNotFound)
	}

	log.Info("Starting watcher in filePath", fmt.Sprintf("filePath:%s", path))
	watcher, err := New(path, func(filename string) error {
		return nil
	})
	if err != nil {
		log.Fatal(err)
		return
	}

	watcher.Watch()
	log.Info("Watcher started")
	<-watcher.done
	log.Info("Watcher closed/done")
}

func (fw *FolderWatcher) Watch() {
	fw.lock.Lock()
	defer fw.lock.Unlock()
	if fw.started {
		return
	}

	fw.started = true
	go func() {
		for {
			select {
			case <-fw.stopWatch:
				log.Info("graceful shutdown")
				close(fw.done)
				return
			case event, ok := <-fw.w.Events:
				if !ok {
					return
				}
				log.Printf("Event Object: %+v", event)
				if event.Op&fsnotify.Create == fsnotify.Create {
					log.Info("created file:",  "eventName:" + event.Name)

					if err := fw.onCreate(event.Name); err != nil {
						log.Printf("error when calling onCreate for %s", event.Name)
					}
				}
				if event.Op&fsnotify.Remove == fsnotify.Remove {
					log.Info("onRemove file:",  "eventName:" + event.Name)

					if err := fw.onCreate(event.Name); err != nil {
						log.Printf("error when calling onRemove for %s", event.Name)
					}
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Info("write file:", "eventName:" + event.Name)

					if err := fw.onCreate(event.Name); err != nil {
						log.Printf("error when calling onWrite for %s", event.Name)
					}
				}
				if event.Op&fsnotify.Rename == fsnotify.Rename {
					log.Info("renaming file:", "eventName:" + event.Name)

					if err := fw.onCreate(event.Name); err != nil {
						log.Printf("error when calling OnRename for %s", event.Name)
					}
				}
			case err, ok := <-fw.w.Errors:
				if !ok {
					return
				}
				log.Fatal(err)
			}
		}
	}()
}
