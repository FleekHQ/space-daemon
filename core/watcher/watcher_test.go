package watcher

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	w "github.com/radovskyb/watcher"
)

// TODO: Use mockery to create mock interface implementations
type handlerMock struct {
	mock.Mock
}

func (h *handlerMock) OnCreate(path string, fileInfo os.FileInfo) {
	h.Called(path, fileInfo)
}

func (h *handlerMock) OnRemove(path string, fileInfo os.FileInfo) {
	h.Called(path, fileInfo)
}

func (h *handlerMock) OnWrite(path string, fileInfo os.FileInfo) {
	h.Called(path, fileInfo)
}

func (h *handlerMock) OnRename(path string, fileInfo os.FileInfo, oldPath string) {
	h.Called(path, fileInfo, oldPath)
}

func (h *handlerMock) OnMove(path string, fileInfo os.FileInfo, oldPath string) {
	h.Called(path, fileInfo, oldPath)
}

func isTriggeredEvent(info os.FileInfo) bool {
	return info.Name() == "triggered event"
}

func startWatcher(t *testing.T, watchPaths ...string) (context.Context, *FolderWatcher, error) {
	ctx := context.Background()
	watcher, err := New(WithPaths(watchPaths...))
	if err != nil {
		return nil, nil, err
	}

	// execute
	go func() {
		err = watcher.Watch(ctx)
		if err != nil {
			t.Fatal(err)
		}
	}()

	return ctx, watcher, nil
}

func TestFolderWatcher_Watch_Triggers_Handler_OnCreate(t *testing.T) {
	// setup
	_, watcher, err := startWatcher(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		watcher.Close()
	})

	handler := new(handlerMock)
	handler.On("OnCreate", "-", mock.MatchedBy(isTriggeredEvent)).Return()
	watcher.RegisterHandler(handler)

	// trigger event
	watcher.w.TriggerEvent(w.Create, nil)

	// wait a few ms for async event to trigger handler
	<-time.After(time.Millisecond * 100)

	// assert
	handler.AssertNumberOfCalls(t, "OnCreate", 1)
	handler.AssertExpectations(t)
}

func TestFolderWatcher_Watch_Triggers_Handler_OnRemove(t *testing.T) {
	// setup
	_, watcher, err := startWatcher(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		watcher.Close()
	})

	handler := new(handlerMock)
	handler.On("OnRemove", "-", mock.MatchedBy(isTriggeredEvent)).Return()
	watcher.RegisterHandler(handler)

	// trigger event
	watcher.w.TriggerEvent(w.Remove, nil)

	// wait a few ms for async event to trigger handler
	<-time.After(time.Millisecond * 100)

	// assert
	handler.AssertNumberOfCalls(t, "OnRemove", 1)
	handler.AssertExpectations(t)
}
