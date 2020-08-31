package textile

import (
	"context"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/textileio/go-threads/api/client"
	core "github.com/textileio/go-threads/core/db"
	bc "github.com/textileio/textile/api/buckets/client"
)

type BucketMirrorSchema struct {
	RemoteDbID string `json:"remoteDbId"`
	HubAddr    string `json:"HubAddr"`
}

// Creates a mirror bucket.
func (tc *textileClient) createMirrorBucket(ctx context.Context, schema BucketSchema) (*BucketMirrorSchema, error) {
	bucketSlug := schema.Slug

	hubCtx, err := tc.getHubCtx(ctx)
	if err != nil {
		return nil, err
	}

	log.Debug("Creating a new mirror bucket with slug " + bucketSlug)
	if b, _ := tc.GetBucket(hubCtx, bucketSlug); b != nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	hubCtx, dbID, err := tc.GetBucketContext(hubCtx, bucketSlug)
	if err != nil {
		return nil, err
	}

	// create mirror bucket
	log.Debug("Creating Mirror Bucket in db " + dbID.String())
	_, err = tc.bucketsClient.Init(hubCtx, bc.WithName(bucketSlug), bc.WithPrivate(true))
	if err != nil {
		return nil, err
	}

	// We store the bucket in a meta thread so that we can later fetch a list of all buckets
	log.Debug("Mirror Bucket " + bucketSlug + " created. Storing metadata.")
	mirrorSchema, err := tc.storeMirrorBucketInCollection(hubCtx, schema)
	if err != nil {
		return nil, err
	}

	return mirrorSchema, nil
}

func (tc *textileClient) storeMirrorBucketInCollection(hubCtx context.Context, instance BucketSchema) (*BucketMirrorSchema, error) {
	bucketSlug := instance.Slug

	log.Debug("storeMirrorBucketInCollection: Storing mirror bucket " + bucketSlug)
	if existingBucket, err := tc.findBucketInCollection(hubCtx, bucketSlug); err == nil {
		log.Debug("storeMirrorBucketInCollection: Bucket already in collection")
		return existingBucket.BucketMirrorSchema, nil
	}

	log.Debug("storeMirrorBucketInCollection: Initializing DB")
	metaHubCtx, metaDbID, err := tc.initBucketCollection(hubCtx)
	if err != nil && metaDbID == nil {
		return nil, err
	}

	hubCtx, remoteDbID, err := tc.GetBucketContext(hubCtx, bucketSlug)
	if err != nil {
		return nil, err
	}

	instance.ID = ""
	instance.RemoteDbID = remoteDbID.String()

	instances := client.Instances{instance}
	log.Debug("storeMirrorBucketInCollection: Creating instance")

	res, err := tc.threads.Create(metaHubCtx, *metaDbID, bucketCollectionName, instances)
	if err != nil {
		return nil, err
	}
	log.Debug("storeMirrorBucketInCollection: Stored mrror bucket with DbIB=" + instance.DbID)

	return &BucketMirrorSchema{
		RemoteDbID: core.InstanceID(res[0]).String(),
		HubAddr:    tc.cfg.GetString(config.TextileHubTarget, ""),
	}, nil
}
