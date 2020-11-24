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
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/db"
	bc "github.com/textileio/textile/v2/api/bucketsd/client"
	buckets_pb "github.com/textileio/textile/v2/api/bucketsd/pb"
	"github.com/textileio/textile/v2/cmd"
	tdb "github.com/textileio/textile/v2/threaddb"
)

func NotFound(slug string) error {
	return errors.New(fmt.Sprintf("bucket %s not found", slug))
}

type GetBucketForRemoteFileInput struct {
	Path   string
	DbID   string
	Bucket string
}

// Gets a wrapped bucket
// remoteFile is optional. Include if looking for wrappers for remote buckets (mainly used for received files)
func (tc *textileClient) GetBucket(ctx context.Context, slug string, remoteFile *GetBucketForRemoteFileInput) (Bucket, error) {
	if err := tc.requiresRunning(); err != nil {
		return nil, err
	}

	return tc.getBucket(ctx, slug, remoteFile)
}

// Gets a wrapped bucket
// remoteFile is optional. Include if looking for wrappers for remote buckets (mainly used for received files)
func (tc *textileClient) getBucket(ctx context.Context, slug string, remoteFile *GetBucketForRemoteFileInput) (Bucket, error) {
	var root *buckets_pb.Root
	getContextFn := tc.getOrCreateBucketContext
	bucketsClient := tc.bucketsClient
	var err error

	if remoteFile == nil {
		_, root, err = tc.getBucketRootFromSlug(ctx, slug)
	} else {
		root, getContextFn, err = tc.getBucketRootFromReceivedFile(ctx, remoteFile)
		bucketsClient = tc.hb
	}
	if err != nil {
		return nil, err
	}

	b := bucket.New(
		root,
		getContextFn,
		tc.getSecureBucketsClient(bucketsClient),
	)

	// Attach a notifier if the bucket is local
	// So that local ops can be synced to the remote node
	if remoteFile == nil && tc.notifier != nil {
		b.AttachNotifier(tc.notifier)
	}

	return b, nil
}

func (tc *textileClient) getBucketForMirror(ctx context.Context, slug string) (Bucket, error) {
	root, getContextFn, _, err := tc.getBucketRootForMirror(ctx, slug)
	if err != nil {
		return nil, err
	}

	b := bucket.New(
		root,
		getContextFn,
		tc.getSecureBucketsClient(tc.hb),
	)

	return b, nil
}

func (tc *textileClient) GetDefaultBucket(ctx context.Context) (Bucket, error) {
	return tc.GetBucket(ctx, defaultPersonalBucketSlug, nil)
}

func (tc *textileClient) getBucketContext(ctx context.Context, sDbID string, bucketSlug string, ishub bool, enckey []byte) (context.Context, *thread.ID, error) {
	dbID, err := utils.ParseDbIDFromString(sDbID)
	if err != nil {
		log.Error("Error casting thread id", err)
		return nil, nil, err
	}
	ctx, err = utils.GetThreadContext(ctx, bucketSlug, *dbID, ishub, tc.kc, tc.hubAuth, nil)

	if err != nil {
		return nil, nil, err
	}

	ctx = common.NewBucketEncryptionKeyContext(ctx, enckey)

	return ctx, dbID, err
}

// Returns a context that works for accessing a bucket
func (tc *textileClient) getOrCreateBucketContext(ctx context.Context, bucketSlug string) (context.Context, *thread.ID, error) {
	m := tc.GetModel()
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

	managedKey, err := tc.kc.GetManagedThreadKey(getBucketThreadManagedKey(bucketSlug))
	if err != nil {
		return nil, nil, err
	}
	pk, _, err := tc.kc.GetStoredKeyPairInLibP2PFormat()
	if err != nil {
		return nil, nil, err
	}

	if err := tc.threads.NewDB(ctx, dbID, db.WithNewManagedThreadKey(managedKey), db.WithNewManagedLogKey(pk)); err != nil {
		return nil, nil, err
	}

	log.Debug("getOrCreateBucketContext: Thread DB Created")
	bucketSchema, err = m.CreateBucket(ctx, bucketSlug, utils.CastDbIDToString(dbID))
	if err != nil {
		return nil, nil, err
	}

	bucketCtx, _, err := tc.getBucketContext(ctx, utils.CastDbIDToString(dbID), bucketSlug, false, bucketSchema.EncryptionKey)
	if err != nil {
		return nil, nil, err
	}

	return bucketCtx, &dbID, err
}

func (tc *textileClient) ListBuckets(ctx context.Context) ([]Bucket, error) {
	if err := tc.requiresRunning(); err != nil {
		return nil, err
	}

	return tc.listBuckets(ctx)
}

