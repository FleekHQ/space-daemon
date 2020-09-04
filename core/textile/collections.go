package textile

import (
	"context"

	"github.com/FleekHQ/space-daemon/core/keychain"
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
	*BucketMirrorSchema
}

type MirrorFileSchema struct {
	ID         core.InstanceID `json:"_id"`
	Path       string          `json:"path"`
	BucketSlug string          `json:"bucket_slug"`
	Backup     bool            `json:"backup"`
	Shared     bool            `json:"shared"`

	DbID string
}

// 32 bytes aes key + 16 bytes salt/IV + 32 bytes HMAC key
const BucketEncryptionKeyLength = 32 + 16 + 32

const metaThreadName = "metathreadV1"

const bucketCollectionName = "BucketMetadata"

var errBucketNotFound = errors.New("Bucket not found")

const mirrorFileCollectionName = "MirrorFileMetadata"

var errMirrorFileNotFound = errors.New("Mirror file not found")

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
	if existingBucket, err := tc.FindBucketInCollection(ctx, bucketSlug); err == nil {
		log.Debug("storeBucketInCollection: Bucket already in collection")
		return existingBucket, nil
	}

	log.Debug("storeBucketInCollection: Initializing db")
	metaCtx, metaDbID, err := tc.initBucketCollection(ctx)
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
		Slug:   newInstance.Slug,
		ID:     core.InstanceID(id),
		DbID:   newInstance.DbID,
		Backup: newInstance.Backup,
	}, nil
}

func (tc *textileClient) upsertBucketInCollection(ctx context.Context, bucketSlug, dbID string) (*BucketSchema, error) {
	metaCtx, metaDbID, err := tc.initBucketCollection(ctx)
	if err != nil && metaDbID == nil {
		return nil, err
	}

	if existingBucket, err := tc.FindBucketInCollection(ctx, bucketSlug); err == nil {
		tc.threads.Delete(metaCtx, *metaDbID, bucketCollectionName, []string{existingBucket.ID.String()})
	}

	return tc.storeBucketInCollection(ctx, bucketSlug, dbID)
}

func (tc *textileClient) toggleBucketBackupInCollection(ctx context.Context, bucketSlug string, backup bool) (*BucketSchema, error) {
	metaCtx, metaDbID, err := tc.initBucketCollection(ctx)
	if err != nil && metaDbID == nil {
		return nil, err
	}

	bucket, err := tc.FindBucketInCollection(ctx, bucketSlug)
	if err != nil {
		return nil, err
	}

	bucket.Backup = backup

	instances := client.Instances{bucket}

	err = tc.threads.Save(metaCtx, *metaDbID, bucketCollectionName, instances)
	if err != nil {
		return nil, err
	}

	return bucket, nil
}

func (tc *textileClient) storeBucketMirrorSchema(ctx context.Context, bucketSlug string, mirrorSchema *BucketMirrorSchema) (*BucketSchema, error) {
	metaCtx, metaDbID, err := tc.initBucketCollection(ctx)
	if err != nil && metaDbID == nil {
		return nil, err
	}

	bucket, err := tc.FindBucketInCollection(ctx, bucketSlug)
	if err != nil {
		return nil, err
	}

	bucket.RemoteDbID = mirrorSchema.RemoteDbID
	bucket.HubAddr = mirrorSchema.HubAddr

	instances := client.Instances{bucket}

	err = tc.threads.Save(metaCtx, *metaDbID, bucketCollectionName, instances)
	if err != nil {
		return nil, err
	}

	return bucket, nil
}

