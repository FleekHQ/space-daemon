package listener

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/FleekHQ/space-poc/core/textile/client"
	"github.com/FleekHQ/space-poc/core/textile/handler"
	"github.com/FleekHQ/space-poc/log"
	tc "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
)

type TextileThreadListener struct {
	bucketSlug         string
	textileClient      *client.TextileClient
	started            bool
	lock               sync.Mutex
	publishLock        sync.RWMutex
	waitForCloseSignal chan bool
	handlers           []handler.EventHandler
}

func New(textileClient *client.TextileClient, bucketSlug string) *TextileThreadListener {
	return &TextileThreadListener{
		bucketSlug:    bucketSlug,
		started:       false,
		textileClient: textileClient,
	}
}

// Starts listening from Textile Thread
func (tl *TextileThreadListener) Listen(ctx context.Context) error {
	var bucketCtx context.Context
	var dbID *thread.ID
	var err error

	if bucketCtx, dbID, err = tl.textileClient.GetBucketContext(tl.bucketSlug); err != nil {
		return err
	}

	var cancel context.CancelFunc
	bucketCtx, cancel = context.WithCancel(bucketCtx)
	defer cancel()
	opt := tc.ListenOption{}

	var threads *tc.Client

	if threads, err = tl.textileClient.GetThreadsConnection(); err != nil {
		return err
	}

	channel, err := threads.Listen(bucketCtx, *dbID, []tc.ListenOption{opt})

	tl.setToStarted()

	listenerEventHandler := func(val tc.ListenEvent) {
		log.Debug("received from channel!!!!")
		instance := &handler.Bucket{}
		if val.Err != nil {
			log.Error("error getting threads listener event", err)
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
			case val, ok := <-channel:
				if ok {
					listenerEventHandler(val)
				}
			}
		}
	}()

	log.Debug("Starting textile threads listener")
	// Block until we get close request
	<-tl.waitForCloseSignal
	return nil
}

func (tl *TextileThreadListener) publishEvent(bucketData *handler.Bucket, listenEvent *tc.ListenEvent) {
	tl.publishLock.RLock()
	defer tl.publishLock.RUnlock()

	for _, handler := range tl.handlers {
		tl.publishEventToHandler(handler, bucketData, listenEvent)
	}
}

func (tl *TextileThreadListener) publishEventToHandler(handler handler.EventHandler, bucketData *handler.Bucket, listenEvent *tc.ListenEvent) {
	switch listenEvent.Action.Type {
	case tc.ActionCreate:
		handler.OnCreate(bucketData, listenEvent)
	case tc.ActionDelete:
		handler.OnRemove(bucketData, listenEvent)
	case tc.ActionSave:
		handler.OnSave(bucketData, listenEvent)
	}
}

func (tl *TextileThreadListener) setToStarted() {
	tl.lock.Lock()
	defer tl.lock.Unlock()
	if tl.started {
		return
	}
	tl.started = true
	tl.waitForCloseSignal = make(chan bool, 1)
}

// Stops listening to Textile Thread
func (tl *TextileThreadListener) Close() {
	tl.lock.Lock()
	defer tl.lock.Unlock()

	if !tl.started {
		return
	}

	tl.waitForCloseSignal <- true
}

// Registers an handler.EventHandler that handles events in Textile
func (tl *TextileThreadListener) RegisterHandler(handler handler.EventHandler) {
	tl.publishLock.Lock()
	defer tl.publishLock.Unlock()
	tl.handlers = append(tl.handlers, handler)
}
