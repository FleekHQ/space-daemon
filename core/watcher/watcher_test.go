package watcher

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	w "github.com/radovskyb/watcher"
)

// TODO: Use mockery to create mocks interface implementations
type handlerMock struct {
	mock.Mock
}

func (h *handlerMock) OnCreate(ctx context.Context, path string, fileInfo os.FileInfo) {
	h.Called(ctx, path, fileInfo)
}

func (h *handlerMock) OnRemove(ctx context.Context, path string, fileInfo os.FileInfo) {
	h.Called(ctx, path, fileInfo)
}

func (h *handlerMock) OnWrite(ctx context.Context, path string, fileInfo os.FileInfo) {
	h.Called(ctx, path, fileInfo)
}

func (h *handlerMock) OnRename(ctx context.Context, path string, fileInfo os.FileInfo, oldPath string) {
	h.Called(ctx, path, fileInfo, oldPath)
}

func (h *handlerMock) OnMove(ctx context.Context, path string, fileInfo os.FileInfo, oldPath string) {
	h.Called(ctx, path, fileInfo, oldPath)
}

func isTriggeredEvent(info os.FileInfo) bool {
	return info.Name() == "triggered event"
}

func startWatcher(t *testing.T, watchPaths ...string) (context.Context, FolderWatcher, error) {
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

	handler := new(handlerMock)
	handler.On("OnCreate", mock.Anything, "-", mock.MatchedBy(isTriggeredEvent)).Return()
	watcher.RegisterHandler(handler)

	// trigger event
	watcher.(*folderWatcher).w.TriggerEvent(w.Create, nil)

	// wait a few ms for async event to trigger handler
	<-time.After(time.Millisecond * 100)

	// assert
	handler.AssertNumberOfCalls(t, "OnCreate", 1)
	handler.AssertExpectations(t)

	// cleanup
	watcher.Close()
}

func TestFolderWatcher_Watch_Triggers_Handler_OnRemove(t *testing.T) {
	// setup
	_, watcher, err := startWatcher(t)
	if err != nil {
		t.Fatal(err)
	}

	handler := new(handlerMock)
	handler.On("OnRemove", mock.Anything, "-", mock.MatchedBy(isTriggeredEvent)).Return()
	watcher.RegisterHandler(handler)

	// trigger event
	watcher.(*folderWatcher).w.TriggerEvent(w.Remove, nil)

	// wait a few ms for async event to trigger handler
	<-time.After(time.Millisecond * 100)

	// assert
	handler.AssertNumberOfCalls(t, "OnRemove", 1)
	handler.AssertExpectations(t)

	// cleanup
	watcher.Close()
}
