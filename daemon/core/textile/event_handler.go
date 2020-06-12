package textile

import (
	"github.com/FleekHQ/space/daemon/log"
	tc "github.com/textileio/go-threads/api/client"
)

// EventHandler
type EventHandler interface {
	OnCreate(bucketData *BucketData, listenEvent *tc.ListenEvent)
	OnRemove(bucketData *BucketData, listenEvent *tc.ListenEvent)
	OnSave(bucketData *BucketData, listenEvent *tc.ListenEvent)
}

// Implements EventHandler and defaults to logging actions performed
type defaultListenerHandler struct{}

func (h *defaultListenerHandler) OnCreate(bucketData *BucketData, listenEvent *tc.ListenEvent) {
	log.Info("Default Listener Handler: OnCreate")
}

func (h *defaultListenerHandler) OnRemove(bucketData *BucketData, listenEvent *tc.ListenEvent) {
	log.Info("Default Listener Handler: OnRemove")
}

func (h *defaultListenerHandler) OnSave(bucketData *BucketData, listenEvent *tc.ListenEvent) {
	log.Info("Default Listener Handler: OnSave")
}
