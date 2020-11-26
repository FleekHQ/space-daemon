package utils

import (
	"context"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/core/store"
	"github.com/FleekHQ/space-daemon/core/textile/hub"
	"github.com/FleekHQ/space-daemon/log"
	crypto "github.com/libp2p/go-libp2p-crypto"
	tc "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/db"
	nc "github.com/textileio/go-threads/net/api/client"
	bucketsproto "github.com/textileio/textile/v2/api/bucketsd/pb"
	"github.com/textileio/textile/v2/api/common"
	"github.com/textileio/textile/v2/cmd"
	"golang.org/x/crypto/pbkdf2"
)

func CastDbIDToString(dbID thread.ID) string {
	bytes := dbID.Bytes()
	return base32.StdEncoding.EncodeToString(bytes)
}

func ParseDbIDFromString(dbID string) (*thread.ID, error) {
	bytes, err := base32.StdEncoding.DecodeString(dbID)
	if err != nil {
		return nil, err
	}
	id, err := thread.Cast(bytes)
	if err != nil {
		return nil, err
	}

	return &id, nil
}

type DeterministicThreadVariant string

const (
	MetathreadThreadVariant DeterministicThreadVariant = "metathread"
	MirrorBucketVariant     DeterministicThreadVariant = "mirror_bucket"
)

func MirrorBucketVariantGen(mirrorBucketSlug string) DeterministicThreadVariant {
	return DeterministicThreadVariant(string(MirrorBucketVariant) + ":" + mirrorBucketSlug)
}

func NewDeterministicThreadID(kc keychain.Keychain, threadVariant DeterministicThreadVariant) (thread.ID, error) {
	size := 32
	variant := thread.Raw

	priv, _, err := kc.GetStoredKeyPairInLibP2PFormat()
	if err != nil {
		return thread.ID([]byte{}), err
	}

	privInBytes, err := priv.Raw()
	if err != nil {
		return thread.ID([]byte{}), err
	}

	// Do a key derivation based on the private key, a constant nonce, and the thread variant
	num := pbkdf2.Key(privInBytes, []byte("threadID"+threadVariant), 256, size, sha512.New)
	if err != nil {
		return thread.ID([]byte{}), err
	}

	// The following code just concats the key derived from the private key (num)
	// with some constants such as the thread version and the textile thread variant
	numlen := len(num)
	// two 8 bytes (max) numbers plus num
	buf := make([]byte, 2*binary.MaxVarintLen64+numlen)
	n := binary.PutUvarint(buf, thread.V1)
	n += binary.PutUvarint(buf[n:], uint64(variant))
	cn := copy(buf[n:], num)
	if cn != numlen {
		panic("copy length is inconsistent")
	}

	return thread.ID(buf[:n+numlen]), nil
}

func getThreadName(userPubKey []byte, bucketSlug string) string {
	return hex.EncodeToString(userPubKey) + "-" + bucketSlug
}

