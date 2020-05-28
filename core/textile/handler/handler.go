package handler

import (
	"github.com/FleekHQ/space-poc/log"
	tc "github.com/textileio/go-threads/api/client"
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

// EventHandler
type EventHandler interface {
	OnCreate(bucketData *Bucket, listenEvent *tc.ListenEvent)
	OnRemove(bucketData *Bucket, listenEvent *tc.ListenEvent)
	OnSave(bucketData *Bucket, listenEvent *tc.ListenEvent)
}

// Implements EventHandler and defaults to logging actions performed
type DefaultListenerHandler struct{}

func (h *DefaultListenerHandler) OnCreate(bucketData *Bucket, listenEvent *tc.ListenEvent) {
	log.Info("Default Listener Handler: OnCreate")
}

func (h *DefaultListenerHandler) OnRemove(bucketData *Bucket, listenEvent *tc.ListenEvent) {
	log.Info("Default Listener Handler: OnRemove")
}

func (h *DefaultListenerHandler) OnSave(bucketData *Bucket, listenEvent *tc.ListenEvent) {
	log.Info("Default Listener Handler: OnSave")
}