func (tc *textileClient) FindBucketInCollection(ctx context.Context, bucketSlug string) (*BucketSchema, error) {
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

// Returns the store key for a thread ID. It uses the keychain to obtain the public key, since the store key depends on it.
func getThreadIDStoreKey(bucketSlug string, kc keychain.Keychain) ([]byte, error) {
	pub, err := kc.GetStoredPublicKey()
	if err != nil {
		return nil, err
	}

	pubInBytes, err := pub.Raw()
	if err != nil {
		return nil, err
	}

	result := []byte(threadIDStoreKey + "_" + bucketSlug)
	result = append(result, pubInBytes...)

	return result, nil
}

func (tc *textileClient) findOrCreateMetaThreadID(ctx context.Context) (*thread.ID, error) {
	storeKey, err := getThreadIDStoreKey(metaThreadName, tc.kc)
	if err != nil {
		return nil, err
	}

	if val, _ := tc.store.Get(storeKey); val != nil {
		// Cast the stored dbID from bytes to thread.ID
		if dbID, err := thread.Cast(val); err != nil {
			return nil, err
		} else {
			return &dbID, nil
		}
	}

	// thread id does not exist yet

	// We need to create an ID that's derived deterministically from the user private key
	// The reason for this is that the user needs to be able to restore the exact ID when moving across devices.
	// The only consideration is that we must try to avoid dbID collisions with other users.
	dbID, err := utils.NewDeterministicThreadID(tc.kc, utils.MetathreadThreadVariant)
	if err != nil {
		return nil, err
	}

	dbIDInBytes := dbID.Bytes()

	log.Debug("Created meta thread in db " + dbID.String())

	if err := tc.threads.NewDB(ctx, dbID); err != nil {
		return nil, err
	}

	if err := tc.store.Set(storeKey, dbIDInBytes); err != nil {
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
	if err != nil {
		return nil, nil, err
	}

	return metathreadCtx, dbID, nil
}

func (tc *textileClient) initMirrorFileCollection(ctx context.Context) (context.Context, *thread.ID, error) {
	metaCtx, dbID, err := tc.getMetaThreadContext(ctx, tc.isConnectedToHub)
	if err != nil {
		return nil, nil, err
	}

	if err = tc.threads.NewDB(metaCtx, *dbID); err != nil {
		log.Debug("initMirrorFileCollection: db already exists")
	}
	if err := tc.threads.NewCollection(metaCtx, *dbID, db.CollectionConfig{
		Name:   mirrorFileCollectionName,
		Schema: util.SchemaFromInstance(&MirrorFileSchema{}, false),
		Indexes: []db.Index{{
			Path:   "path",
			Unique: true, // TODO: multicolumn index
		}},
	}); err != nil {
		log.Debug("initMirrorFileCollection: collection already exists")
	}

	return metaCtx, dbID, nil
}

func (tc *textileClient) findMirrorFileByPathAndBucketSlug(ctx context.Context, path, bucketSlug string) (*MirrorFileSchema, error) {
	metaCtx, dbID, err := tc.initMirrorFileCollection(ctx)
	if err != nil || dbID == nil {
		return nil, err
	}

	rawMirrorFiles, err := tc.threads.Find(metaCtx, *dbID, mirrorFileCollectionName, db.Where("path").Eq(path), &MirrorFileSchema{})
	if err != nil {
		return nil, err
	}

	if rawMirrorFiles == nil {
		return nil, errMirrorFileNotFound
	}

	mirror_files := rawMirrorFiles.([]*MirrorFileSchema)
	if len(mirror_files) == 0 {
		return nil, errMirrorFileNotFound
	}

	log.Debug("returning mirror file with dbid " + mirror_files[0].DbID)
	return mirror_files[0], nil
}

func (tc *textileClient) createMirrorFile(ctx context.Context, mirrorFile *MirrorFile) (*MirrorFileSchema, error) {
	metaCtx, metaDbID, err := tc.initMirrorFileCollection(ctx)
	if err != nil && metaDbID == nil {
		return nil, err
	}

	_, err = tc.findMirrorFileByPathAndBucketSlug(ctx, mirrorFile.Path, mirrorFile.BucketSlug)
	if err != nil {
		return nil, err
	}

	newInstance := &MirrorFileSchema{
		Path:       mirrorFile.Path,
		BucketSlug: mirrorFile.BucketSlug,
		Backup:     mirrorFile.Backup,
		Shared:     mirrorFile.Shared,
	}

	instances := client.Instances{newInstance}

	res, err := tc.threads.Create(metaCtx, *metaDbID, mirrorFileCollectionName, instances)
	if err != nil {
		return nil, err
	}
	log.Debug("stored mirror file with dbid " + newInstance.DbID)

	id := res[0]
	return &MirrorFileSchema{
		Path:       newInstance.Path,
		BucketSlug: newInstance.BucketSlug,
		Backup:     newInstance.Backup,
		Shared:     newInstance.Shared,
		ID:         core.InstanceID(id),
		DbID:       newInstance.DbID,
	}, nil
}
