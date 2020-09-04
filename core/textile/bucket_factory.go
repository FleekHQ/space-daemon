package textile

import (
	"context"
	"errors"
	"fmt"

	"github.com/FleekHQ/space-daemon/core/textile/common"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/space/domain"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/FleekHQ/space-daemon/core/textile/bucket"
	"github.com/FleekHQ/space-daemon/core/textile/utils"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/alecthomas/jsonschema"
	textileApiClient "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/db"
	bc "github.com/textileio/textile/api/buckets/client"
	buckets_pb "github.com/textileio/textile/api/buckets/pb"
	"github.com/textileio/textile/cmd"
	tdb "github.com/textileio/textile/threaddb"
)

func NotFound(slug string) error {
	return errors.New(fmt.Sprintf("bucket %s not found", slug))
}

func (tc *textileClient) GetBucket(ctx context.Context, slug string) (Bucket, error) {
	if err := tc.requiresRunning(); err != nil {
		return nil, err
	}

	return tc.getBucket(ctx, slug)
}

func (tc *textileClient) getBucket(ctx context.Context, slug string) (Bucket, error) {
	ctx, root, err := tc.getBucketRootFromSlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	b := bucket.New(
		root,
		tc.getOrCreateBucketContext,
		NewSecureBucketsClient(
			tc.bucketsClient,
			slug,
		),
	)

	return b, nil
}

func (tc *textileClient) GetDefaultBucket(ctx context.Context) (Bucket, error) {
	return tc.GetBucket(ctx, defaultPersonalBucketSlug)
}

func (tc *textileClient) getBucketContext(ctx context.Context, sDbID string, bucketSlug string, ishub bool, enckey []byte) (context.Context, *thread.ID, error) {
	log.Debug("getBucketContext: Getting bucket context with dbid:" + sDbID)

	dbID, err := utils.ParseDbIDFromString(sDbID)
	if err != nil {
		log.Error("Error casting thread id", err)
		return nil, nil, err
	}
	ctx, err = utils.GetThreadContext(ctx, bucketSlug, *dbID, ishub, tc.kc, tc.hubAuth)

	if err != nil {
		return nil, nil, err
	}

	ctx = common.NewBucketEncryptionKeyContext(ctx, enckey)

	return ctx, dbID, err
}

// Returns a context that works for accessing a bucket
func (tc *textileClient) getOrCreateBucketContext(ctx context.Context, bucketSlug string) (context.Context, *thread.ID, error) {
	log.Debug("getOrCreateBucketContext: Getting bucket context")

	log.Debug("getOrCreateBucketContext: Fetching thread id from meta store")
	m := tc.getModel()
	bucketSchema, notFoundErr := m.FindBucket(ctx, bucketSlug)

	if notFoundErr == nil { // This means the bucket was already present in the schema
		var err error
		var dbID *thread.ID
		ctx, dbID, err = tc.getBucketContext(ctx, bucketSchema.DbID, bucketSlug, false, bucketSchema.EncryptionKey)
		if err != nil {
			return nil, nil, err
		}

		return ctx, dbID, err
	}

	// We need to create the thread and store it in the collection
	log.Debug("getOrCreateBucketContext: Thread ID not found in meta store. Generating a new one...")
	dbID := thread.NewIDV1(thread.Raw, 32)

	log.Debug("getOrCreateBucketContext: Creating Thread DB for bucket " + bucketSlug + " at db " + dbID.String())
	if err := tc.threads.NewDB(ctx, dbID); err != nil {
		return nil, nil, err
	}
	log.Debug("getOrCreateBucketContext: Thread DB Created")
	bucketSchema, err := m.CreateBucket(ctx, bucketSlug, utils.CastDbIDToString(dbID))
	if err != nil {
		return nil, nil, err
	}

	bucketCtx, _, err := tc.getBucketContext(ctx, utils.CastDbIDToString(dbID), bucketSlug, false, bucketSchema.EncryptionKey)
	if err != nil {
		return nil, nil, err
	}
	log.Debug("getOrCreateBucketContext: Returning bucket context")

	return bucketCtx, &dbID, err
}

func (tc *textileClient) ListBuckets(ctx context.Context) ([]Bucket, error) {
	if err := tc.requiresRunning(); err != nil {
		return nil, err
	}

	return tc.listBuckets(ctx)
}

func (tc *textileClient) listBuckets(ctx context.Context) ([]Bucket, error) {
	bucketList, err := tc.getModel().ListBuckets(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]Bucket, 0)
	for _, b := range bucketList {
		bucketObj, err := tc.getBucket(ctx, b.Slug)
		if err != nil {
			return nil, err
		}
		result = append(result, bucketObj)
	}

	return result, nil
}

func (tc *textileClient) getBucketRootFromSlug(ctx context.Context, slug string) (context.Context, *buckets_pb.Root, error) {
	ctx, _, err := tc.getOrCreateBucketContext(ctx, slug)
	if err != nil {
		return nil, nil, err
	}

	bucketListReply, err := tc.bucketsClient.List(ctx)

	if err != nil {
		return nil, nil, err
	}

	for _, root := range bucketListReply.Roots {
		if root.Name == slug {
			return ctx, root, nil
		}
	}
	return nil, nil, NotFound(slug)
}

