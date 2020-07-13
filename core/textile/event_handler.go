package textile

import (
	"github.com/FleekHQ/space-daemon/core/textile/bucket"
	"github.com/FleekHQ/space-daemon/log"
	tc "github.com/textileio/go-threads/api/client"
)

// EventHandler
type EventHandler interface {
	OnCreate(bucketData *bucket.BucketData, listenEvent *tc.ListenEvent)
	OnRemove(bucketData *bucket.BucketData, listenEvent *tc.ListenEvent)
	OnSave(bucketData *bucket.BucketData, listenEvent *tc.ListenEvent)
}

// Implements EventHandler and defaults to logging actions performed
type defaultListenerHandler struct{}

func (h *defaultListenerHandler) OnCreate(bucketData *bucket.BucketData, listenEvent *tc.ListenEvent) {
	log.Info("Default Listener Handler: OnCreate")
}

func (h *defaultListenerHandler) OnRemove(bucketData *bucket.BucketData, listenEvent *tc.ListenEvent) {
	log.Info("Default Listener Handler: OnRemove")
}

func (h *defaultListenerHandler) OnSave(bucketData *bucket.BucketData, listenEvent *tc.ListenEvent) {
	log.Info("Default Listener Handler: OnSave")
}
