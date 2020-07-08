package textile

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/textile-new/bucket"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/textileio/go-threads/core/thread"
	bc "github.com/textileio/textile/api/buckets/client"
	buckets_pb "github.com/textileio/textile/api/buckets/pb"
	"github.com/textileio/textile/api/common"
)

func NotFound(slug string) error {
	return errors.New(fmt.Sprintf("bucket %s not found", slug))
}

func (tc *textileClient) GetBucket(slug string) (Bucket, error) {
	ctx, root, err := tc.getBucketRootFromSlug(slug)
	if err != nil {
		return nil, err
	}
	b := bucket.New(root, ctx, tc.bucketsClient)

	return b, nil
}

func (tc *textileClient) GetDefaultBucket() (Bucket, error) {
	return tc.GetBucket(defaultPersonalBucketSlug)
}

func getThreadName(userPubKey []byte, bucketSlug string) string {
	return hex.EncodeToString(userPubKey) + "-" + bucketSlug
}

// Returns a context that works for accessing a bucket
func (tc *textileClient) getBucketContext(bucketSlug string) (context.Context, *thread.ID, error) {
	log.Debug("getBucketContext: Getting bucket context")
	var err error
	if err = tc.requiresRunning(); err != nil {
		return nil, nil, err
	}

	ctx := tc.ctx
	var publicKey crypto.PubKey
	kc := keychain.New(tc.store)
	if _, publicKey, err = kc.GetStoredKeyPairInLibP2PFormat(); err != nil {
		return nil, nil, err
	}

	var pubKeyInBytes []byte
	if pubKeyInBytes, err = publicKey.Bytes(); err != nil {
		return nil, nil, err
	}

	ctx = common.NewThreadNameContext(ctx, getThreadName(pubKeyInBytes, bucketSlug))

	var dbID *thread.ID
	log.Debug("getBucketContext: Fetching thread id from local store")
	if dbID, err = tc.findOrCreateThreadID(ctx, tc.threads, bucketSlug); err != nil {
		return nil, nil, err
	}
	log.Debug("getBucketContext: got dbID " + dbID.String())

	ctx = common.NewThreadIDContext(ctx, *dbID)
	log.Debug("getBucketContext: Returning bucket context")
	return ctx, dbID, nil
}

func (tc *textileClient) ListBuckets() ([]Bucket, error) {
	bucketList, err := tc.getBucketsFromCollection()
	if err != nil {
		return nil, err
	}

	result := make([]Bucket, 0)
	for _, b := range bucketList {
		bucketObj, err := tc.GetBucket(b.Slug)
		if err != nil {
			return nil, err
		}
		result = append(result, bucketObj)
	}

	return result, nil
}

func (tc *textileClient) getBucketRootFromSlug(slug string) (context.Context, *buckets_pb.Root, error) {
	ctx, _, err := tc.getBucketContext(slug)
	if err != nil {
		return nil, nil, err
	}

	bucketListReply, err := tc.bucketsClient.List(ctx)

	for _, root := range bucketListReply.Roots {
		if root.Name == slug {
			return ctx, root, nil
		}
	}
	return nil, nil, NotFound(slug)
}

// Creates a bucket.
func (tc *textileClient) CreateBucket(bucketSlug string) (Bucket, error) {
	log.Debug("Creating a new bucket with slug " + bucketSlug)
	var err error

	if b, _ := tc.GetBucket(bucketSlug); b != nil {
		return b, nil
	}

	ctx, dbID, err := tc.getBucketContext(bucketSlug)

	if err != nil {
		return nil, err
	}

	// create bucket
	b, err := tc.bucketsClient.Init(ctx, bc.WithName(bucketSlug), bc.WithPrivate(true))
	if err != nil {
		return nil, err
	}

	// We store the bucket in a meta thread so that we can later fetch a list of all buckets
	log.Debug("Bucket " + bucketSlug + " created. Storing metadata.")
	_, err = tc.storeBucketInCollection(bucketSlug, dbID.String())
	if err != nil {
		return nil, err
	}

	newB := bucket.New(b.Root, ctx, tc.bucketsClient)

	return newB, nil
}
