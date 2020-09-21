package textile

import (
	"context"
	"io"

	"github.com/FleekHQ/space-daemon/config"

	"github.com/FleekHQ/space-daemon/core/ipfs"

	"github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	api_buckets_pb "github.com/textileio/textile/api/buckets/pb"

	"github.com/FleekHQ/space-daemon/core/textile/bucket"

	"github.com/FleekHQ/space-daemon/core/textile/utils"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/db"
	bc "github.com/textileio/textile/api/buckets/client"
)

// Get a public bucket on hub. Public bucket has no encryption and its content should be accessible directly via ipfs/ipns
// Only use this bucket for items that is okay to be publicly shared
func (tc *textileClient) GetPublicShareBucket(ctx context.Context) (Bucket, error) {
	if err := tc.requiresRunning(); err != nil {
		return nil, err
	}

	return tc.getOrCreatePublicBucket(ctx, defaultPublicShareBucketSlug)
}

func (tc *textileClient) getOrCreatePublicBucket(ctx context.Context, bucketSlug string) (Bucket, error) {
	ctx, dbId, err := tc.getPublicShareBucketContext(ctx, bucketSlug)
	if err != nil {
		return nil, err
	}

	// find if bucket exists
	buckets, err := tc.hb.List(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get public bucket")
	}

	if buckets != nil {
		for _, bucketRoot := range buckets.Roots {
			if bucketRoot.Name == bucketSlug {
				return bucket.New(
					bucketRoot,
					tc.getPublicShareBucketContext,
					tc.hb,
				), nil
			}
		}
	}

	// else create bucketRoot
	bucketRoot, err := tc.createPublicBucket(ctx, *dbId, bucketSlug)
	if err != nil {
		return nil, err
	}

	newB := bucket.New(
		bucketRoot,
		tc.getPublicShareBucketContext,
		tc.hb,
	)

	return newB, nil
}

func (tc *textileClient) getPublicShareBucketContext(ctx context.Context, bucketSlug string) (context.Context, *thread.ID, error) {
	dbId, err := tc.getPublicShareThread(ctx)
	if err != nil {
		return nil, nil, err
	}
	ctx, err = utils.GetThreadContext(ctx, bucketSlug, dbId, true, tc.kc, tc.hubAuth, nil)
	if err != nil {
		return nil, nil, err
	}

	return ctx, &dbId, nil
}

// Creates a public bucket for current user.
func (tc *textileClient) createPublicBucket(ctx context.Context, dbId thread.ID, bucketSlug string) (*api_buckets_pb.Root, error) {
	log.Debug("Creating a new public bucket")

	hubCtx, _, err := tc.getBucketContext(ctx, utils.CastDbIDToString(dbId), bucketSlug, true, nil)
	if err != nil {
		return nil, err
	}

	b, err := tc.hb.Create(hubCtx, bc.WithName(bucketSlug), bc.WithPrivate(false))
	if err != nil {
		return nil, err
	}

	return b.Root, nil
}

const publicShareThreadStoreKey = "publicSharedThreadKey"

// Creates a remote hub thread for the public sharing bucket
func (tc *textileClient) getPublicShareThread(ctx context.Context) (thread.ID, error) {
	// check if db id already exists
	storedDbId, err := tc.store.Get([]byte(publicShareThreadStoreKey))
	if err == nil {
		return thread.Cast(storedDbId)
	}

	// else create new db
	ctx, err = tc.getHubCtx(ctx)
	if err != nil {
		return thread.Undef, err
	}

	dbId := thread.NewIDV1(thread.Raw, 32)

	managedKey, err := tc.kc.GetManagedThreadKey()
	if err != nil {
		log.Error("error getting managed thread key", err)
		return thread.Undef, err
	}

	if err := tc.ht.NewDB(ctx, dbId, db.WithNewManagedThreadKey(managedKey)); err != nil {
		return thread.Undef, err
	}
	log.Debug("Public share thread created")

	err = tc.store.Set([]byte(publicShareThreadStoreKey), []byte(dbId))
	if err != nil {
		return thread.Undef, errors.Wrap(err, "failed to persist public share thread")
	}

	return dbId, nil
}

// DownloadPublicGatewayItem download a cid content from the hubs public gateway
func (tc *textileClient) DownloadPublicGatewayItem(ctx context.Context, cid cid.Cid) (io.ReadCloser, error) {
	gatewayUrl := tc.cfg.GetString(config.TextileHubGatewayUrl, "https://hub-dev.space.storage")
	return ipfs.DownloadIpfsItem(ctx, gatewayUrl, cid)
}
