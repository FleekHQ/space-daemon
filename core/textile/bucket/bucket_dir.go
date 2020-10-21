package bucket

import (
	"bytes"
	"context"
	"regexp"
	"strings"

	"github.com/FleekHQ/space-daemon/core/textile/utils"
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
	ctx, _, err = b.GetContext(ctx)
	if err != nil {
		return nil, nil, err
	}

	// append .keep file to the end of the directory
	emptyDirPath := strings.TrimRight(path, "/") + "/" + keepFileName
	return b.bucketsClient.PushPath(ctx, b.Key(), emptyDirPath, &bytes.Buffer{})
}

// ListDirectory returns a list of items in a particular directory
func (b *Bucket) ListDirectory(ctx context.Context, path string) (*DirEntries, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	ctx, _, err := b.GetContext(ctx)
	if err != nil {
		return nil, err
	}

	result, err := b.bucketsClient.ListPath(ctx, b.Key(), path)
	if err != nil {
		return nil, err
	}

	return (*DirEntries)(result), err
}

// DeleteDirOrFile will delete file or directory at path
func (b *Bucket) DeleteDirOrFile(ctx context.Context, path string) (path.Resolved, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	ctx, _, err := b.GetContext(ctx)
	if err != nil {
		return nil, err
	}

	return b.bucketsClient.RemovePath(ctx, b.Key(), path)
}

// return the recursive items count for a path
func (b *Bucket) ItemsCount(ctx context.Context, path string, withRecursive bool) (int32, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	var count int32

	dir, err := b.ListDirectory(ctx, path)
	if err != nil {
		return 0, err
	}

	count = dir.Item.ItemsCount

	if withRecursive {
		for _, item := range dir.Item.Items {
			if utils.IsMetaFileName(item.Name) {
				continue
			}

			if item.IsDir {
				n, err := b.ItemsCount(ctx, item.Path, withRecursive)
				if err != nil {
					return 0, err
				}

				count += n
			}
		}
	}

	return count, nil
}

// iterate over the bucket
func (b *Bucket) Each(ctx context.Context, path string, iterator EachFunc, withRecursive bool) (int, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	var count int

	dir, err := b.ListDirectory(ctx, path)
	if err != nil {
		return 0, err
	}

	for _, item := range dir.Item.Items {
		if utils.IsMetaFileName(item.Name) {
			continue
		}

		var n int

		if withRecursive && item.IsDir {
			if n, err = b.Each(ctx, item.Path, iterator, withRecursive); err != nil {
				return 0, err
			}

			count += n
			continue
		}

		if err := iterator(ctx, b, item.Path); err != nil {
			return 0, err
		}

		count += n
	}

	return count, nil
}
