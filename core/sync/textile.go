package sync

import (
	"encoding/json"

	"github.com/FleekHQ/space-daemon/core/textile/bucket"

	"github.com/FleekHQ/space-daemon/core/events"
	"github.com/FleekHQ/space-daemon/log"
	tc "github.com/textileio/go-threads/api/client"
)

func (h *textileHandler) OnCreate(bucketData *bucket.BucketData, listenEvent *tc.ListenEvent) {
	log.Info("Default Listener Handler: OnCreate")
	instance := &bucket.BucketData{}
	if err := json.Unmarshal(listenEvent.Action.Instance, instance); err != nil {
		log.Error("failed to unmarshal listen result: %v", err)
	}
	evt := events.NewTextileEvent(instance.Name)
	h.notifier.SendTextileEvent(evt)
}

func (h *textileHandler) OnRemove(bucketData *bucket.BucketData, listenEvent *tc.ListenEvent) {
	log.Info("Default Listener Handler: OnRemove")
	instance := &bucket.BucketData{}
	if err := json.Unmarshal(listenEvent.Action.Instance, instance); err != nil {
		log.Error("failed to unmarshal listen result: %v", err)
	}
	evt := events.NewTextileEvent(instance.Name)
	h.notifier.SendTextileEvent(evt)
}

func (h *textileHandler) OnSave(bucketData *bucket.BucketData, listenEvent *tc.ListenEvent) {
	log.Info("Default Listener Handler: OnSave")
	instance := &bucket.BucketData{}
	if err := json.Unmarshal(listenEvent.Action.Instance, instance); err != nil {
		log.Error("failed to unmarshal listen result: %v", err)
	}
	evt := events.NewTextileEvent(instance.Name)
	h.notifier.SendTextileEvent(evt)
}
