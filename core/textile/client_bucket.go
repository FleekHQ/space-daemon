package textile

import (
	"context"
	"errors"
	"fmt"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/alecthomas/jsonschema"
	"github.com/libp2p/go-libp2p-core/crypto"
	ma "github.com/multiformats/go-multiaddr"
	tc "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/db"
	bc "github.com/textileio/textile/api/buckets/client"
	bucketsproto "github.com/textileio/textile/api/buckets/pb"
	"github.com/textileio/textile/api/common"
	buckets "github.com/textileio/textile/buckets"
	"github.com/textileio/textile/cmd"
)

var (
	schema  *jsonschema.Schema
	indexes = []db.Index{{
		Path: "path",
	}}
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
	bucketList, err := tc.getBucketsFromCollection()
	if err != nil {
		return nil, err
	}

	result := make([]Bucket, 0)
	for _, r := range bucketList {
		bucket, err := tc.bucketsClient.Init(ctx, r.Slug)
		if err != nil {
			return nil, err
		}
		b := tc.getNewBucket(bucket.Root)
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
	bucketList, err := tc.getBucketsFromCollection()
	if err != nil {
		log.Error("error while fetching bucket list ", err)
		return nil, err
	}
	for _, r := range bucketList {
		if r.Slug == bucketSlug {
			log.Warn("BucketData already exists", "bucketSlug:"+bucketSlug)
			b, err := tc.bucketsClient.Init(ctx, bucketSlug)
			if err != nil {
				return nil, err
			}
			return tc.getNewBucket(b.Root), nil
		}
	}

	// create bucket
	log.Debug("Generating bucket")
	// We store the bucket in a meta thread so that we can later fetch a list of all buckets
	tc.storeBucketInCollection(bucketSlug)
	b, err := tc.bucketsClient.Init(ctx, bc.WithName(bucketSlug), bc.WithPrivate(true))
	if err != nil {
		return nil, err
	}

	newB := tc.getNewBucket(b.Root)

	return newB, nil
}

func (tc *textileClient) ShareBucket(ctx context.Context, bucketSlug string) (*tc.DBInfo, error) {
	dbBytes, err := tc.store.Get([]byte(getThreadIDStoreKey(bucketSlug)))

	if err != nil {
		return nil, err
	}

	dbID, err := thread.Cast(dbBytes)
	b, err := tc.threads.GetDBInfo(ctx, dbID)

	// replicate with the hub
	hubma := tc.cfg.GetString(config.TextileHubMa, "")
	if _, err := tc.netc.AddReplicator(ctx, dbID, cmd.AddrFromStr(hubma)); err != nil {
		log.Error("Unable to replicate on the hub: ", err)
		// proceeding still because local/public IP
		// addresses could be used to join thread
	}

	return b, err
}

func (tc *textileClient) JoinBucket(ctx context.Context, slug string, ti *domain.ThreadInfo) (bool, error) {
	k, err := thread.KeyFromString(ti.Key)

	// get the DB ID from the first ma
	ma1, err := ma.NewMultiaddr(ti.Addresses[0])
	if err != nil {
		return false, fmt.Errorf("Unable to parse multiaddr")
	}
	dbID, err := thread.FromAddr(ma1)
	if err != nil {
		return false, fmt.Errorf("Unable to parse db id")
	}

	for _, a := range ti.Addresses {
		ma1, err := ma.NewMultiaddr(a)
		if err != nil {
			log.Error("Unable to parse multiaddr", err)
			continue
		}

		err = tc.threads.NewDBFromAddr(ctx, ma1, k)

		if err != nil {
			log.Error("Unable to join addr", err)
			continue
		}

		// exit on the first address that works
		tc.SaveBucketThreadID(ctx, slug, dbID.String())
		return true, nil
	}

	log.Info("unable to join any advertised addresses, so joining via the hub instead")

	// if it reached here then no addresses worked, try the hub
	hubma, err := ma.NewMultiaddr(tc.cfg.GetString(config.TextileHubMa, "") + "/thread/" + dbID.String())
	if err != nil {
		log.Info("error getting hubma")
		log.Fatal(err)
		return false, err
	}

	reflector := jsonschema.Reflector{ExpandedStruct: true}
	schema = reflector.Reflect(&buckets.Bucket{})
	err = tc.threads.NewDBFromAddr(ctx, hubma, k, db.WithNewManagedCollections(db.CollectionConfig{
		Name:    "buckets",
		Schema:  schema,
		Indexes: indexes,
	}))
	if err != nil {
		log.Error("error joining thread via hub: ", err)
		return false, err
	}

	tc.SaveBucketThreadID(ctx, slug, dbID.String())
	return true, nil
}

func (tc *textileClient) getNewBucket(b *bucketsproto.Root) Bucket {
	newB := newBucket(b, tc, tc.bucketsClient)
	return tc.setBucket(newB.Slug(), newB)
}
