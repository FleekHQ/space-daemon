package keychain

import (
	"errors"

	"github.com/tyler-smith/go-bip39"
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
	if val, _ := kc.store.Get([]byte(PublicKeyStoreKey)); val != nil && o.override == false {
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

	seed := bip39.NewSeed(mnemonic, o.password)[:32]

	_, _, err := kc.generateAndStoreKeyPair(seed)
	if err != nil {
		return "", err
	}

	return mnemonic, nil
}
