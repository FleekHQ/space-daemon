package services

import (
	"context"
	"encoding/hex"
	"errors"

	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/textile/hub"
	"github.com/textileio/go-threads/core/thread"

)

// Generates a key pair and returns a mnemonic for recovering that key later on
func (s *Space) GenerateKeyPair(ctx context.Context, useForce bool) (string, error) {
	var mnemonic string
	var err error
	if useForce {
		mnemonic, err = s.keychain.GenerateKeyFromMnemonic(keychain.WithOverride())
	} else {
		mnemonic, err = s.keychain.GenerateKeyFromMnemonic()
	}
	if err != nil {
		return "", err
	}

	return mnemonic, nil
}

func (s *Space) RestoreKeyPairFromMnemonic(ctx context.Context, mnemonic string) error {
	_, err := s.keychain.GenerateKeyFromMnemonic(keychain.WithMnemonic(mnemonic), keychain.WithOverride())

	return err
}

func (s *Space) GetPublicKey(ctx context.Context) (string, error) {
	pub, err := s.keychain.GetStoredPublicKey()
	if err != nil {
		return "", err
	}

	publicKeyBytes, err := pub.Raw()
	if err != nil {
		return "", err
	}

	publicKeyHex := hex.EncodeToString(publicKeyBytes)

	return publicKeyHex, nil
}

func (s *Space) GetHubAuthToken(ctx context.Context) (string, error) {
	tokenCtx, err := hub.GetHubToken(ctx, s.store, s.keychain, s.cfg)

	if err != nil {
		return "", err
	}

	threadToken, ok := thread.TokenFromContext(tokenCtx)

	if ok {
		return string(threadToken), nil;
	}

	return "", nil;
}

func (s *Space) GetMnemonic(ctx context.Context) (string, error) {
	mnemonic, err := s.keychain.GetStoredMnemonic()
	if err != nil {
		return "", err
	}

	if mnemonic == "" {
		return "", errors.New("No mnemonic seed stored in the keychain")
	}

	return mnemonic, nil
}
