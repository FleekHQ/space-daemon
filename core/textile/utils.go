package textile

import (
	"crypto/sha512"
	"encoding/base32"
	"encoding/binary"

	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/store"
	"github.com/textileio/go-threads/core/thread"
	"golang.org/x/crypto/pbkdf2"
)

func castDbIDToString(dbID thread.ID) string {
	bytes := dbID.Bytes()
	return base32.StdEncoding.EncodeToString(bytes)
}

func parseDbIDFromString(dbID string) (*thread.ID, error) {
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

type deterministicThreadVariant string

var (
	metathreadThreadVariant deterministicThreadVariant = "metathread"
)

func newDeterministicThreadID(st *store.Store, threadVariant deterministicThreadVariant) (thread.ID, error) {
	size := 32
	variant := thread.Raw

	kc := keychain.New(*st)

	priv, _, err := kc.GetStoredKeyPairInLibP2PFormat()
	if err != nil {
		return thread.ID([]byte{}), err
	}

	privInBytes, err := priv.Raw()
	if err != nil {
		return thread.ID([]byte{}), err
	}

	num := pbkdf2.Key(privInBytes, []byte("threadID"+threadVariant), 256, size, sha512.New)
	if err != nil {
		return thread.ID([]byte{}), err
	}

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
