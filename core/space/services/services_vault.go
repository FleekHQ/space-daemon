package services

import (
	"context"
	"encoding/hex"

	"github.com/FleekHQ/space-daemon/core/backup"
	"github.com/libp2p/go-libp2p-core/crypto"
)

// Creates an obfuscated local file that contains everything needed to restore the state from this or another device
func (s *Space) CreateLocalKeysBackup(ctx context.Context, path string) error {
	priv, _, err := s.keychain.GetStoredKeyPairInLibP2PFormat()
	if err != nil {
		return err
	}

	privInBytes, err := priv.Raw()
	if err != nil {
		return err
	}

	textileState, err := s.tc.SerializeState(ctx)
	if err != nil {
		return err
	}

	b := &backup.Backup{
		PrivateKey:          hex.EncodeToString(privInBytes),
		TextileClientBackup: hex.EncodeToString(textileState),
	}

	if err := backup.MarshalBackup(path, b); err != nil {
		return err
	}

	return nil
}

// Restores the state by receiving the path to a local backup
// Warning: This will delete any local state before restoring the backup
func (s *Space) RecoverKeysByLocalBackup(ctx context.Context, path string) error {
	// Retrieve the backup
	b, err := backup.UnmarshalBackup(path)
	if err != nil {
		return err
	}

	privInBytes, err := hex.DecodeString(b.PrivateKey)
	if err != nil {
		return err
	}

	textileClientStateInBytes, err := hex.DecodeString(b.TextileClientBackup)
	if err != nil {
		return err
	}

	// Restore keychain
	priv, err := crypto.UnmarshalEd25519PrivateKey(privInBytes)
	if err != nil {
		return err
	}
	if err := s.keychain.ImportExistingKeyPair(priv); err != nil {
		return err
	}

	// Restore Textile Client state
	if err := s.tc.RestoreState(ctx, textileClientStateInBytes); err != nil {
		return err
	}

	return nil
}
