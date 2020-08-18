package keychain

import (
	"crypto/sha512"
	"errors"

	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/pbkdf2"
)

type generateKeyFromMnemonicOpts struct {
	override bool
	mnemonic string
	password string
}

var defaultMnemonicOpts = generateKeyFromMnemonicOpts{
	override: false,
	mnemonic: "",
	password: "",
}

type GenerateKeyFromMnemonicOpts func(o *generateKeyFromMnemonicOpts)

func WithMnemonic(mnemonic string) GenerateKeyFromMnemonicOpts {
	return func(o *generateKeyFromMnemonicOpts) {
		if mnemonic != "" {
			o.mnemonic = mnemonic
		}
	}
}

func WithPassword(password string) GenerateKeyFromMnemonicOpts {
	return func(o *generateKeyFromMnemonicOpts) {
		if password != "" {
			o.password = password
		}
	}
}

func WithOverride() GenerateKeyFromMnemonicOpts {
	return func(o *generateKeyFromMnemonicOpts) {
		o.override = true
	}
}

// Generates a public/private key pair using ed25519 algorithm.
// It stores it in the local db and returns the mnemonic.
// If Mnemonic is a blank string, it generates a random one.
// If there's already a key pair stored, it overrides it if override is set to true. Returns an error otherwise
func (kc *keychain) GenerateKeyFromMnemonic(opts ...GenerateKeyFromMnemonicOpts) (string, error) {
	o := defaultMnemonicOpts
	for _, opt := range opts {
		opt(&o)
	}
	if val, _ := kc.GetStoredPublicKey(); val != nil && o.override == false {
		newErr := errors.New("Error while executing GenerateKeyFromMnemonic. Key pair already exists.")
		return "", newErr
	}

	mnemonic := o.mnemonic

	if mnemonic == "" {
		entropy, err := bip39.NewEntropy(256)
		if err != nil {
			return "", err
		}

		mnemonic, err = bip39.NewMnemonic(entropy)
		if err != nil {
			return "", err
		}
	}

	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, o.password)
	if err != nil {
		return "", err
	}

	// The seed returned by bip39 is fixed to size = 64 bytes.
	// However the seed in ed25519 needs to have size 32.
	// So to fix this we derive a key again based on the previous one, but with the correct size.
	compressedSeed := pbkdf2.Key(seed, []byte("iter2"+o.password), 512, 32, sha512.New)

	_, _, err = kc.generateAndStoreKeyPair(compressedSeed, mnemonic)
	if err != nil {
		return "", err
	}

	return mnemonic, nil
}
