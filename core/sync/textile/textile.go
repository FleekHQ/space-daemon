package textile

import (
	"encoding/json"

	"github.com/FleekHQ/space-poc/core/events"
	"github.com/FleekHQ/space-poc/core/textile/handler"
	"github.com/FleekHQ/space-poc/log"
	tc "github.com/textileio/go-threads/api/client"
)

type TextileNotifier interface {
	SendTextileEvent(event events.TextileEvent)
}

// Implementation to handle events from textile
type Handler struct {
	notifier TextileNotifier
}

// Creates a New Textile Handler // TODO: define what is needed from handler like pushNotification func etc
func NewHandler(notifier TextileNotifier) *Handler {
	return &Handler{
		notifier: notifier,
	}
}

func (h *Handler) OnCreate(bucketData *handler.Bucket, listenEvent *tc.ListenEvent) {
	log.Info("Default Listener Handler: OnCreate")
	instance := &handler.Bucket{}
	if err := json.Unmarshal(listenEvent.Action.Instance, instance); err != nil {
		log.Error("failed to unmarshal listen result: %v", err)
	}
	evt := events.NewTextileEvent(instance.Name)
	h.notifier.SendTextileEvent(evt)
}

func (h *Handler) OnRemove(bucketData *handler.Bucket, listenEvent *tc.ListenEvent) {
	log.Info("Default Listener Handler: OnRemove")
	instance := &handler.Bucket{}
	if err := json.Unmarshal(listenEvent.Action.Instance, instance); err != nil {
		log.Error("failed to unmarshal listen result: %v", err)
	}
	evt := events.NewTextileEvent(instance.Name)
	h.notifier.SendTextileEvent(evt)
}

func (h *Handler) OnSave(bucketData *handler.Bucket, listenEvent *tc.ListenEvent) {
	log.Info("Default Listener Handler: OnSave")
	instance := &handler.Bucket{}
	if err := json.Unmarshal(listenEvent.Action.Instance, instance); err != nil {
		log.Error("failed to unmarshal listen result: %v", err)
	}
	evt := events.NewTextileEvent(instance.Name)
	h.notifier.SendTextileEvent(evt)
}
