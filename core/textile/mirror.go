package textile

import (
	"context"
	"io"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/core/textile/model"
	"github.com/FleekHQ/space-daemon/core/textile/utils"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/textileio/go-threads/core/thread"
	bc "github.com/textileio/textile/api/buckets/client"
	"github.com/textileio/textile/buckets"
)

func (tc *textileClient) IsMirrorFile(ctx context.Context, path, bucketSlug string) bool {
	mirrorFile, _ := tc.GetModel().FindMirrorFileByPathAndBucketSlug(ctx, path, bucketSlug)
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
	_, err := tc.GetModel().CreateMirrorFile(ctx, mf)
	if err != nil {
		return nil, err
	}

	return mf, nil
}

func (tc *textileClient) addCurrentUserAsFileOwner(ctx context.Context, bucketsClient *SecureBucketClient, key, path string) error {
	roles := make(map[string]buckets.Role)
	pk, err := tc.kc.GetStoredPublicKey()
	if err != nil {
		return err
	}
	tpk := thread.NewLibp2pPubKey(pk)
	roles[tpk.String()] = buckets.Admin

	return bucketsClient.PushPathAccessRoles(ctx, key, path, roles)
}

func (tc *textileClient) UploadFileToHub(ctx context.Context, b Bucket, path string, reader io.Reader) (result path.Resolved, root path.Path, err error) {
	// XXX: locking?

	bucket, err := tc.GetModel().FindBucket(ctx, b.Slug())
	if err != nil {
		return nil, nil, err
	}

	hubCtx, _, err := tc.getBucketContext(ctx, bucket.RemoteDbID, b.Slug(), true, bucket.EncryptionKey)
	if err != nil {
		return nil, nil, err
	}

	bucketsClient := NewSecureBucketsClient(
		tc.hb,
		b.Slug(),
	)

	result, root, err = bucketsClient.PushPath(hubCtx, bucket.RemoteBucketKey, path, reader)
	if err != nil {
		return nil, nil, err
	}

	err = tc.addCurrentUserAsFileOwner(hubCtx, bucketsClient, bucket.RemoteBucketKey, path)
	if err != nil {
		// not returning since we dont want to halt the whole process
		// also acl will still work since they are the owner
		// of the thread so this is more for showing members view
		log.Error("Unable to push path access roles for owner", err)
	}

	return result, root, nil
}

// Creates a mirror bucket.
func (tc *textileClient) createMirrorBucket(ctx context.Context, schema model.BucketSchema) (*model.MirrorBucketSchema, error) {
	log.Debug("Creating a new mirror bucket with slug " + defaultPersonalMirrorBucketSlug)
	dbID, err := tc.createMirrorThread(ctx)
	if err != nil {
		return nil, err
	}
	hubCtx, _, err := tc.getBucketContext(ctx, utils.CastDbIDToString(*dbID), defaultPersonalMirrorBucketSlug, true, schema.EncryptionKey)
	if err != nil {
		return nil, err
	}

	// create mirror bucket
	// TODO: use bucketname + _mirror to support any local buckets not just personal
	b, err := tc.hb.Create(hubCtx, bc.WithName(defaultPersonalMirrorBucketSlug), bc.WithPrivate(true))
	if err != nil {
		return nil, err
	}

	return &model.MirrorBucketSchema{
		RemoteDbID:      utils.CastDbIDToString(*dbID),
		RemoteBucketKey: b.Root.Key,
		HubAddr:         tc.cfg.GetString(config.TextileHubTarget, ""),
	}, nil
}

// Creates a remote hub thread for the mirror bucket
func (tc *textileClient) createMirrorThread(ctx context.Context) (*thread.ID, error) {
	log.Debug("createMirrorThread: Generating a new threadID ...")
	var err error
	ctx, err = tc.getHubCtx(ctx)
	if err != nil {
		return nil, err
	}

	dbID := thread.NewIDV1(thread.Raw, 32)

	log.Debug("createMirrorThread: Creating Thread DB for bucket at db " + dbID.String())
	if err := tc.ht.NewDB(ctx, dbID); err != nil {
		return nil, err
	}
	log.Debug("createMirrorThread: Thread DB Created")
	return &dbID, nil
}
