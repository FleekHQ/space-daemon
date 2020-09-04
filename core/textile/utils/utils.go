package utils

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/base32"
	"encoding/binary"

	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/textileio/go-threads/core/thread"
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

// randBytes returns random bytes in a byte slice of size.
func RandBytes(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	return b, err
}
