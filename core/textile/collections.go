package textile

import (
	"context"

	"github.com/textileio/go-threads/api/client"
	core "github.com/textileio/go-threads/core/db"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/db"
	"github.com/textileio/go-threads/util"
)

type bucketSchema struct {
	ID   core.InstanceID `json:"_id"`
	Slug string
}

const metaThreadName = "metathread"
const bucketCollectionName = "BucketMetadata"

func (tc *textileClient) initBucketCollection(ctx context.Context) (*thread.ID, error) {
	dbID, err := tc.findOrCreateThreadID(ctx, tc.threads, metaThreadName)
	if err != nil {
		return nil, err
	}

	if err = tc.threads.NewDB(ctx, *dbID); err != nil {
		return nil, err
	}
	if err := tc.threads.NewCollection(ctx, *dbID, db.CollectionConfig{
		Name:   bucketCollectionName,
		Schema: util.SchemaFromInstance(&bucketSchema{}, false),
	}); err != nil {
		return nil, err
	}

	return dbID, nil
}

func (tc *textileClient) storeBucketInCollection(bucketSlug string) error {
	ctx, err := tc.GetBaseThreadsContext(context.Background())
	if err != nil {
		return err
	}
	dbID, err := tc.initBucketCollection(ctx)
	if err != nil {
		return err
	}

	newInstance := &bucketSchema{
		Slug: bucketSlug,
		ID:   "",
	}

	instances := client.Instances{newInstance}

	_, err = tc.threads.Create(ctx, *dbID, bucketCollectionName, instances)
	return err
}

func (tc *textileClient) getBucketsFromCollection() ([]*bucketSchema, error) {
	ctx, err := tc.GetBaseThreadsContext(context.Background())
	if err != nil {
		return nil, err
	}
	dbID, err := tc.initBucketCollection(ctx)
	if err != nil {
		return nil, err
	}

	rawBuckets, err := tc.threads.Find(ctx, *dbID, bucketCollectionName, &db.Query{}, &bucketSchema{})
	buckets := rawBuckets.([]*bucketSchema)

	return buckets, nil
}
