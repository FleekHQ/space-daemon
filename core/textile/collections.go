package textile

import (
	"context"
	"errors"

	"github.com/FleekHQ/space-daemon/log"
	"github.com/textileio/go-threads/api/client"
	core "github.com/textileio/go-threads/core/db"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/db"
	"github.com/textileio/go-threads/util"
)

type BucketSchema struct {
	ID   core.InstanceID `json:"_id"`
	Slug string          `json:"slug"`
	DbID string
}

const metaThreadName = "metathread"
const bucketCollectionName = "BucketMetadata"

var errBucketNotFound = errors.New("Bucket not found")

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
	log.Debug("stored bucket with dbid " + newInstance.DbID)

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

	rawBuckets, err := tc.threads.Find(metaCtx, *dbID, bucketCollectionName, db.Where("slug").Eq(bucketSlug), &BucketSchema{})
	if err != nil {
		return nil, err
	}

	if rawBuckets == nil {
		return nil, errBucketNotFound
	}

	buckets := rawBuckets.([]*BucketSchema)
	if len(buckets) == 0 {
		return nil, errBucketNotFound
	}
	log.Debug("returning bucket with dbid " + buckets[0].DbID)
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

func getThreadIDStoreKey(bucketSlug string) []byte {
	return []byte(threadIDStoreKey + "_" + bucketSlug)
}

func (tc *textileClient) findOrCreateMetaThreadID(ctx context.Context) (*thread.ID, error) {
	if val, _ := tc.store.Get(getThreadIDStoreKey(metaThreadName)); val != nil {
		// Cast the stored dbID from bytes to thread.ID
		if dbID, err := thread.Cast(val); err != nil {
			return nil, err
		} else {
			return &dbID, nil
		}
	}

	// thread id does not exist yet

	dbID := thread.NewIDV1(thread.Raw, 32)
	dbIDInBytes := dbID.Bytes()

	log.Debug("Created meta thread in db " + dbID.String())

	if err := tc.threads.NewDB(ctx, dbID); err != nil {
		return nil, err
	}

	if err := tc.store.Set([]byte(getThreadIDStoreKey(metaThreadName)), dbIDInBytes); err != nil {
		newErr := errors.New("error while storing thread id: check your local space db accessibility")
		return nil, newErr
	}

	return &dbID, nil
}

func (tc *textileClient) getMetaThreadContext(ctx context.Context, useHub bool) (context.Context, *thread.ID, error) {
	var err error

	var dbID *thread.ID
	if dbID, err = tc.findOrCreateMetaThreadID(ctx); err != nil {
		return nil, nil, err
	}

	metathreadCtx, err := tc.getThreadContext(ctx, metaThreadName, *dbID, false)
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
