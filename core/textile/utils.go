package textile

import (
	"encoding/base32"
	"encoding/binary"

	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/store"
	"github.com/textileio/go-threads/core/thread"
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

type deterministicThreadVariant []byte

var (
	metathreadThreadVariant deterministicThreadVariant = []byte{0x15}
)

func newDeterministicThreadID(st *store.Store, threadVariant deterministicThreadVariant) (thread.ID, error) {
	size := 32
	variant := thread.Raw

	kc := keychain.New(*st)

	msg, err := kc.Sign(threadVariant)
	if err != nil {
		return thread.ID([]byte{}), err
	}

	num := make([]byte, size)
	copy(num, msg[:size])

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
