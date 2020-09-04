package utils

import (
	"context"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"

	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/textile/hub"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/textile/api/common"
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

var (
	MetathreadThreadVariant DeterministicThreadVariant = "metathread"
)

func NewDeterministicThreadID(kc keychain.Keychain, threadVariant DeterministicThreadVariant) (thread.ID, error) {
	size := 32
	variant := thread.Raw

	pub, err := kc.GetStoredPublicKey()
	if err != nil {
		return thread.ID([]byte{}), err
	}

	pubInBytes, err := pub.Raw()
	if err != nil {
		return thread.ID([]byte{}), err
	}

	// Do a key derivation based on the private key, a constant nonce, and the thread variant
	num := pbkdf2.Key(pubInBytes, []byte("threadID"+threadVariant), 256, size, sha512.New)
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
func GetThreadContext(parentCtx context.Context, threadName string, dbID thread.ID, hub bool, kc keychain.Keychain, hubAuth hub.HubAuth) (context.Context, error) {
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
	if publicKey, err = kc.GetStoredPublicKey(); err != nil {
		return nil, err
	}

	var pubKeyInBytes []byte
	if pubKeyInBytes, err = publicKey.Bytes(); err != nil {
		return nil, err
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
