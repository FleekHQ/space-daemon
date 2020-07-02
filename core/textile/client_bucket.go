package textile

import (
	"context"
	"errors"
	"fmt"

	"github.com/FleekHQ/space-poc/core/keychain"
	"github.com/FleekHQ/space-poc/log"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/textileio/go-threads/core/thread"
	bucketsproto "github.com/textileio/textile/api/buckets/pb"
	"github.com/textileio/textile/api/common"
)

func NotFound(slug string) error {
	return errors.New(fmt.Sprintf("bucket %s not found", slug))
}

/*
  bucket concurrent methods are helpers that try to keep
  track of a set of the buckets we have, the goal is for all buckets to be singletons
  at the client level so we are always working with unique locks per bucket
*/

// NOTE: Be careful to not call this method without releasing locks first
func (tc *textileClient) getBucket(slug string) Bucket {
	tc.bucketsLock.RLock()

	defer tc.bucketsLock.RUnlock()

	if b, exists := tc.buckets[slug]; exists {
		return b
	}

	return nil
}

// NOTE: Be careful to not call this method without releasing locks first
func (tc *textileClient) setBucket(slug string, b Bucket) Bucket {
	tc.bucketsLock.Lock()

	defer tc.bucketsLock.Unlock()
	if b := tc.buckets[slug]; b != nil {
		return b
	}
	tc.buckets[slug] = b.(*bucket)

	return tc.buckets[slug]
}

// NOTE: Be careful to not call this method without releasing locks first
func (tc *textileClient) setBuckets(buckets []Bucket) []Bucket {
	tc.bucketsLock.Lock()

	defer tc.bucketsLock.Unlock()

	results := make([]Bucket, 0)
	for _, b := range buckets {
		if tc.buckets[b.Slug()] == nil {
			tc.buckets[b.Slug()] = b.(*bucket)
		}
		results = append(results, tc.buckets[b.Slug()])
	}

	return results
}

func (tc *textileClient) GetBucket(ctx context.Context, slug string) (Bucket, error) {
	if b := tc.getBucket(slug); b != nil {
		return b, nil
	}

	buckets, err := tc.ListBuckets(ctx)
	if err != nil {
		log.Error("error while fetching bucketsClient in GetBucket", err)
		return nil, err
	}
	if len(buckets) == 0 {
		log.Error("no bucketsClient found", err)
		return nil, NotFound(slug)
	}
	for _, b := range buckets {
		if b.Slug() == slug {
			return tc.setBucket(slug, b), nil
		}
	}

	return nil, NotFound(slug)
}

func (tc *textileClient) GetDefaultBucket(ctx context.Context) (Bucket, error) {
	return tc.GetBucket(ctx, defaultPersonalBucketSlug)
}

func (tc *textileClient) GetLocalBucketContext(ctx context.Context, bucketSlug string) (context.Context, *thread.ID, error) {
	var publicKey crypto.PubKey
	var err error
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
	log.Debug("Fetching thread id from local store")
	if dbID, err = tc.findOrCreateThreadID(ctx, tc.threads, bucketSlug); err != nil {
		return nil, nil, err
	}

	ctx = common.NewThreadIDContext(ctx, *dbID)
	return ctx, dbID, nil
}

// Returns a context that works for accessing a bucket
func (tc *textileClient) GetBucketContext(ctx context.Context, bucketSlug string) (context.Context, *thread.ID, error) {
	if err := tc.requiresRunning(); err != nil {
		return nil, nil, err
	}

	ctx, err := tc.GetBaseThreadsContext(ctx)
	if err != nil {
		return nil, nil, err
	}
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
	log.Debug("Fetching thread id from local store")
	if dbID, err = tc.findOrCreateThreadID(ctx, tc.threads, bucketSlug); err != nil {
		return nil, nil, err
	}

	ctx = common.NewThreadIDContext(ctx, *dbID)

	return ctx, dbID, nil
}

func (tc *textileClient) ListBuckets(ctx context.Context) ([]Bucket, error) {
	threadsCtx, _, err := tc.GetLocalBucketContext(ctx, defaultPersonalBucketSlug)

	if err != nil {
		log.Error("error in ListBuckets while fetching bucket context", err)
		return nil, err
	}

	bucketList, err := tc.bucketsClient.List(threadsCtx)
	if err != nil {
		return nil, err
	}

	result := make([]Bucket, 0)
	for _, r := range bucketList.Roots {
		b := tc.getNewBucket(r)
		result = append(result, b)
	}

	return tc.setBuckets(result), nil
}

// Creates a bucket.
func (tc *textileClient) CreateBucket(ctx context.Context, bucketSlug string) (Bucket, error) {
	log.Debug("Creating a new bucket with slug " + bucketSlug)
	var err error

	if b := tc.getBucket(bucketSlug); b != nil {
		return b, nil
	}

	if ctx, _, err = tc.GetLocalBucketContext(ctx, bucketSlug); err != nil {
		return nil, err
	}

	// return if bucket aready exists
	// TODO: see if threads.find would be faster
	bucketList, err := tc.bucketsClient.List(ctx)
	if err != nil {
		log.Error("error while fetching bucket list ", err)
		return nil, err
	}
	for _, r := range bucketList.Roots {
		if r.Name == bucketSlug {
			log.Warn("BucketData already exists", "bucketSlug:"+bucketSlug)
			return tc.getNewBucket(r), nil
		}
	}

	// create bucket
	log.Debug("Generating bucket")
	b, err := tc.bucketsClient.Init(ctx, bucketSlug)
	if err != nil {
		return nil, err
	}

	newB := tc.getNewBucket(b.Root)

	return newB, nil
}

func (tc *textileClient) getNewBucket(b *bucketsproto.Root) Bucket {
	newB := newBucket(b, tc, tc.bucketsClient)
	return tc.setBucket(newB.Slug(), newB)
}
