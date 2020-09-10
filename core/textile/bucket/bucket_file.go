package bucket

import (
	"bytes"
	"context"
	"io"
	"regexp"

	"github.com/FleekHQ/space-daemon/core/ipfs"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/ipfs/interface-go-ipfs-core/path"
)

func (b *Bucket) FileExists(ctx context.Context, path string) (bool, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	ctx, _, err := b.getContext(ctx)
	if err != nil {
		return false, err
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

func (b *Bucket) UploadFile(ctx context.Context, path string, reader io.Reader) (result path.Resolved, root path.Path, err error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	ctx, _, err = b.getContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	return b.bucketsClient.PushPath(ctx, b.Key(), path, reader)
}

// GetFile pulls path from bucket writing it to writer if it's a file.
func (b *Bucket) GetFile(ctx context.Context, path string, w io.Writer) error {
	b.lock.RLock()
	defer b.lock.RUnlock()

	ctx, _, err := b.getContext(ctx)
	if err != nil {
		return err
	}

	if err := b.bucketsClient.PullPath(ctx, b.Key(), path, w); err != nil {
		log.Error("error in GetFile from textile client", err)
		return err
	}

	return nil
}

func (b *Bucket) GetPathAccessRoles(ctx context.Context, path string) ([]string, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	ctx, _, err := b.getContext(ctx)
	if err != nil {
		return nil, err
	}

	// sbc := NewSecureBucketsClient(tc.hb, b)

	// rs, err := sbc.PullPathAccessRoles(ctx, file.BucketKey, path)
	// if err != nil {
	// 	return nil, err
	// }

	pubks := make([]string, 0)
	// for _, pubk := range rs {
	// 	pubks = append(pubks, pubk)
	// }

	return pubks, nil
}
