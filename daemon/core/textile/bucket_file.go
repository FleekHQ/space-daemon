package textile

import (
	"bytes"
	"context"
	"io"
	"regexp"

	"github.com/FleekHQ/space/core/ipfs"
	"github.com/FleekHQ/space/log"
	"github.com/ipfs/interface-go-ipfs-core/path"
)

func (b *bucket) FileExists(ctx context.Context, path string) (bool, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	ctx, _, err := b.GetContext(ctx)
	if err != nil {
		return false, nil
	}

	lp, err := b.bucketsClient.ListPath(ctx, b.Key(), path)
	if err != nil {
		match, _ := regexp.MatchString(".*no link named.*under.*", err.Error())
		if match {
			return false, nil
		}
		log.Info("error doing list path on non existent directoy: ", err.Error())
		// Since a nil would be interpreted as a false
		return false, err
	}

	var fsHash string
	if _, err := ipfs.GetFileHash(&bytes.Buffer{}); err != nil {
		log.Error("Unable to get filehash: ", err)
		return false, err
	}

	item := lp.GetItem()
	if item.Cid == fsHash {
		return true, nil
	}

	return false, nil
}

func (b *bucket) UploadFile(ctx context.Context, path string, reader io.Reader) (result path.Resolved, root path.Path, err error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	ctx, _, err = b.GetContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	return b.bucketsClient.PushPath(ctx, b.Key(), path, reader)
}

// GetFile pulls path from bucket writing it to writer if it's a file.
func (b *bucket) GetFile(ctx context.Context, path string, w io.Writer) error {
	b.lock.RLock()
	defer b.lock.RUnlock()

	ctx, _, err := b.GetContext(ctx)
	if err != nil {
		return err
	}
	if err := b.bucketsClient.PullPath(ctx, b.Key(), path, w); err != nil {
		log.Error("error in GetFile from textile client", err)
		return err
	}

	return nil
}
