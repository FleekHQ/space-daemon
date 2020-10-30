package model

import (
	"context"

	"github.com/FleekHQ/space-daemon/core/textile/utils"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/pkg/errors"
	"github.com/textileio/go-threads/api/client"
	core "github.com/textileio/go-threads/core/db"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/db"
	"github.com/textileio/go-threads/util"
)

type BucketSchema struct {
	ID            core.InstanceID `json:"_id"`
	Slug          string          `json:"slug"`
	Backup        bool            `json:"backup"`
	EncryptionKey []byte          `json:"hub_key"`
	DbID          string
	*MirrorBucketSchema
}

const bucketModelName = "BucketMetadata"

// 32 bytes aes key + 16 bytes salt/IV + 32 bytes HMAC key
const BucketEncryptionKeyLength = 32 + 16 + 32

var errBucketNotFound = errors.New("Bucket not found")

func (m *model) CreateBucket(ctx context.Context, bucketSlug, dbID string) (*BucketSchema, error) {
	log.Debug("Model.CreateBucket: Storing bucket " + bucketSlug)
	if existingBucket, err := m.FindBucket(ctx, bucketSlug); err == nil {
		log.Debug("Model.CreateBucket: Bucket already in collection")
		return existingBucket, nil
	}

	log.Debug("Model.CreateBucket: Initializing db")
	metaCtx, metaDbID, err := m.initBucketModel(ctx)
	if err != nil && metaDbID == nil {
		return nil, err
	}

	bucketEncryptionKey, err := utils.RandBytes(BucketEncryptionKeyLength)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate bucket encryption key")
	}

	newInstance := &BucketSchema{
		Slug:          bucketSlug,
		ID:            "",
		DbID:          dbID,
		Backup:        true,
		EncryptionKey: bucketEncryptionKey,
		MirrorBucketSchema: &MirrorBucketSchema{
			HubAddr:          "",
			RemoteBucketKey:  "",
			RemoteDbID:       "",
			RemoteBucketSlug: "",
		},
	}

	instances := client.Instances{newInstance}
	log.Debug("Model.CreateBucket: Creating instance")

	res, err := m.threads.Create(metaCtx, *metaDbID, bucketModelName, instances)
	if err != nil {
		return nil, err
	}
	log.Debug("Model.CreateBucket: stored bucket with dbid " + newInstance.DbID)

	id := res[0]
	return &BucketSchema{
		Slug:   newInstance.Slug,
		ID:     core.InstanceID(id),
		DbID:   newInstance.DbID,
		Backup: newInstance.Backup,
		MirrorBucketSchema: &MirrorBucketSchema{
			HubAddr:          newInstance.MirrorBucketSchema.HubAddr,
			RemoteBucketKey:  newInstance.MirrorBucketSchema.RemoteBucketKey,
			RemoteDbID:       newInstance.MirrorBucketSchema.RemoteDbID,
			RemoteBucketSlug: newInstance.MirrorBucketSchema.RemoteBucketSlug,
		},
	}, nil
}

func (m *model) UpsertBucket(ctx context.Context, bucketSlug, dbID string) (*BucketSchema, error) {
	metaCtx, metaDbID, err := m.initBucketModel(ctx)
	if err != nil && metaDbID == nil {
		return nil, err
	}

	if existingBucket, err := m.FindBucket(ctx, bucketSlug); err == nil {
		m.threads.Delete(metaCtx, *metaDbID, bucketModelName, []string{existingBucket.ID.String()})
	}

	return m.CreateBucket(ctx, bucketSlug, dbID)
}

func (m *model) BucketBackupToggle(ctx context.Context, bucketSlug string, backup bool) (*BucketSchema, error) {
	metaCtx, metaDbID, err := m.initBucketModel(ctx)
	if err != nil && metaDbID == nil {
		return nil, err
	}

	bucket, err := m.FindBucket(ctx, bucketSlug)
	if err != nil {
		return nil, err
	}

	bucket.Backup = backup

	instances := client.Instances{bucket}

	err = m.threads.Save(metaCtx, *metaDbID, bucketModelName, instances)
	if err != nil {
		return nil, err
	}

	return bucket, nil
}

func (m *model) FindBucket(ctx context.Context, bucketSlug string) (*BucketSchema, error) {
	metaCtx, dbID, err := m.initBucketModel(ctx)
	if err != nil || dbID == nil {
		return nil, err
	}

	rawBuckets, err := m.threads.Find(metaCtx, *dbID, bucketModelName, db.Where("slug").Eq(bucketSlug), &BucketSchema{})
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

	return buckets[0], nil
}

func (m *model) ListBuckets(ctx context.Context) ([]*BucketSchema, error) {
	metaCtx, dbID, err := m.initBucketModel(ctx)
	if err != nil && dbID == nil {
		return nil, err
	}

	rawBuckets, err := m.threads.Find(metaCtx, *dbID, bucketModelName, &db.Query{}, &BucketSchema{})
	if rawBuckets == nil {
		return []*BucketSchema{}, nil
	}
	buckets := rawBuckets.([]*BucketSchema)
	return buckets, nil
}

func (m *model) initBucketModel(ctx context.Context) (context.Context, *thread.ID, error) {
	metaCtx, dbID, err := m.getMetaThreadContext(ctx)
	if err != nil {
		return nil, nil, err
	}

	m.threads.NewCollection(metaCtx, *dbID, GetBucketCollectionConfig())

	return metaCtx, dbID, nil
}

func GetBucketCollectionConfig() db.CollectionConfig {
	return db.CollectionConfig{
		Name:   bucketModelName,
		Schema: util.SchemaFromInstance(&BucketSchema{}, false),
		Indexes: []db.Index{{
			Path:   "slug",
			Unique: true,
		}},
	}
}
