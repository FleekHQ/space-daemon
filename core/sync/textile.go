package sync

import (
	"encoding/json"

	"github.com/FleekHQ/space-poc/core/events"
	"github.com/FleekHQ/space-poc/core/textile/handler"
	"github.com/FleekHQ/space-poc/log"
	tc "github.com/textileio/go-threads/api/client"
)



func (h *textileHandler) OnCreate(bucketData *handler.Bucket, listenEvent *tc.ListenEvent) {
	log.Info("Default Listener Handler: OnCreate")
	instance := &handler.Bucket{}
	if err := json.Unmarshal(listenEvent.Action.Instance, instance); err != nil {
		log.Error("failed to unmarshal listen result: %v", err)
	}
	evt := events.NewTextileEvent(instance.Name)
	h.notifier.SendTextileEvent(evt)
}

func (h *textileHandler) OnRemove(bucketData *handler.Bucket, listenEvent *tc.ListenEvent) {
	log.Info("Default Listener Handler: OnRemove")
	instance := &handler.Bucket{}
	if err := json.Unmarshal(listenEvent.Action.Instance, instance); err != nil {
		log.Error("failed to unmarshal listen result: %v", err)
	}
	evt := events.NewTextileEvent(instance.Name)
	h.notifier.SendTextileEvent(evt)
}

func (h *textileHandler) OnSave(bucketData *handler.Bucket, listenEvent *tc.ListenEvent) {
	log.Info("Default Listener Handler: OnSave")
	instance := &handler.Bucket{}
	if err := json.Unmarshal(listenEvent.Action.Instance, instance); err != nil {
		log.Error("failed to unmarshal listen result: %v", err)
	}
	evt := events.NewTextileEvent(instance.Name)
	h.notifier.SendTextileEvent(evt)
}