func (tc *textileClient) listBuckets(ctx context.Context) ([]Bucket, error) {
	bucketList, err := tc.GetModel().ListBuckets(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]Bucket, 0)
	for _, b := range bucketList {
		// Skip listing the mirror bucket
		if b.Slug == defaultPersonalMirrorBucketSlug {
			continue
		}
		bucketObj, err := tc.getBucket(ctx, b.Slug, nil)
		if err != nil {
			return nil, err
		}
		result = append(result, bucketObj)
	}

	return result, nil
}

func (tc *textileClient) getBucketRootFromReceivedFile(ctx context.Context, file *GetBucketForRemoteFileInput) (*buckets_pb.Root, bucket.GetBucketContextFn, error) {
	receivedFile, err := tc.GetModel().FindReceivedFile(ctx, file.DbID, file.Bucket, file.Path)
	if err != nil {
		return nil, nil, err
	}

	getCtxFn := func(ctx context.Context, slug string) (context.Context, *thread.ID, error) {
		return tc.getBucketContext(ctx, receivedFile.DbID, receivedFile.Bucket, true, receivedFile.EncryptionKey)
	}

	remoteCtx, _, err := getCtxFn(ctx, receivedFile.Bucket)
	if err != nil {
		return nil, nil, err
	}

	sbs := tc.getSecureBucketsClient(tc.hb)

	b, err := sbs.ListPath(remoteCtx, receivedFile.BucketKey, receivedFile.Path)

	if err != nil {
		return nil, nil, err
	}

	if b != nil {
		return b.GetRoot(), getCtxFn, nil
	}

	return nil, nil, NotFound(receivedFile.Bucket)
}

func (tc *textileClient) getBucketRootForMirror(ctx context.Context, slug string) (*buckets_pb.Root, bucket.GetBucketContextFn, string, error) {
	bucket, err := tc.GetModel().FindBucket(ctx, slug)
	if err != nil {
		return nil, nil, "", err
	}

	getCtxFn := func(ctx context.Context, slug string) (context.Context, *thread.ID, error) {
		return tc.getBucketContext(ctx, bucket.RemoteDbID, bucket.RemoteBucketSlug, true, bucket.EncryptionKey)
	}

	remoteCtx, _, err := getCtxFn(ctx, bucket.RemoteBucketSlug)
	if err != nil {
		return nil, nil, "", err
	}

	sbs := tc.getSecureBucketsClient(tc.hb)

	b, err := sbs.ListPath(remoteCtx, bucket.RemoteBucketKey, "")

	if err != nil {
		return nil, nil, "", err
	}

	if b != nil {
		return b.GetRoot(), getCtxFn, bucket.RemoteBucketSlug, nil
	}

	return nil, nil, "", NotFound(bucket.RemoteBucketSlug)
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
	m := tc.GetModel()

	if b, _ := tc.getBucket(ctx, bucketSlug, nil); b != nil {
		return b, nil
	}

	ctx, dbID, err := tc.getOrCreateBucketContext(ctx, bucketSlug)

	if err != nil {
		return nil, err
	}

	log.Debug("Creating Bucket in db " + dbID.String())
	// create bucket
	b, err := tc.bucketsClient.Create(ctx, bc.WithName(bucketSlug), bc.WithPrivate(true))
	if err != nil {
		return nil, err
	}

	// We store the bucket in a meta thread so that we can later fetch a list of all buckets
	log.Debug("Bucket " + bucketSlug + " created. Storing metadata.")
	schema, err := m.CreateBucket(ctx, bucketSlug, utils.CastDbIDToString(*dbID))
	if err != nil {
		return nil, err
	}

	tc.sync.NotifyBucketCreated(schema.Slug, schema.EncryptionKey)
	tc.sync.NotifyBucketRestore(bucketSlug)

	newB := bucket.New(
		b.Root,
		tc.getOrCreateBucketContext,
		tc.getSecureBucketsClient(tc.bucketsClient),
	)

	return newB, nil
}

func (tc *textileClient) ShareBucket(ctx context.Context, bucketSlug string) (*db.Info, error) {
	bs, err := tc.GetModel().FindBucket(ctx, bucketSlug)
	if err != nil {
		return nil, err
	}

	dbID, err := utils.ParseDbIDFromString(bs.DbID)
	b, err := tc.threads.GetDBInfo(ctx, *dbID)

	// replicate to the hub
	hubma := tc.cfg.GetString(config.TextileHubMa, "")
	if hubma == "" {
		return nil, fmt.Errorf("no textile hub set")
	}

	if _, err := tc.netc.AddReplicator(ctx, *dbID, cmd.AddrFromStr(hubma)); err != nil {
		log.Error("Unable to replicate on the hub: ", err)
		// proceeding still because local/public IP
		// addresses could be used to join thread
	}

	return &b, err
}

