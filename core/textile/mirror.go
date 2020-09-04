package textile

import (
	"context"
	"io"
	"os"

	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/core/textile/model"
	"github.com/FleekHQ/space-daemon/core/textile/utils"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/textileio/go-threads/core/thread"
	bc "github.com/textileio/textile/api/buckets/client"
)

func (tc *textileClient) IsMirrorFile(ctx context.Context, path, bucketSlug string) bool {
	mirrorFile, _ := tc.getModel().FindMirrorFileByPathAndBucketSlug(ctx, path, bucketSlug)
	if mirrorFile != nil {
		return true
	}

	return false
}

func (tc *textileClient) MarkMirrorFileBackup(ctx context.Context, path, bucketSlug string) (*domain.MirrorFile, error) {
	mf := &domain.MirrorFile{
		Path:       path,
		BucketSlug: bucketSlug,
		Backup:     true,
		Shared:     false,
	}
	// TODO: upsert
	_, err := tc.getModel().CreateMirrorFile(ctx, mf)
	if err != nil {
		return nil, err
	}

	return mf, nil
}

func (tc *textileClient) UploadFileToHub(ctx context.Context, b Bucket, path string, reader io.Reader) (result path.Resolved, root path.Path, err error) {
	// XXX: locking?

	bucket, err := tc.getModel().FindBucket(ctx, b.Slug())
	if err != nil {
		return nil, nil, err
	}

	hubCtx, _, err := tc.getBucketContext(ctx, b.Slug(), bucket.RemoteDbID, true, bucket.EncryptionKey)
	if err != nil {
		return nil, nil, err
	}

	return tc.hb.PushPath(hubCtx, bucket.RemoteBucketKey, path, reader)
}

// Creates a mirror bucket.
func (tc *textileClient) createMirrorBucket(ctx context.Context, schema model.BucketSchema) (*model.MirrorBucketSchema, error) {
	bucketSlug := schema.Slug

	log.Debug("Creating a new mirror bucket with slug " + bucketSlug)
	dbID, err := tc.createMirrorThread(ctx)
	if err != nil {
		return nil, err
	}
	hubCtx, _, err := tc.getBucketContext(ctx, utils.CastDbIDToString(*dbID), schema.Slug, true, schema.EncryptionKey)
	if err != nil {
		return nil, err
	}

	// create mirror bucket
	b, err := tc.hb.Init(hubCtx, bc.WithName(bucketSlug), bc.WithPrivate(true))
	if err != nil {
		return nil, err
	}

	return &model.MirrorBucketSchema{
		RemoteDbID:      dbID.String(),
		RemoteBucketKey: b.Root.Key,
		HubAddr:         os.Getenv("TXL_HUB_TARGET"),
	}, nil
}

// Creates a remote hub thread for the mirror bucket
func (tc *textileClient) createMirrorThread(ctx context.Context) (*thread.ID, error) {
	log.Debug("createMirrorThread: Generating a new threadID ...")
	dbID := thread.NewIDV1(thread.Raw, 32)

	log.Debug("createMirrorThread: Creating Thread DB for bucket at db " + dbID.String())
	if err := tc.ht.NewDB(ctx, dbID); err != nil {
		return nil, err
	}
	log.Debug("createMirrorThread: Thread DB Created")
	return &dbID, nil
}
