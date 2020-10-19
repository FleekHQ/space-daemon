package services

import (
	"context"
	"encoding/hex"
	"errors"

	"github.com/FleekHQ/space-daemon/core/keychain"
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
	if err != nil {
		return err
	}

	return nil
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
	tokens, err := s.hub.GetTokensWithCache(ctx)
	if err != nil {
		return "", err
	}

	return tokens.HubToken, nil
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

func (s *Space) DeleteKeypair(ctx context.Context) error {
	err := s.waitForTextileInit(ctx)
	if err != nil {
		return err
	}

	if err := s.keychain.DeleteKeypair(); err != nil {
		return err
	}

	// Tell the textile client to stop operations
	if err := s.tc.RemoveKeys(); err != nil {
		return err
	}

	// Clear badger store
	if err := s.store.DropAll(); err != nil {
		return err
	}

	return nil
}
