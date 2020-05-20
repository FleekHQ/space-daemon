package watcher

import (
	"github.com/FleekHQ/space-poc/logger"
	"github.com/fsnotify/fsnotify"
	"log"
	"sync"
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

func StartWatcher() {
	logger.Info("Starting watcher")
	watcher, err := New("/Users/perfect/Terminal/mirror-path", func(filename string) error {
		return nil
	})
	if err != nil {
		log.Fatal(err)
		return
	}

	watcher.Watch()
	logger.Info("Watcher started")
	<-watcher.done
	logger.Info("Watcher closed/done")
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
				logger.Info("graceful shutdown")
				close(fw.done)
				return
			case event, ok := <-fw.w.Events:
				if !ok {
					return
				}
				log.Printf("Event Object: %+v", event)
				if event.Op&fsnotify.Create == fsnotify.Create {
					log.Println("created file:", event.Name)

					if err := fw.onCreate(event.Name); err != nil {
						log.Printf("error when calling onCreate for %s", event.Name)
					}
				}
				if event.Op&fsnotify.Remove == fsnotify.Remove {
					log.Println("onRemove file:", event.Name)

					if err := fw.onCreate(event.Name); err != nil {
						log.Printf("error when calling onRemove for %s", event.Name)
					}
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("write file:", event.Name)

					if err := fw.onCreate(event.Name); err != nil {
						log.Printf("error when calling onWrite for %s", event.Name)
					}
				}
				if event.Op&fsnotify.Rename == fsnotify.Rename {
					log.Println("renaming file:", event.Name)

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