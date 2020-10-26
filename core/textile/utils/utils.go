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

	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/store"
	"github.com/FleekHQ/space-daemon/core/textile/hub"
	crypto "github.com/libp2p/go-libp2p-crypto"
	tc "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/db"
	"github.com/textileio/textile/v2/api/common"
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

func IsMetaFileName(pathOrName string) bool {
	_, name := filepath.Split(pathOrName)

	if name == ".textileseed" || name == ".textile" {
		return true
	}

	return false
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

// Finds or creates a thread ID that's based on the user private key and the specified variant
// Using the same private key, variant and thread name will always end up generating the same key
func FindOrCreateDeterministicThreadID(ctx context.Context, variant DeterministicThreadVariant, threadName string, kc keychain.Keychain, st store.Store, threads *tc.Client) (*thread.ID, error) {
	storeKey, err := getDeterministicthreadStoreKey(kc, variant)
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

	err = threads.NewDB(threadCtx, dbID, db.WithNewManagedThreadKey(managedKey))
	if err != nil && err.Error() != "rpc error: code = Unknown desc = db already exists" {
		return nil, err
	}

	if err := st.Set(storeKey, dbIDInBytes); err != nil {
		newErr := errors.New("error while storing thread id: check your local space db accessibility")
		return nil, newErr
	}

	return &dbID, nil
}
