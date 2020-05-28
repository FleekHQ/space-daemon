package handler

import (
	"github.com/FleekHQ/space-poc/log"
	tc "github.com/textileio/go-threads/api/client"
)

// EventHandler
type EventHandler interface {
	OnCreate(bucketData *Bucket, listenEvent *tc.ListenEvent)
	OnRemove(bucketData *Bucket, listenEvent *tc.ListenEvent)
	OnSave(bucketData *Bucket, listenEvent *tc.ListenEvent)
}

// Implements EventHandler and defaults to logging actions performed
type defaultListenerHandler struct{}

func (h *defaultListenerHandler) OnCreate(bucketData *Bucket, listenEvent *tc.ListenEvent) {
	log.Info("Default Listener Handler: OnCreate")
}

func (h *defaultListenerHandler) OnRemove(bucketData *Bucket, listenEvent *tc.ListenEvent) {
	log.Info("Default Listener Handler: OnRemove")
}

func (h *defaultListenerHandler) OnSave(bucketData *Bucket, listenEvent *tc.ListenEvent) {
	log.Info("Default Listener Handler: OnSave")
}
