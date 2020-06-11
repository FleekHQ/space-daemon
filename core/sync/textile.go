package sync

import (
	"encoding/json"
	"github.com/FleekHQ/space-poc/core/textile"

	"github.com/FleekHQ/space-poc/core/events"
	"github.com/FleekHQ/space-poc/log"
	tc "github.com/textileio/go-threads/api/client"
)



func (h *textileHandler) OnCreate(bucketData *textile.BucketData, listenEvent *tc.ListenEvent) {
	log.Info("Default Listener Handler: OnCreate")
	instance := &textile.BucketData{}
	if err := json.Unmarshal(listenEvent.Action.Instance, instance); err != nil {
		log.Error("failed to unmarshal listen result: %v", err)
	}
	evt := events.NewTextileEvent(instance.Name)
	h.notifier.SendTextileEvent(evt)
}

func (h *textileHandler) OnRemove(bucketData *textile.BucketData, listenEvent *tc.ListenEvent) {
	log.Info("Default Listener Handler: OnRemove")
	instance := &textile.BucketData{}
	if err := json.Unmarshal(listenEvent.Action.Instance, instance); err != nil {
		log.Error("failed to unmarshal listen result: %v", err)
	}
	evt := events.NewTextileEvent(instance.Name)
	h.notifier.SendTextileEvent(evt)
}

func (h *textileHandler) OnSave(bucketData *textile.BucketData, listenEvent *tc.ListenEvent) {
	log.Info("Default Listener Handler: OnSave")
	instance := &textile.BucketData{}
	if err := json.Unmarshal(listenEvent.Action.Instance, instance); err != nil {
		log.Error("failed to unmarshal listen result: %v", err)
	}
	evt := events.NewTextileEvent(instance.Name)
	h.notifier.SendTextileEvent(evt)
}
