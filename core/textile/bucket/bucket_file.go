package bucket

import (
	"context"
	"io"
	"regexp"
	"time"

	"github.com/opentracing/opentracing-go"

	"github.com/FleekHQ/space-daemon/log"
	"github.com/ipfs/interface-go-ipfs-core/path"
)

func (b *Bucket) FileExists(ctx context.Context, pth string) (bool, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	ctx, _, err := b.GetContext(ctx)
	if err != nil {
		return false, err
	}

	listPathRes, err := b.bucketsClient.ListPath(ctx, b.GetData().Key, pth)
	if err != nil {
		return false, err
	}

	ctxWithDeadline, ctxCancel := context.WithDeadline(ctx, time.Now().Add(3*time.Second))
	defer ctxCancel()

	// Call ListIpfsPath with deadline to avoid waiting too much for DHT to resolve
	_, err = b.bucketsClient.ListIpfsPath(ctxWithDeadline, path.New(listPathRes.Item.Cid))
	if err != nil {
		match, _ := regexp.MatchString(".*no link named.*under.*", err.Error())
		if match {
			return false, nil
		}
		// Since a nil would be interpreted as a false
		return false, err
	}

	return true, nil
}

func (b *Bucket) UpdatedAt(ctx context.Context, pth string) (int64, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	ctx, _, err := b.GetContext(ctx)
	if err != nil {
		return 0, err
	}

	response, err := b.bucketsClient.ListPath(ctx, b.GetData().Key, pth)
	if err != nil {
		return 0, err
	}

	return response.Item.Metadata.UpdatedAt, nil
}

func (b *Bucket) UploadFile(
	ctx context.Context,
	path string,
	reader io.Reader,
) (result path.Resolved, root path.Path, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Bucket.UploadFile")
	defer span.Finish()

	b.lock.Lock()
	defer b.lock.Unlock()

	ctx, _, err = b.GetContext(ctx)
	if err != nil {
		return nil, nil, err
	}

	result, root, err = b.bucketsClient.PushPath(ctx, b.Key(), path, reader)
	if err != nil {
		return nil, nil, err
	}

	if b.notifier != nil {
		b.notifier.OnUploadFile(b.Slug(), path, result, root)
	}

	return result, root, nil
}

func (b *Bucket) DownloadFile(ctx context.Context, path string, reader io.Reader) (result path.Resolved, root path.Path, err error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	ctx, _, err = b.GetContext(ctx)
	if err != nil {
		return nil, nil, err
	}

	result, root, err = b.bucketsClient.PushPath(ctx, b.Key(), path, reader)
	if err != nil {
		return nil, nil, err
	}

	// no notification

	return result, root, nil
}

// GetFile pulls path from bucket writing it to writer if it's a file.
func (b *Bucket) GetFile(ctx context.Context, path string, w io.Writer) error {
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
