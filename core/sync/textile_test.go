package sync

import (
	"testing"

	"github.com/FleekHQ/space-daemon/core/textile-new/bucket"

	"github.com/FleekHQ/space-daemon/mocks"
	"github.com/stretchr/testify/mock"
	tc "github.com/textileio/go-threads/api/client"
)

func TestTextileHandler_OnCreate(t *testing.T) {
	n := new(mocks.TextileNotifier)
	th := &textileHandler{
		notifier: n,
	}

	b := []byte(`{"Key":"bafzbeid2zp544qy6ktwdlr5xxsmsioclbxj42dkbqckm35e6l5biqlo3tq","Name":"test-bucket-1"}`)

	buck := &bucket.BucketData{}
	action := tc.Action{
		Collection: "buckets",
		Type:       1,
		InstanceID: "dummy-id",
		Instance:   b,
	}
	evt := &tc.ListenEvent{
		Action: action,
	}

	n.On("SendTextileEvent", mock.Anything).Return()

	th.OnCreate(buck, evt)
	n.AssertExpectations(t)
}

func TestTextileHandler_OnRemove(t *testing.T) {
	n := new(mocks.TextileNotifier)
	th := &textileHandler{
		notifier: n,
	}

	b := []byte(`{"Key":"bafzbeid2zp544qy6ktwdlr5xxsmsioclbxj42dkbqckm35e6l5biqlo3tq","Name":"test-bucket-1"}`)

	buck := &bucket.BucketData{}
	action := tc.Action{
		Collection: "buckets",
		Type:       1,
		InstanceID: "dummy-id",
		Instance:   b,
	}
	evt := &tc.ListenEvent{
		Action: action,
	}

	n.On("SendTextileEvent", mock.Anything).Return()

	th.OnRemove(buck, evt)
	n.AssertExpectations(t)
}

func TestTextileHandler_OnSave(t *testing.T) {
	n := new(mocks.TextileNotifier)
	th := &textileHandler{
		notifier: n,
	}

	b := []byte(`{"Key":"bafzbeid2zp544qy6ktwdlr5xxsmsioclbxj42dkbqckm35e6l5biqlo3tq","Name":"test-bucket-1"}`)

	buck := &bucket.BucketData{}
	action := tc.Action{
		Collection: "buckets",
		Type:       1,
		InstanceID: "dummy-id",
		Instance:   b,
	}
	evt := &tc.ListenEvent{
		Action: action,
	}

	n.On("SendTextileEvent", mock.Anything).Return()

	th.OnSave(buck, evt)
	n.AssertExpectations(t)
}
