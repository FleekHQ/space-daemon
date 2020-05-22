package spacefs

import (
	"context"
	"errors"
	"github.com/FleekHQ/space-poc/core/spacestore"
	"log"
	"testing"
)

//
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

	result, err := fs.LookupPath("/static")
	if err != nil {
		t.Fatal(err)
	}

	log.Printf("Path %s", result.Path())

	result, err = fs.LookupPath("/static/js/2.b4ef1316.chunk.js")
	if err != nil {
		t.Fatal(err)
	}
	log.Printf("Path %s", result.Path())
	attr, err := result.Attribute()
	if err != nil {
		t.Fatal(err)
	}
	log.Printf("Name %s", attr.Name())
	log.Printf("IsDir %v", attr.IsDir())
	log.Printf("Size %d", attr.Size())

	result, err = fs.LookupPath("/static/js")
	if err != nil {
		t.Fatal(err)
	}

	dirOps, ok := result.(DirOps)
	if !ok {
		t.Fatal(errors.New("result is not a DirOps"))
	}
	jsDirectory, err := dirOps.ReadDir()
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range jsDirectory {
		dirAttr, err := dir.Attribute()
		if err != nil {
			t.Fatal(err)
		}
		log.Printf("\nName: %s\nIs Dir: %v\nSize: %d\n", dirAttr.Name(), dirAttr.IsDir(), dirAttr.Size())
	}
}
