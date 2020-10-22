package textile

import (
	"context"

	"github.com/textileio/go-threads/api/client"
)

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

	bucket, err := l.client.GetBucket(ctx, l.bucketSlug, nil)
	if err != nil {
		return err
	}
	bucketData := bucket.GetData()

	if err != nil {
		return err
	}

	eventChan, err := l.client.Listen(ctx, bucketSchema.RemoteDbID)
	if err != nil {
		return err
	}

Loop:
	for {
		select {
		case ev := <-eventChan:
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
