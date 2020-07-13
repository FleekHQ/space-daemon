package keychain

import (
	"crypto/ed25519"

	"errors"

	db "github.com/FleekHQ/space-daemon/core/store"
	"github.com/libp2p/go-libp2p-core/crypto"
)

const PrivateKeyStoreKey = "private_key"
const PublicKeyStoreKey = "public_key"

var (
	ErrKeyPairNotFound = errors.New("No key pair found in the local db.")
)

type keychain struct {
	store db.Store
}

type Keychain interface {
	GenerateKeyPair() ([]byte, []byte, error)
	GetStoredKeyPairInLibP2PFormat() (crypto.PrivKey, crypto.PubKey, error)
	GenerateKeyPairWithForce() ([]byte, []byte, error)
	Sign([]byte) ([]byte, error)
}

func New(store db.Store) *keychain {
	return &keychain{
		store: store,
	}
}

// Generates a public/private key pair using ed25519 algorithm.
// It stores it in the local db and returns the key pair key.
// If there's already a key pair stored, it returns an error.
// Use GenerateKeyPairWithForce if you want to override existing keys
func (kc *keychain) GenerateKeyPair() ([]byte, []byte, error) {
	if val, _ := kc.store.Get([]byte(PublicKeyStoreKey)); val != nil {
		newErr := errors.New("Error while executing GenerateKeyPair. Key pair already exists. Use GenerateKeyPairWithForce if you want to override it.")
		return nil, nil, newErr
	}

	return kc.generateAndStoreKeyPair()
}

// Returns the stored key pair using the same signature than libp2p's GenerateEd25519Key function
func (kc *keychain) GetStoredKeyPairInLibP2PFormat() (crypto.PrivKey, crypto.PubKey, error) {
	var priv []byte
	var pub []byte
	var err error

	if pub, err = kc.store.Get([]byte(PublicKeyStoreKey)); err != nil {
		newErr := ErrKeyPairNotFound
		return nil, nil, newErr
	}

	if priv, err = kc.store.Get([]byte(PrivateKeyStoreKey)); err != nil {
		newErr := ErrKeyPairNotFound
		return nil, nil, newErr
	}

	var unmarshalledPriv crypto.PrivKey
	var unmarshalledPub crypto.PubKey

	if unmarshalledPriv, err = crypto.UnmarshalEd25519PrivateKey(priv); err != nil {
		return nil, nil, err
	}

	if unmarshalledPub, err = crypto.UnmarshalEd25519PublicKey(pub); err != nil {
		return nil, nil, err
	}

	return unmarshalledPriv, unmarshalledPub, nil
}

// Generates a public/private key pair using ed25519 algorithm.
// It stores it in the local db and returns the key pair.
// Warning: If there's already a key pair stored, it overrides it.
func (kc *keychain) GenerateKeyPairWithForce() ([]byte, []byte, error) {
	return kc.generateAndStoreKeyPair()
}

func (kc *keychain) generateAndStoreKeyPair() ([]byte, []byte, error) {
	// Compute the key from a random seed
	pub, priv, err := ed25519.GenerateKey(nil)

	if err != nil {
		return nil, nil, err
	}

	// Store the key pair in the db
	if err = kc.store.Set([]byte(PublicKeyStoreKey), pub); err != nil {
		return nil, nil, err
	}

	if err = kc.store.Set([]byte(PrivateKeyStoreKey), priv); err != nil {
		return nil, nil, err
	}

	return pub, priv, nil
}

// Signs a message using the stored private key.
// Returns an error if the private key cannot be found.
func (kc *keychain) Sign(message []byte) ([]byte, error) {
	if priv, err := kc.store.Get([]byte(PrivateKeyStoreKey)); err != nil {
		return nil, err
	} else {
		signedBytes := ed25519.Sign(priv, message)
		return signedBytes, nil
	}
}
