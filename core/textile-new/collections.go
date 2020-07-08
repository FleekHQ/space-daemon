package textile

import (
	"context"
	"errors"

	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/textileio/go-threads/api/client"
	core "github.com/textileio/go-threads/core/db"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/db"
	"github.com/textileio/go-threads/util"
	"github.com/textileio/textile/api/common"
)

type BucketSchema struct {
	ID   core.InstanceID `json:"_id"`
	Slug string          `json:"slug"`
	DbID string
}

const metaThreadName = "metathread"
const bucketCollectionName = "BucketMetadata"

var errBucketNotFound = errors.New("Bucket not found")

func (tc *textileClient) getMetaThreadContext(ctx context.Context, useHub bool) (context.Context, *thread.ID, error) {
	log.Debug("getMetaThreadContext: Getting context")
	var err error
	if err = tc.requiresRunning(); err != nil {
		return nil, nil, err
	}
	metathreadCtx := ctx
	if useHub == true {
		metathreadCtx, err = tc.getHubCtx(ctx)
		if err != nil {
			return nil, nil, err
		}
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

	metathreadCtx = common.NewThreadNameContext(metathreadCtx, getThreadName(pubKeyInBytes, metaThreadName))

	var dbID *thread.ID
	log.Debug("getMetaThreadContext: Fetching thread id from local store")
	if dbID, err = tc.findOrCreateThreadID(metathreadCtx, tc.threads, metaThreadName); err != nil {
		return nil, nil, err
	}
	log.Debug("getMetaThreadContext: got dbID " + dbID.String())

	metathreadCtx = common.NewThreadIDContext(metathreadCtx, *dbID)
	log.Debug("getMetaThreadContext: Returning context")
	return metathreadCtx, dbID, nil
}

func (tc *textileClient) initBucketCollection(ctx context.Context) (context.Context, *thread.ID, error) {
	metaCtx, dbID, err := tc.getMetaThreadContext(ctx, tc.isConnectedToHub)
	if err != nil {
		return nil, nil, err
	}

	if err = tc.threads.NewDB(metaCtx, *dbID); err != nil {
		log.Debug("initBucketCollection: db already exists")
	}
	if err := tc.threads.NewCollection(metaCtx, *dbID, db.CollectionConfig{
		Name:   bucketCollectionName,
		Schema: util.SchemaFromInstance(&BucketSchema{}, false),
		Indexes: []db.Index{{
			Path:   "slug",
			Unique: true,
		}},
	}); err != nil {
		log.Debug("initBucketCollection: collection already exists")
	}

	return metaCtx, dbID, nil
}

func (tc *textileClient) storeBucketInCollection(ctx context.Context, bucketSlug, dbID string) (*BucketSchema, error) {
	log.Debug("storeBucketInCollection: Storing bucket " + bucketSlug)
	if existingBucket, err := tc.findBucketInCollection(ctx, bucketSlug); err == nil {
		log.Debug("storeBucketInCollection: Bucket already in collection")
		return existingBucket, nil
	}

	log.Debug("storeBucketInCollection: Initializing db")
	metaCtx, metaDbID, err := tc.initBucketCollection(ctx)
	if err != nil && metaDbID == nil {
		return nil, err
	}

	newInstance := &BucketSchema{
		Slug: bucketSlug,
		ID:   "",
		DbID: dbID,
	}

	instances := client.Instances{newInstance}
	log.Debug("storeBucketInCollection: Creating instance")

	res, err := tc.threads.Create(metaCtx, *metaDbID, bucketCollectionName, instances)
	if err != nil {
		return nil, err
	}
	id := res[0]
	return &BucketSchema{
		Slug: newInstance.Slug,
		ID:   core.InstanceID(id),
		DbID: newInstance.DbID,
	}, nil
}

func (tc *textileClient) upsertBucketInCollection(ctx context.Context, bucketSlug, dbID string) (*BucketSchema, error) {
	metaCtx, metaDbID, err := tc.initBucketCollection(ctx)
	if err != nil && metaDbID == nil {
		return nil, err
	}

	if existingBucket, err := tc.findBucketInCollection(ctx, bucketSlug); err == nil {
		tc.threads.Delete(metaCtx, *metaDbID, bucketCollectionName, []string{existingBucket.ID.String()})
	}

	return tc.storeBucketInCollection(ctx, bucketSlug, dbID)
}

func (tc *textileClient) findBucketInCollection(ctx context.Context, bucketSlug string) (*BucketSchema, error) {
	metaCtx, dbID, err := tc.initBucketCollection(ctx)
	if err != nil || dbID == nil {
		return nil, err
	}
	log.Debug("findBucketInCollection: finding bucket " + bucketSlug + " in db " + dbID.String())

	rawBuckets, err := tc.threads.Find(metaCtx, *dbID, bucketCollectionName, db.Where("slug").Eq(bucketSlug).UseIndex("slug"), &BucketSchema{})
	log.Debug("findBucketInCollection: got buckets collection response")
	if rawBuckets == nil {
		return nil, errBucketNotFound
	}

	buckets := rawBuckets.([]*BucketSchema)
	if len(buckets) == 0 {
		return nil, errBucketNotFound
	}
	return buckets[0], nil
}

func (tc *textileClient) getBucketsFromCollection(ctx context.Context) ([]*BucketSchema, error) {
	metaCtx, dbID, err := tc.initBucketCollection(ctx)
	if err != nil && dbID == nil {
		return nil, err
	}

	rawBuckets, err := tc.threads.Find(metaCtx, *dbID, bucketCollectionName, &db.Query{}, &BucketSchema{})
	if rawBuckets == nil {
		return []*BucketSchema{}, nil
	}
	buckets := rawBuckets.([]*BucketSchema)
	return buckets, nil
}
