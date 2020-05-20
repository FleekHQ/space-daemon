package spacefs

import (
	"context"
	"github.com/FleekHQ/space-poc/examples/fleek-fs-sync/spacestore"
	"testing"
)

func TestSpaceFS_LookupPath(t *testing.T) {
	ctx := context.Background()
	memStore, err := spacestore.NewMemoryStore(ctx)
	if err != nil {
		t.Fatal(err)
	}

	fs, err := NewSpaceFS(ctx, memStore)
	if err != nil {
		t.Fatal(err)
	}

	_, err = fs.LookupPath("/home")
	if err != nil {
		t.Fatal(err)
	}
}