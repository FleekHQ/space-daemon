package bucket

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/FleekHQ/space-daemon/log"
	"github.com/ipfs/interface-go-ipfs-core/path"
)

// Keep file is added to empty directories
var keepFileName = ".keep"

func (b *Bucket) DirExists(ctx context.Context, path string) (bool, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	_, err := b.ListDirectory(ctx, path)

	log.Debug("returned from bucket call")

	if err != nil {
		// NOTE: not sure if this is the best approach but didnt
		// want to loop over items each time
		match, _ := regexp.MatchString(".*no link named.*under.*", err.Error())
		if match {
			return false, nil
		}
		log.Info("error doing list path on non existent directoy: ", err.Error())
		// Since a nil would be interpreted as a false
		return false, err
	}
	return true, nil
}

// CreateDirectory creates an empty directory
// Because textile doesn't support empty directory an empty .keep file is created
// in the directory
func (b *Bucket) CreateDirectory(ctx context.Context, path string) (result path.Resolved, root path.Path, err error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	ctx, _, err = b.getContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	// append .keep file to the end of the directory
	emptyDirPath := strings.TrimRight(path, "/") + "/" + keepFileName
	log.Debug("Creating directory", "path:"+emptyDirPath)
	return b.bucketsClient.PushPath(ctx, b.Key(), emptyDirPath, &bytes.Buffer{})
}

// ListDirectory returns a list of items in a particular directory
func (b *Bucket) ListDirectory(ctx context.Context, path string) (*DirEntries, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	ctx, _, err := b.getContext(ctx)
	if err != nil {
		return nil, err
	}

	result, err := b.bucketsClient.ListPath(ctx, b.Key(), path)

	for _, item := range result.Item.Items {
		log.Debug(
			"List directory result",
			"path:"+path,
			fmt.Sprintf("Name:%v", item.Name),
			fmt.Sprintf("GetIsDir:%v", item.GetIsDir()),
			fmt.Sprintf("IsDir:%v", item.IsDir),
		)
	}

	if err != nil {
		log.Error("Error listing directory", err)
	}
	return (*DirEntries)(result), err
}

// DeleteDirOrFile will delete file or directory at path
func (b *Bucket) DeleteDirOrFile(ctx context.Context, path string) (path.Resolved, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	ctx, _, err := b.getContext(ctx)
	if err != nil {
		return nil, err
	}

	return b.bucketsClient.RemovePath(ctx, b.Key(), path)
}
