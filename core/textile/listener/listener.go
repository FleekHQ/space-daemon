package listener

import (
	"context"
	"encoding/json"
	"sync"

	textile "github.com/FleekHQ/space-poc/core/textile/client"
	"github.com/FleekHQ/space-poc/core/textile/handler"
	"github.com/FleekHQ/space-poc/log"
	threadsc "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
)

type ThreadListener interface {
	Listen(ctx context.Context) error
	Close()
	RegisterHandler(handler handler.EventHandler)
}

type textileThreadListener struct {
	bucketSlug         string
	textileClient      textile.Client
	started            bool
	lock               sync.Mutex
	publishLock        sync.RWMutex
	waitForCloseSignal chan bool
	handlers           []handler.EventHandler
}

func New(textileClient textile.Client, bucketSlug string, handlers []handler.EventHandler) ThreadListener {
	return &textileThreadListener{
		bucketSlug:    bucketSlug,
		started:       false,
		textileClient: textileClient,
		handlers:      handlers,
	}
}

// Starts listening from Textile Thread
func (tl *textileThreadListener) Listen(ctx context.Context) error {
	var bucketCtx context.Context
	var dbID *thread.ID
	var err error

	if bucketCtx, dbID, err = tl.textileClient.GetBucketContext(ctx, tl.bucketSlug); err != nil {
		return err
	}

	var cancel context.CancelFunc
	bucketCtx, cancel = context.WithCancel(bucketCtx)
	defer cancel()
	opt := threadsc.ListenOption{}

	var threads *threadsc.Client

	if threads, err = tl.textileClient.GetThreadsConnection(); err != nil {
		return err
	}

	channel, err := threads.Listen(bucketCtx, *dbID, []threadsc.ListenOption{opt})

	tl.setToStarted()

	listenerEventHandler := func(val threadsc.ListenEvent) {
		log.Debug("received from channel!!!!")
		instance := &handler.Bucket{}
		if val.Err != nil {
			log.Error("error getting threadsc listener event", err)
			return
		}
		if err = json.Unmarshal(val.Action.Instance, instance); err != nil {
			log.Error("failed to unmarshal listen result", err)
			return
		}

		if len(tl.handlers) == 0 {
			tl.publishEventToHandler(&handler.DefaultListenerHandler{}, instance, &val)
		} else {
			tl.publishEvent(instance, &val)
		}
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				tl.Close()
				return
			case val, ok := <-channel:
				if ok {
					listenerEventHandler(val)
				}
			}
		}
	}()

	log.Debug("Starting textile threadsc listener")
	// Block until we get close request
	<-tl.waitForCloseSignal
	log.Debug("Textile threadsc listener closed")
	return nil
}

func (tl *textileThreadListener) publishEvent(bucketData *handler.Bucket, listenEvent *threadsc.ListenEvent) {
	tl.publishLock.RLock()
	defer tl.publishLock.RUnlock()

	for _, handler := range tl.handlers {
		tl.publishEventToHandler(handler, bucketData, listenEvent)
	}
}

func (tl *textileThreadListener) publishEventToHandler(handler handler.EventHandler, bucketData *handler.Bucket, listenEvent *threadsc.ListenEvent) {
	switch listenEvent.Action.Type {
	case threadsc.ActionCreate:
		handler.OnCreate(bucketData, listenEvent)
	case threadsc.ActionDelete:
		handler.OnRemove(bucketData, listenEvent)
	case threadsc.ActionSave:
		handler.OnSave(bucketData, listenEvent)
	}
}

func (tl *textileThreadListener) setToStarted() {
	tl.lock.Lock()
	defer tl.lock.Unlock()
	if tl.started {
		return
	}
	tl.started = true
	tl.waitForCloseSignal = make(chan bool, 1)
}

// Stops listening to Textile Thread
func (tl *textileThreadListener) Close() {
	tl.lock.Lock()
	defer tl.lock.Unlock()

	if !tl.started {
		return
	}

	tl.waitForCloseSignal <- true
}

// Registers an handler.EventHandler that handles events in Textile
func (tl *textileThreadListener) RegisterHandler(handler handler.EventHandler) {
	tl.publishLock.Lock()
	defer tl.publishLock.Unlock()
	tl.handlers = append(tl.handlers, handler)
}
