package textile

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/space/domain"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/textile-new/bucket"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/alecthomas/jsonschema"
	"github.com/libp2p/go-libp2p-core/crypto"
	textileApiClient "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/db"
	bc "github.com/textileio/textile/api/buckets/client"
	buckets_pb "github.com/textileio/textile/api/buckets/pb"
	"github.com/textileio/textile/api/common"
	buckets "github.com/textileio/textile/buckets"
	"github.com/textileio/textile/cmd"
)

func NotFound(slug string) error {
	return errors.New(fmt.Sprintf("bucket %s not found", slug))
}

func (tc *textileClient) GetBucket(ctx context.Context, slug string) (Bucket, error) {
	ctx, root, err := tc.getBucketRootFromSlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	b := bucket.New(root, ctx, tc.bucketsClient)

	return b, nil
}

func (tc *textileClient) GetDefaultBucket(ctx context.Context) (Bucket, error) {
	return tc.GetBucket(ctx, defaultPersonalBucketSlug)
}

func getThreadName(userPubKey []byte, bucketSlug string) string {
	return hex.EncodeToString(userPubKey) + "-" + bucketSlug
}

// Returns a context that works for accessing a bucket
func (tc *textileClient) getBucketContext(ctx context.Context, bucketSlug string, useHub bool) (context.Context, *thread.ID, error) {
	log.Debug("getBucketContext: Getting bucket context")
	var err error
	if err = tc.requiresRunning(); err != nil {
		return nil, nil, err
	}
	bucketCtx := ctx
	if useHub == true {
		bucketCtx, err = tc.getHubCtx(ctx)
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

	bucketCtx = common.NewThreadNameContext(bucketCtx, getThreadName(pubKeyInBytes, bucketSlug))

	var dbID thread.ID
	log.Debug("getBucketContext: Fetching thread id from meta store")
	bucketSchema, err := tc.findBucketInCollection(bucketCtx, bucketSlug)
	if err == nil {
		var castErr error
		dbID, castErr = thread.Cast([]byte(bucketSchema.DbID))
		if castErr != nil {
			log.Error("Error casting thread id", castErr)
			return nil, nil, castErr
		}
	} else {
		log.Debug("getBucketContext: Thread ID not found in meta store. Generating a new one...")
		dbID = thread.NewIDV1(thread.Raw, 32)

		log.Debug("getBucketContext: Creating Thread DB")
		if err := tc.threads.NewDB(ctx, dbID); err != nil {
			return nil, nil, err
		}
		log.Debug("getBucketContext: Thread DB Created")
		_, err := tc.storeBucketInCollection(bucketCtx, bucketSlug, castDbIDToString(dbID))
		if err != nil {
			return nil, nil, err
		}
	}

	log.Debug("getBucketContext: got dbID " + dbID.String())

	bucketCtx = common.NewThreadIDContext(bucketCtx, dbID)
	log.Debug("getBucketContext: Returning bucket context")
	return bucketCtx, &dbID, nil
}

func (tc *textileClient) ListBuckets(ctx context.Context) ([]Bucket, error) {
	bucketList, err := tc.getBucketsFromCollection(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]Bucket, 0)
	for _, b := range bucketList {
		bucketObj, err := tc.GetBucket(ctx, b.Slug)
		if err != nil {
			return nil, err
		}
		result = append(result, bucketObj)
	}

	return result, nil
}

func (tc *textileClient) getBucketRootFromSlug(ctx context.Context, slug string) (context.Context, *buckets_pb.Root, error) {
	ctx, _, err := tc.getBucketContext(ctx, slug, tc.isConnectedToHub)
	if err != nil {
		return nil, nil, err
	}

	bucketListReply, err := tc.bucketsClient.List(ctx)

	for _, root := range bucketListReply.Roots {
		if root.Name == slug {
			return ctx, root, nil
		}
	}
	return nil, nil, NotFound(slug)
}

// Creates a bucket.
func (tc *textileClient) CreateBucket(ctx context.Context, bucketSlug string) (Bucket, error) {
	log.Debug("Creating a new bucket with slug " + bucketSlug)
	var err error

	if b, _ := tc.GetBucket(ctx, bucketSlug); b != nil {
		return b, nil
	}

	ctx, dbID, err := tc.getBucketContext(ctx, bucketSlug, tc.isConnectedToHub)

	if err != nil {
		return nil, err
	}

	// create bucket
	b, err := tc.bucketsClient.Init(ctx, bc.WithName(bucketSlug), bc.WithPrivate(true))
	if err != nil {
		return nil, err
	}

	// We store the bucket in a meta thread so that we can later fetch a list of all buckets
	log.Debug("Bucket " + bucketSlug + " created. Storing metadata.")
	_, err = tc.storeBucketInCollection(ctx, bucketSlug, dbID.String())
	if err != nil {
		return nil, err
	}

	newB := bucket.New(b.Root, ctx, tc.bucketsClient)

	return newB, nil
}

func (tc *textileClient) ShareBucket(ctx context.Context, bucketSlug string) (*textileApiClient.DBInfo, error) {
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

func (tc *textileClient) joinBucketViaAddress(ctx context.Context, address string, key thread.Key, bucketSlug string) error {
	multiaddress, err := ma.NewMultiaddr(address)
	if err != nil {
		log.Error("Unable to parse multiaddr", err)
		return err
	}

	err = tc.threads.NewDBFromAddr(ctx, multiaddress, key)
	if err != nil {
		log.Error("Unable to join addr", err)
		return err
	}

	var (
		schema  *jsonschema.Schema
		indexes = []db.Index{{
			Path: "path",
		}}
	)

	reflector := jsonschema.Reflector{ExpandedStruct: true}
	schema = reflector.Reflect(&buckets.Bucket{})
	err = tc.threads.NewDBFromAddr(ctx, multiaddress, key, db.WithNewManagedCollections(db.CollectionConfig{
		Name:    "buckets",
		Schema:  schema,
		Indexes: indexes,
	}))
	if err != nil {
		log.Error("error joining thread via hub: ", err)
		return err
	}

	dbID, err := thread.FromAddr(multiaddress)

	tc.upsertBucketInCollection(ctx, bucketSlug, castDbIDToString(dbID))

	return nil
}

func castDbIDToString(dbID thread.ID) string {
	return string(dbID.Bytes())
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
