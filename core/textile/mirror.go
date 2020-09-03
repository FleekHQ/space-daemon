package textile

import (
	"context"
	"io"

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
		return nil, err
	}

	hubCtx, dbID, err := tc.getBucketContext(ctx, b.Slug(), bucket.RemoteDbID, true)
	if err != nil {
		return nil, err
	}

	return tc.hubb.PushPath(ctx, b.Key(), path, reader)
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
	_, err = tc.hubb.Init(hubCtx, bc.WithName(bucketSlug), bc.WithPrivate(true))
	if err != nil {
		return nil, err
	}

	// We store the bucket in a meta thread so that we can later fetch a list of all buckets
	log.Debug("Mirror Bucket " + bucketSlug + " created. Storing metadata.")
	mirrorSchema, err := tc.storeBucketMirrorSchema(hubCtx, schema)
	if err != nil {
		return nil, err
	}

	return mirrorSchema, nil
}
