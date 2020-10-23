package keychain

import (
	"crypto/ed25519"
	"crypto/sha512"
	"encoding/hex"
	"os"
	"path"
	"strings"

	"golang.org/x/crypto/pbkdf2"

	"errors"

	"github.com/99designs/keyring"
	ri "github.com/FleekHQ/space-daemon/core/keychain/keyring"
	"github.com/FleekHQ/space-daemon/core/store"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/textileio/go-threads/core/thread"
)

const PrivateKeyStoreKey = "key"
const PublicKeyStoreKey = "pub"

const privKeyMnemonicSeparator = "___"

var (
	ErrKeyPairNotFound = errors.New("No key pair found in the local db.")
)

type keychain struct {
	fileDir string
	st      store.Store
	ring    ri.Keyring
	privKey *crypto.PrivKey
}

type Keychain interface {
	GenerateKeyPair() (pub []byte, priv []byte, err error)
	GenerateKeyFromMnemonic(...GenerateKeyFromMnemonicOpts) (mnemonic string, err error)
	GetStoredKeyPairInLibP2PFormat() (crypto.PrivKey, crypto.PubKey, error)
	GetStoredPublicKey() (crypto.PubKey, error)
	GetStoredMnemonic() (string, error)
	GetManagedThreadKey(threadKeyName string) (thread.Key, error)
	GenerateKeyPairWithForce() (pub []byte, priv []byte, err error)
	Sign([]byte) ([]byte, error)
	ImportExistingKeyPair(priv crypto.PrivKey, mnemonic string) error
	DeleteKeypair() error
}

type keychainOptions struct {
	fileDir string
	store   store.Store

	// Don't use kc.ring directly, use getKeyRing() instead
	ring ri.Keyring
}

var defaultKeychainOptions = keychainOptions{
	fileDir: store.DefaultRootDir,
}

// Helper function for setting keychain file path for Windows/Linux
func WithPath(path string) Option {
	return func(o *keychainOptions) {
		if path != "" {
			o.fileDir = path
		}
	}
}

func WithStore(st store.Store) Option {
	return func(o *keychainOptions) {
		if st != nil {
			o.store = st
		}
	}
}

// Used to inject a mock keyring in tests or in case you want to use a custom keyring implementation
func WithKeyring(ring ri.Keyring) Option {
	return func(o *keychainOptions) {
		if ring != nil {
			o.ring = ring
		}
	}
}

type Option func(o *keychainOptions)

func New(opts ...Option) *keychain {
	o := defaultKeychainOptions
	for _, opt := range opts {
		opt(&o)
	}

	if o.store == nil {
		defaultStore := store.New(store.WithPath(o.fileDir))
		o.store = defaultStore
	}

	return &keychain{
		fileDir: o.fileDir,
		st:      o.store,
		ring:    o.ring,
	}
}

// Generates a public/private key pair using ed25519 algorithm.
// It stores it in the local db and returns the key pair key.
// If there's already a key pair stored, it returns an error.
// Use GenerateKeyPairWithForce if you want to override existing keys
func (kc *keychain) GenerateKeyPair() ([]byte, []byte, error) {
	if val, _ := kc.GetStoredPublicKey(); val != nil {
		newErr := errors.New("Error while executing GenerateKeyPair. Key pair already exists. Use GenerateKeyPairWithForce if you want to override it.")
		return nil, nil, newErr
	}

	return kc.generateAndStoreKeyPair(nil, "")
}

// Returns the stored key pair using the same signature than libp2p's GenerateEd25519Key function
func (kc *keychain) GetStoredKeyPairInLibP2PFormat() (crypto.PrivKey, crypto.PubKey, error) {
	var priv []byte
	var err error

	if kc.privKey != nil {
		return *kc.privKey, (*kc.privKey).GetPublic(), nil
	}

	if priv, _, err = kc.retrieveKeyPair(); err != nil {
		newErr := ErrKeyPairNotFound
		return nil, nil, newErr
	}

	var unmarshalledPriv crypto.PrivKey

	if unmarshalledPriv, err = crypto.UnmarshalEd25519PrivateKey(priv); err != nil {
		return nil, nil, err
	}

	kc.privKey = &unmarshalledPriv

	unmarshalledPub := unmarshalledPriv.GetPublic()

	return unmarshalledPriv, unmarshalledPub, nil
}

// Generates a public/private key pair using ed25519 algorithm.
// It stores it in the local db and returns the key pair.
// Warning: If there's already a key pair stored, it overrides it.
func (kc *keychain) GenerateKeyPairWithForce() ([]byte, []byte, error) {
	return kc.generateAndStoreKeyPair(nil, "")
}

// Returns the public key currently in use in LibP2P format.
// Returns an error if there's no public key set.
// Unlike GetStoredKeyPairInLibP2PFormat, this method does not access the keychain
func (kc *keychain) GetStoredPublicKey() (crypto.PubKey, error) {
	ring, err := kc.getKeyRing()
	if err != nil {
		return nil, err
	}
	_, err = ring.GetMetadata(PrivateKeyStoreKey)
	if err == keyring.ErrKeyNotFound {
		return nil, ErrKeyPairNotFound
	}

	pubInBytes, err := kc.st.Get([]byte(PublicKeyStoreKey))
	if err != nil {
		return nil, err
	}

	if pubInBytes == nil {
		return nil, ErrKeyPairNotFound
	}

	pub, err := crypto.UnmarshalEd25519PublicKey(pubInBytes)
	if err != nil {
		return nil, err
	}

	return pub, nil
}

func (kc *keychain) GetStoredMnemonic() (string, error) {
	_, mnemonic, err := kc.retrieveKeyPair()
	if err != nil {
		return "", err
	}

	return mnemonic, nil
}

