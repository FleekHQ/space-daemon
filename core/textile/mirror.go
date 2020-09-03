package textile

import (
	"context"
	"io"
	"os"

	"github.com/FleekHQ/space-daemon/log"
	"github.com/ipfs/interface-go-ipfs-core/path"
	bc "github.com/textileio/textile/api/buckets/client"
)

type BucketMirrorSchema struct {
	RemoteDbID string `json:"remoteDbId"`
	HubAddr    string `json:"HubAddr"`
}

type MirrorFile struct {
	Path       string
	BucketSlug string
	Backup     bool
	Shared     bool
}

func (tc *textileClient) IsMirrorFile(ctx context.Context, path, bucketSlug string) bool {
	mirrorFile, _ := tc.findMirrorFileByPathAndBucketSlug(ctx, path, bucketSlug)
	if mirrorFile != nil {
		return true
	}

	return false
}

func (tc *textileClient) MarkMirrorFileBackup(ctx context.Context, path, bucketSlug string) (*MirrorFile, error) {
	mf := &MirrorFile{
		Path:       path,
		BucketSlug: bucketSlug,
		Backup:     true,
		Shared:     false,
	}
	// TODO: upsert
	_, err := tc.createMirrorFile(ctx, mf)
	if err != nil {
		return nil, err
	}

	return mf, nil
}

func (tc *textileClient) UploadFileToHub(ctx context.Context, b Bucket, path string, reader io.Reader) (result path.Resolved, root path.Path, err error) {
	// XXX: locking?

	bucket, err := tc.FindBucketInCollection(ctx, b.Slug())
	if err != nil {
		return nil, nil, err
	}

	hubCtx, _, err := tc.getBucketContext(ctx, b.Slug(), bucket.RemoteDbID, true)
	if err != nil {
		return nil, nil, err
	}

	return tc.hb.PushPath(hubCtx, b.Key(), path, reader)
}

// Creates a mirror bucket.
func (tc *textileClient) createMirrorBucket(ctx context.Context, schema BucketSchema) (*BucketMirrorSchema, error) {
	bucketSlug := schema.Slug

	log.Debug("Creating a new mirror bucket with slug " + bucketSlug)
	hubCtx, dbID, err := tc.getBucketContext(ctx, bucketSlug, schema.DbID, true)
	if err != nil {
		return nil, err
	}

	// create mirror bucket
	_, err = tc.hb.Init(hubCtx, bc.WithName(bucketSlug), bc.WithPrivate(true))
	if err != nil {
		return nil, err
	}

	return &BucketMirrorSchema{
		RemoteDbID: dbID.String(),
		HubAddr:    os.Getenv("TXL_HUB_TARGET"),
	}, nil
}
