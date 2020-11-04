package textile

import (
	"context"
	"errors"

	"github.com/FleekHQ/space-daemon/core/textile/utils"
	"github.com/textileio/go-threads/api/client"
	threadsClient "github.com/textileio/go-threads/api/client"
)

func (tc *textileClient) Listen(ctx context.Context, dbID, threadName string) (<-chan threadsClient.ListenEvent, error) {
	db, err := utils.ParseDbIDFromString(dbID)
	if err != nil {
		return nil, err
	}

	newCtx, err := utils.GetThreadContext(ctx, "", *db, true, tc.kc, tc.hubAuth, tc.ht)
	if err != nil {
		return nil, err
	}

	return tc.ht.Listen(newCtx, *db, nil)
}

func (tc *textileClient) addListener(ctx context.Context, bucketSlug string) error {
	if err := tc.requiresHubConnection(); err != nil {
		return err
	}
	handler := newRestorerListenerHandler(tc.sync, tc.store, tc.ipfsClient)
	handlers := []EventHandler{handler}
	listener := NewListener(tc, bucketSlug, handlers)
	tc.dbListeners[bucketSlug] = listener

	go func() {
		err := listener.Listen(ctx)
		if err != nil {
			// Remove element from map as it's not listening anymore
			delete(tc.dbListeners, bucketSlug)
		}
	}()

	return nil
}

func (tc *textileClient) DeleteListeners(ctx context.Context) {
	for k, _ := range tc.dbListeners {
		delete(tc.dbListeners, k)
	}
}

func (tc *textileClient) initializeListeners(ctx context.Context) error {
	if err := tc.requiresHubConnection(); err != nil {
		return err
	}

	tc.closeListeners()

	buckets, err := tc.listBuckets(ctx)
	if err != nil {
		return err
	}

	for _, bucket := range buckets {
		tc.addListener(ctx, bucket.Slug())
	}

	return nil
}

func (tc *textileClient) closeListeners() {
	for key, listener := range tc.dbListeners {
		listener.Close()
		delete(tc.dbListeners, key)
	}

}

type listener struct {
	client     Client
	bucketSlug string
	handlers   []EventHandler
	shutdown   chan (bool)
}

func NewListener(client Client, bucketSlug string, handlers []EventHandler) *listener {
	return &listener{
		client:     client,
		bucketSlug: bucketSlug,
		handlers:   handlers,
		shutdown:   make(chan (bool)),
	}
}

func (l *listener) Listen(ctx context.Context) error {
	bucketSchema, err := l.client.GetModel().FindBucket(ctx, l.bucketSlug)
	if bucketSchema == nil || bucketSchema.RemoteDbID == "" {
		return errors.New("Bucket does not have a linked mirror bucket")
	}

	bucket, err := l.client.GetBucket(ctx, l.bucketSlug, nil)
	if err != nil {
		return err
	}
	bucketData := bucket.GetData()

	if err != nil {
		return err
	}

	eventChan, err := l.client.Listen(ctx, bucketSchema.RemoteDbID, bucketSchema.RemoteBucketSlug)
	if err != nil {
		return err
	}

Loop:
	for {
		select {
		case ev := <-eventChan:
			if ev.Err != nil {
				return ev.Err
			}

			for _, handler := range l.handlers {
				switch ev.Action.Type {
				case client.ActionCreate:
					handler.OnCreate(&bucketData, &ev)
				case client.ActionSave:
					handler.OnSave(&bucketData, &ev)
				case client.ActionDelete:
					handler.OnRemove(&bucketData, &ev)
				}
			}
		case <-l.shutdown:
			break Loop
		}
	}

	return nil
}

func (l *listener) Close() {
	l.shutdown <- true
}