// Stores an existing private key in the keychain
// Warning: If there's already a key pair stored, this will override it.
func (kc *keychain) ImportExistingKeyPair(priv crypto.PrivKey, mnemonic string) error {
	privInBytes, err := priv.Raw()
	if err != nil {
		return err
	}
	pubInBytes, err := priv.GetPublic().Raw()
	if err != nil {
		return err
	}

	// Store the key pair in the db
	if err := kc.storeKeyPair(privInBytes, pubInBytes, mnemonic); err != nil {
		return err
	}

	kc.privKey = &priv

	return nil
}

func (kc *keychain) DeleteKeypair() error {
	ring, err := kc.getKeyRing()
	if err != nil {
		return err
	}

	// Note: currently ignoring error on keychain removal because it's failing randomly.
	// Use GenerateKeyPair with override option instead.
	err = ring.Remove(PrivateKeyStoreKey)
	if err != nil {
		log.Error("Error removing keychaing from keyring", err)
	}

	err = kc.st.Remove([]byte(PublicKeyStoreKey))
	if err != nil {
		return err
	}

	kc.privKey = nil
	return nil
}

func (kc *keychain) generateKeyPair(seed []byte) ([]byte, []byte, error) {
	if seed != nil {
		priv := ed25519.NewKeyFromSeed(seed)
		publicKey := priv.Public()
		pub, ok := publicKey.(ed25519.PublicKey)
		if !ok {
			return nil, nil, errors.New("Error while generating key pair from seed")
		}
		return pub, priv, nil
	}
	// Compute the key from a random seed
	pub, priv, err := ed25519.GenerateKey(nil)
	return pub, priv, err
}

func (kc *keychain) generateAndStoreKeyPair(seed []byte, mnemonic string) ([]byte, []byte, error) {
	// Compute the key from a random seed
	pub, priv, err := kc.generateKeyPair(seed)

	if err != nil {
		return nil, nil, err
	}

	// Store the key pair in the db
	if err := kc.storeKeyPair(priv, pub, mnemonic); err != nil {
		return nil, nil, err
	}

	privkey, err := crypto.UnmarshalEd25519PrivateKey(priv)
	if err != nil {
		log.Warn("Unable to cache priv key")
	}

	kc.privKey = &privkey

	return pub, priv, nil
}

// Signs a message using the stored private key.
// Returns an error if the private key cannot be found.
func (kc *keychain) Sign(message []byte) ([]byte, error) {
	if priv, _, err := kc.retrieveKeyPair(); err != nil {
		return nil, err
	} else {
		signedBytes := ed25519.Sign(priv, message)
		return signedBytes, nil
	}
}

func (kc *keychain) getKeyRing() (ri.Keyring, error) {
	if kc.ring != nil {
		return kc.ring, nil
	}

	ucd, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	return keyring.Open(keyring.Config{
		ServiceName: "space",

		// MacOS keychain
		KeychainTrustApplication:       true,
		KeychainAccessibleWhenUnlocked: true,

		// KDE Wallet
		KWalletAppID:  "space",
		KWalletFolder: "space",

		// Windows
		WinCredPrefix: "space",

		// freedesktop.org's Secret Service
		LibSecretCollectionName: "space",

		// Pass (https://www.passwordstore.org/)
		PassPrefix: "space",
		PassDir:    kc.fileDir + "/kcpw",

		// Fallback encrypted file
		FileDir: path.Join(ucd, "space", "keyring"),
	})
}

func (kc *keychain) storeKeyPair(privKey []byte, pubKey []byte, mnemonic string) error {
	ring, err := kc.getKeyRing()
	if err != nil {
		return err
	}

	privAsHex := hex.EncodeToString(privKey)
	privWithMnemonic := privAsHex + privKeyMnemonicSeparator + mnemonic

	// Store private key together with mnemonic
	// Priv key is stored as 0x1234...890___some mnemonic
	// The idea behind storing them together is that we avoid asking for keychain access twice
	if err := ring.Set(keyring.Item{
		Key:   PrivateKeyStoreKey,
		Data:  []byte(privWithMnemonic),
		Label: "Space App",
	}); err != nil {
		return err
	}

	// Store pub key outside of the key ring for quick access
	if err := kc.st.Set([]byte(PublicKeyStoreKey), pubKey); err != nil {
		return err
	}

	return nil
}

func (kc *keychain) retrieveKeyPair() (privKey []byte, mnemonic string, err error) {
	ring, err := kc.getKeyRing()
	if err != nil {
		return nil, "", err
	}

	privKeyItem, err := ring.Get(PrivateKeyStoreKey)
	if err != nil {
		return nil, "", err
	}

	// Priv key is stored as 0x1234...890___some mnemonic
	// Here we split it to return priv key and mnemonic separately
	privKeyAsStr := string(privKeyItem.Data)
	privKeyParts := strings.Split(privKeyAsStr, privKeyMnemonicSeparator)
	mnemonic = privKeyParts[1]
	privKey, err = hex.DecodeString(privKeyParts[0])
	if err != nil {
		return nil, "", err
	}

	return privKey, mnemonic, nil
}

func (kc *keychain) GetManagedThreadKey(threadKeyName string) (thread.Key, error) {
	size := 32

	priv, _, err := kc.GetStoredKeyPairInLibP2PFormat()
	if err != nil {
		return thread.Key{}, err
	}

	privBytes, err := priv.Raw()
	if err != nil {
		return thread.Key{}, err
	}

	num := pbkdf2.Key(privBytes, []byte("threadKey"+threadKeyName), 256, size, sha512.New)
	if err != nil {
		return thread.Key{}, err
	}

	managedKey, err := thread.KeyFromBytes(num)
	if err != nil {
		return thread.Key{}, err
	}

	return managedKey, nil
}
