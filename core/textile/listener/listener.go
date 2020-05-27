package listener

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/FleekHQ/space-poc/core/textile/client"
	"github.com/FleekHQ/space-poc/log"
	tc "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
)

type Bucket struct {
	Key       string `json:"_id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	DNSRecord string `json:"dns_record,omitempty"`
	//Archives  Archives `json:"archives"`
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

type TextileThreadListener struct {
	bucketSlug         string
	textileClient      *client.TextileClient
	started            bool
	lock               sync.Mutex
	publishLock        sync.RWMutex
	waitForCloseSignal chan bool
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

	go func() {
		for {
			select {
			case <-ctx.Done():
				tl.Close()
			case val, ok := <-channel:
				if ok {
					log.Debug("received from channel!!!!")
					instance := &Bucket{}
					if err = json.Unmarshal(val.Action.Instance, instance); err != nil {
						log.Error("failed to unmarshal listen result: %v", err)
					}

					// if len(tl.handlers) == 0 {
					// 	tl.publishEventToHandler(&defaultWatcherHandler{}, event)
					// } else {
					// 	tl.publishEvent(event)
					// }
				}
			}
		}
	}()

	log.Info("Starting textile threads listener")
	// Block until we get close request
	<-tl.waitForCloseSignal
	return nil
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
