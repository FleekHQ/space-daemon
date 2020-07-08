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
	ID          core.InstanceID `json:"_id"`
	Slug        string          `json:"slug"`
	DbID        string
	BackupInHub bool
}

const metaThreadName = "metathread"
const bucketCollectionName = "BucketMetadata"

var errBucketNotFound = errors.New("Bucket not found")

func (tc *textileClient) initBucketCollection(ctx context.Context) (*thread.ID, error) {
	dbID, err := tc.findOrCreateThreadID(ctx, tc.threads, metaThreadName)
	if err != nil {
		return nil, err
	}

	if err = tc.threads.NewDB(ctx, *dbID); err != nil {
		log.Debug("initBucketCollection: db already exists")
	}
	if err := tc.threads.NewCollection(ctx, *dbID, db.CollectionConfig{
		Name:   bucketCollectionName,
		Schema: util.SchemaFromInstance(&BucketSchema{}, false),
		Indexes: []db.Index{{
			Path:   "slug",
			Unique: true,
		}},
	}); err != nil {
		log.Debug("initBucketCollection: collection already exists")
	}

	return dbID, nil
}

func (tc *textileClient) storeBucketInCollection(ctx context.Context, bucketSlug, dbID string, backupInHub bool) (*BucketSchema, error) {
	log.Debug("storeBucketInCollection: Storing bucket " + bucketSlug)
	if existingBucket, err := tc.findBucketInCollection(ctx, bucketSlug); err == nil {
		log.Debug("storeBucketInCollection: Bucket already in collection")
		return existingBucket, nil
	}

	log.Debug("storeBucketInCollection: Initializing db")
	metaDbID, err := tc.initBucketCollection(ctx)
	if err != nil && metaDbID == nil {
		return nil, err
	}

	newInstance := &BucketSchema{
		Slug:        bucketSlug,
		ID:          "",
		DbID:        dbID,
		BackupInHub: backupInHub,
	}

	instances := client.Instances{newInstance}
	log.Debug("storeBucketInCollection: Creating instance")

	res, err := tc.threads.Create(ctx, *metaDbID, bucketCollectionName, instances)
	if err != nil {
		return nil, err
	}
	id := res[0]
	return &BucketSchema{
		Slug:        newInstance.Slug,
		ID:          core.InstanceID(id),
		DbID:        newInstance.DbID,
		BackupInHub: newInstance.BackupInHub,
	}, nil
}

func (tc *textileClient) findBucketInCollection(ctx context.Context, bucketSlug string) (*BucketSchema, error) {
	dbID, err := tc.initBucketCollection(ctx)
	if err != nil || dbID == nil {
		return nil, err
	}
	log.Debug("findBucketInCollection: finding bucket " + bucketSlug + " in db " + dbID.String())

	rawBuckets, err := tc.threads.Find(ctx, *dbID, bucketCollectionName, db.Where("slug").Eq(bucketSlug).UseIndex("slug"), &BucketSchema{})
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
	dbID, err := tc.initBucketCollection(ctx)
	if err != nil && dbID == nil {
		return nil, err
	}

	rawBuckets, err := tc.threads.Find(ctx, *dbID, bucketCollectionName, &db.Query{}, &BucketSchema{})
	if rawBuckets == nil {
		return []*BucketSchema{}, nil
	}
	buckets := rawBuckets.([]*BucketSchema)
	return buckets, nil
}
