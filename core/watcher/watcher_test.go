package watcher

import (
	"context"
	"fmt"
	"os"
	"testing"

	internalWatcher "github.com/radovskyb/watcher"
)

func TestFolderWatcher_Watch_Should_Trigger_Handler(t *testing.T) {
	ctx := context.Background()
	watcher, err := New()
	if err != nil {
		t.Fatal(err)
		return
	}

	// execute
	go func() {
		err = watcher.Watch(ctx, func(e UpdateEvent, fileInfo os.FileInfo, newPath, oldPath string) {
			if e != Remove {
				t.Fatal(fmt.Errorf("watcher not triggered with 'Remove' event instead got '%s'", e.String()))
			}
		})
		if err != nil {
			t.Fatal(err)
		}
	}()
	// note: using private w to trigger handler for testing purposes
	watcher.w.TriggerEvent(internalWatcher.Remove, nil)

	// cleanup
	watcher.Close()
}