// Creates a bucket.
func (tc *textileClient) CreateBucket(ctx context.Context, bucketSlug string) (Bucket, error) {
	if err := tc.requiresRunning(); err != nil {
		return nil, err
	}

	return tc.createBucket(ctx, bucketSlug)
}

func (tc *textileClient) createBucket(ctx context.Context, bucketSlug string) (Bucket, error) {
	log.Debug("Creating a new bucket with slug " + bucketSlug)
	var err error
	m := tc.getModel()

	if b, _ := tc.getBucket(ctx, bucketSlug); b != nil {
		return b, nil
	}

	if err != nil {
		return nil, err
	}

	ctx, dbID, err := tc.getOrCreateBucketContext(ctx, bucketSlug)

	if err != nil {
		return nil, err
	}

	log.Debug("Creating Bucket in db " + dbID.String())
	// create bucket
	b, err := tc.bucketsClient.Init(ctx, bc.WithName(bucketSlug), bc.WithPrivate(true))
	if err != nil {
		return nil, err
	}

	// We store the bucket in a meta thread so that we can later fetch a list of all buckets
	log.Debug("Bucket " + bucketSlug + " created. Storing metadata.")
	schema, err := m.CreateBucket(ctx, bucketSlug, dbID.String())
	if err != nil {
		return nil, err
	}

	mirrorSchema, err := tc.createMirrorBucket(ctx, *schema)
	if err != nil {
		return nil, err
	}

	if mirrorSchema != nil {
		_, err = m.CreateMirrorBucket(ctx, bucketSlug, mirrorSchema)
		if err != nil {
			return nil, err
		}
	}

	newB := bucket.New(
		b.Root,
		tc.getOrCreateBucketContext,
		NewSecureBucketsClient(
			tc.bucketsClient,
			bucketSlug,
		),
	)

	return newB, nil
}

func (tc *textileClient) ShareBucket(ctx context.Context, bucketSlug string) (*textileApiClient.DBInfo, error) {
	bs, err := tc.getModel().FindBucket(ctx, bucketSlug)

	if err != nil {
		return nil, err
	}

	dbID, err := utils.ParseDbIDFromString(bs.DbID)
	b, err := tc.threads.GetDBInfo(ctx, *dbID)

	// replicate with the hub
	hubma := tc.cfg.GetString(config.TextileHubMa, "")
	if _, err := tc.netc.AddReplicator(ctx, *dbID, cmd.AddrFromStr(hubma)); err != nil {
		log.Error("Unable to replicate on the hub: ", err)
		// proceeding still because local/public IP
		// addresses could be used to join thread
	}

	return b, err
}

func (tc *textileClient) joinBucketViaAddress(ctx context.Context, address string, key thread.Key, bucketSlug string) error {
	multiaddress, err := ma.NewMultiaddr(address)
	if err != nil {
		log.Error("Unable to parse multiaddr", err)
		return err
	}

	var (
		schema  *jsonschema.Schema
		indexes = []db.Index{{
			Path: "path",
		}}
	)

	reflector := jsonschema.Reflector{ExpandedStruct: true}
	schema = reflector.Reflect(&tdb.Bucket{})
	err = tc.threads.NewDBFromAddr(ctx, multiaddress, key, db.WithNewManagedCollections(db.CollectionConfig{
		Name:    "buckets",
		Schema:  schema,
		Indexes: indexes,
	}))
	if err != nil {
		log.Error("Unable to join addr", err)
		return err
	}

	dbID, err := thread.FromAddr(multiaddress)

	tc.getModel().UpsertBucket(ctx, bucketSlug, utils.CastDbIDToString(dbID))

	return nil
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
		if err := tc.joinBucketViaAddress(ctx, a, k, slug); err != nil {
			continue
		}

		return true, nil
	}

	log.Info("unable to join any advertised addresses, so joining via the hub instead")

	// if it reached here then no addresses worked, try the hub
	hubAddr := tc.cfg.GetString(config.TextileHubMa, "") + "/thread/" + dbID.String()
	if err := tc.joinBucketViaAddress(ctx, hubAddr, k, slug); err != nil {
		log.Error("error joining bucket from hub", err)
		return false, err
	}

	return true, nil
}

func (tc *textileClient) ToggleBucketBackup(ctx context.Context, bucketSlug string, bucketBackup bool) (bool, error) {
	bucketSchema, err := tc.getModel().BucketBackupToggle(ctx, bucketSlug, bucketBackup)
	if err != nil {
		return false, err
	}

	return bucketSchema.Backup, nil
}

func (tc *textileClient) IsBucketBackup(ctx context.Context, bucketSlug string) bool {
	bucketSchema, err := tc.getModel().FindBucket(ctx, bucketSlug)
	if err != nil {
		return false
	}

	return bucketSchema.Backup
}
