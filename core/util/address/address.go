package address

import (
	"encoding/hex"

	"github.com/libp2p/go-libp2p-core/crypto"
	"golang.org/x/crypto/sha3"
)

// Returns the address representation of a public key
// If the public key  is malformed it returns an empty string
func DeriveAddress(pubKey crypto.PubKey) string {
	pubBytes, err := pubKey.Raw()
	if err != nil {
		return ""
	}

	hf := sha3.New256()
	hf.Write(pubBytes)

	// Get the hex representation of the SHA3-256 hash
	hexHash := hex.EncodeToString(hf.Sum(nil))

	// Drop the first 14 bytes (28 characters)
	trimmed := hexHash[28:]

	return "0x" + trimmed
}
