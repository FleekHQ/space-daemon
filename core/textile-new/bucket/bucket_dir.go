package bucket

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/FleekHQ/space-daemon/log"
	"github.com/ipfs/interface-go-ipfs-core/path"
)

// Keep file is added to empty directories
var keepFileName = ".keep"

func (b *Bucket) DirExists(path string) (bool, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	_, err := b.ListDirectory(path)

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
func (b *Bucket) CreateDirectory(path string) (result path.Resolved, root path.Path, err error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	ctx, _ := b.getContext()
	// append .keep file to the end of the directory
	emptyDirPath := strings.TrimRight(path, "/") + "/" + keepFileName
	return b.bucketsClient.PushPath(ctx, b.Key(), emptyDirPath, &bytes.Buffer{})
}

// ListDirectory returns a list of items in a particular directory
func (b *Bucket) ListDirectory(path string) (*DirEntries, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	ctx, _ := b.getContext()

	result, err := b.bucketsClient.ListPath(ctx, b.Key(), path)
	return (*DirEntries)(result), err
}

// DeleteDirOrFile will delete file or directory at path
func (b *Bucket) DeleteDirOrFile(path string) (path.Resolved, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	ctx, _ := b.getContext()

	return b.bucketsClient.RemovePath(ctx, b.Key(), path)
}