// Readies a context to access a thread given its name and dbid
func GetThreadContext(parentCtx context.Context, threadName string, dbID thread.ID, hub bool, kc keychain.Keychain, hubAuth hub.HubAuth, threadsClient *tc.Client) (context.Context, error) {
	var err error
	ctx := parentCtx

	// Some threads will be on the hub and some will be local, this flag lets you specify
	// where it is
	if hub {
		ctx, err = hubAuth.GetHubContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	var publicKey crypto.PubKey
	var privKey crypto.PrivKey
	if privKey, publicKey, err = kc.GetStoredKeyPairInLibP2PFormat(); err != nil {
		return nil, err
	}

	var pubKeyInBytes []byte
	if pubKeyInBytes, err = publicKey.Bytes(); err != nil {
		return nil, err
	}

	if threadsClient != nil {
		tok, err := threadsClient.GetToken(ctx, thread.NewLibp2pIdentity(privKey))
		if err != nil {
			return nil, err
		}

		ctx = thread.NewTokenContext(ctx, tok)
	}

	ctx = common.NewThreadNameContext(ctx, getThreadName(pubKeyInBytes, threadName))
	ctx = common.NewThreadIDContext(ctx, dbID)

	return ctx, nil
}

// randBytes returns random bytes in a byte slice of size.
func RandBytes(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	return b, err
}

var metaFileNames = map[string]bool{
	".textileseed": true,
	".textile":     true,
	".DS_Store":    true,
	".Trashes":     true,
	".localized":   true,
}

func IsMetaFileName(pathOrName string) bool {
	_, name := filepath.Split(pathOrName)

	return metaFileNames[name]
}

const threadIDStoreKey = "thread_id"

// Returns the store key for a thread ID. It uses the keychain to obtain the public key, since the store key depends on it.
func getDeterministicthreadStoreKey(kc keychain.Keychain, variant DeterministicThreadVariant) ([]byte, error) {
	pub, err := kc.GetStoredPublicKey()
	if err != nil {
		return nil, err
	}

	pubInBytes, err := pub.Raw()
	if err != nil {
		return nil, err
	}

	result := []byte(threadIDStoreKey + "_" + string(variant))
	result = append(result, pubInBytes...)

	return result, nil
}

// Finds or creates a thread that's based on the user private key and the specified variant
// Using the same private key, variant and thread name will always end up generating the same key
func FindOrCreateDeterministicThread(
	ctx context.Context,
	variant DeterministicThreadVariant,
	threadName string,
	kc keychain.Keychain,
	st store.Store,
	threads *tc.Client,
	cfg config.Config,
	netc *nc.Client,
	hnetc *nc.Client,
	hubAuth hub.HubAuth,
	shouldForceRestore bool,
	dbCollectionConfigs []db.CollectionConfig,
) (*thread.ID, error) {
	storeKey, err := getDeterministicthreadStoreKey(kc, variant)
	if err != nil {
		return nil, err
	}

	pk, _, err := kc.GetStoredKeyPairInLibP2PFormat()
	if err != nil {
		return nil, err
	}

	if val, _ := st.Get(storeKey); val != nil {
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
	dbID, err := NewDeterministicThreadID(kc, variant)
	if err != nil {
		return nil, err
	}

	dbIDInBytes := dbID.Bytes()

	managedKey, err := kc.GetManagedThreadKey(threadName)
	if err != nil {
		return nil, err
	}

	threadCtx, err := GetThreadContext(ctx, threadName, dbID, false, kc, nil, threads)
	if err != nil {
		return nil, err
	}

	hubmaStr := cfg.GetString(config.TextileHubMa, "")
	hubma := cmd.AddrFromStr(hubmaStr)

	hubmaWithThreadID := hubmaStr + "/thread/" + dbID.String()

	hubCtx, err := hubAuth.GetHubContext(ctx)
	if err != nil {
		return nil, err
	}

	_, err = hnetc.GetThread(hubCtx, dbID)
	replThreadExists := err == nil
	if !replThreadExists && shouldForceRestore {
		return nil, err
	}
	if replThreadExists {
		// Try to join remote db in case it was already replicated
		err = threads.NewDBFromAddr(
			threadCtx,
			cmd.AddrFromStr(hubmaWithThreadID),
			managedKey,
			db.WithNewManagedBackfillBlock(true),
			db.WithNewManagedThreadKey(managedKey),
			db.WithNewManagedName(threadName),
			db.WithNewManagedLogKey(pk),
			db.WithNewManagedCollections(
				dbCollectionConfigs...,
			),
		)
		if err == nil || err.Error() == "rpc error: code = Unknown desc = db already exists" || err.Error() == "rpc error: code = Unknown desc = log already exists" {
			return successfulThreadCreation(st, &dbID, dbIDInBytes, storeKey)
		} else if shouldForceRestore == true {
			log.Error("Textile threads require forced restore but there was a restoration issue", err)
			return nil, err
		}
	}

	err = threads.NewDB(threadCtx, dbID, db.WithNewManagedLogKey(pk), db.WithNewManagedThreadKey(managedKey), db.WithNewManagedName(threadName))
	if err != nil && err.Error() != "rpc error: code = Unknown desc = db already exists" {
		return nil, err
	}

	if _, err := netc.AddReplicator(threadCtx, dbID, hubma); err == nil {
		return successfulThreadCreation(st, &dbID, dbIDInBytes, storeKey)
	} else {
		log.Error("error while replicating metathread", err)
	}

	return &dbID, nil
}

func successfulThreadCreation(st store.Store, dbID *thread.ID, dbIDInBytes, storeKey []byte) (*thread.ID, error) {
	if err := st.Set(storeKey, dbIDInBytes); err != nil {
		newErr := errors.New("error while storing thread id: check your local space db accessibility")
		return nil, newErr
	}

	return dbID, nil
}

func MapDirEntryToFileInfo(entry bucketsproto.ListPathResponse, itemPath string) domain.FileInfo {
	item := entry.Item
	info := domain.FileInfo{
		DirEntry: domain.DirEntry{
			Path:          itemPath,
			IsDir:         item.IsDir,
			Name:          item.Name,
			SizeInBytes:   strconv.FormatInt(item.Size, 10),
			FileExtension: strings.Replace(filepath.Ext(item.Name), ".", "", -1),
			// FIXME: real created at needed
			Created: time.Unix(0, item.Metadata.UpdatedAt).Format(time.RFC3339),
			Updated: time.Unix(0, item.Metadata.UpdatedAt).Format(time.RFC3339),
			Members: []domain.Member{},
		},
		IpfsHash:          item.Cid,
		BackedUp:          false,
		LocallyAvailable:  false,
		BackupInProgress:  false,
		RestoreInProgress: false,
	}

	return info
}