func (tc *textileClient) joinBucketViaAddress(ctx context.Context, address string, key thread.Key, bucketSlug string, opts ...db.NewManagedOption) error {
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

	newDbOpts := []db.NewManagedOption{db.WithNewManagedCollections(db.CollectionConfig{
		Name:    "buckets",
		Schema:  schema,
		Indexes: indexes,
	})}
	newDbOpts = append(newDbOpts, opts...)

	err = tc.threads.NewDBFromAddr(ctx, multiaddress, key, newDbOpts...)
	if err != nil {
		log.Error("Unable to join addr", err)
		return err
	}

	dbID, err := thread.FromAddr(multiaddress)
	if err != nil {
		return err
	}

	newBucket, err := tc.GetModel().UpsertBucket(ctx, bucketSlug, utils.CastDbIDToString(dbID))
	if err != nil {
		return err
	}

	newBucketCtx, _, err := tc.getBucketContext(ctx, utils.CastDbIDToString(dbID), bucketSlug, false, newBucket.EncryptionKey)
	if err != nil {
		return err
	}

	// Create bucket in buckets client in case it's not already there
	tc.bucketsClient.Create(newBucketCtx, bc.WithName(bucketSlug), bc.WithPrivate(true))

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
	bucketSchema, err := tc.GetModel().BucketBackupToggle(ctx, bucketSlug, bucketBackup)
	if err != nil {
		return false, err
	}

	if bucketSchema.Backup {
		tc.sync.NotifyBucketBackupOn(bucketSlug)
	} else {
		tc.sync.NotifyBucketBackupOff(bucketSlug)
	}

	return bucketSchema.Backup, nil
}

func (tc *textileClient) BucketBackupRestore(ctx context.Context, bucketSlug string) error {
	tc.sync.NotifyBucketRestore(bucketSlug)

	return nil
}

func (tc *textileClient) IsBucketBackup(ctx context.Context, bucketSlug string) bool {
	bucketSchema, err := tc.GetModel().FindBucket(ctx, bucketSlug)
	if err != nil {
		return false
	}

	return bucketSchema.Backup
}

func GetDefaultBucketSlug() string {
	return defaultPersonalBucketSlug
}

func GetDefaultMirrorBucketSlug() string {
	return defaultPersonalMirrorBucketSlug
}

// Attempts to restore buckets from a hub replication
// Returns nil if there's nothing to restore or the restoration succeeded
func (tc *textileClient) restoreBuckets(ctx context.Context) error {
	bucketList, err := tc.GetModel().ListBuckets(ctx)
	if err != nil {
		return err
	}

	if len(bucketList) == 0 && tc.shouldForceRestore {
		return errors.New("No buckets ready for restore")
	}

	dbs, err := tc.threads.ListDBs(ctx)
	if err != nil {
		return err
	}

	threadsInitialized := true
	for _, b := range bucketList {
		dbID, err := utils.ParseDbIDFromString(b.DbID)
		if err != nil {
			return err
		}

		if _, ok := dbs[*dbID]; !ok {
			threadsInitialized = false
		}
	}

	// Buckets already initialized
	if threadsInitialized {
		return nil
	}

	hubCtx, err := tc.getHubCtx(ctx)
	if err != nil {
		return err
	}

	hubmaStr := tc.cfg.GetString(config.TextileHubMa, "")

	pk, _, err := tc.kc.GetStoredKeyPairInLibP2PFormat()
	if err != nil {
		return err
	}

	// Check if there's a bucket replicated on the hub
	for _, b := range bucketList {
		dbID, err := utils.ParseDbIDFromString(b.DbID)
		if err != nil {
			return err
		}

		_, err = tc.hnetc.GetThread(hubCtx, *dbID)
		replThreadExists := err == nil

		if replThreadExists {
			hubmaWithThreadID := hubmaStr + "/thread/" + dbID.String()

			managedKey, err := tc.kc.GetManagedThreadKey(getBucketThreadManagedKey(b.Slug))
			if err != nil {
				return err
			}

			err = tc.joinBucketViaAddress(
				ctx,
				hubmaWithThreadID,
				managedKey,
				b.Slug,
				db.WithNewManagedBackfillBlock(true),
				db.WithNewManagedLogKey(pk),
				db.WithNewManagedThreadKey(managedKey),
			)
			if err != nil {
				log.Error("could not join replicated bucket", err)
			}

			if err != nil && tc.shouldForceRestore {
				return err
			}
		}
	}

	return nil
}

func getBucketThreadManagedKey(bucketSlug string) string {
	return "bucketKey_" + bucketSlug
}
