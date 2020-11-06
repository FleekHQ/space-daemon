package services

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/FleekHQ/space-daemon/core/backup"
	"github.com/FleekHQ/space-daemon/core/vault"
	"github.com/libp2p/go-libp2p-core/crypto"
)

const separator = "___"

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

	b := &backup.Backup{
		PrivateKey: hex.EncodeToString(privInBytes),
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

	// Restore keychain
	priv, err := crypto.UnmarshalEd25519PrivateKey(privInBytes)
	if err != nil {
		return err
	}
	if err := s.keychain.ImportExistingKeyPair(priv, ""); err != nil {
		return err
	}

	if err := s.tc.RestoreDB(ctx); err != nil {
		s.keychain.DeleteKeypair()
		return err
	}
	return nil
}

// Uses vault service to fetch and decrypt a keypair set
func (s *Space) RecoverKeysByPassphrase(ctx context.Context, uuid string, pass string) error {
	items, err := s.vault.Retrieve(uuid, pass)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		return errors.New("Retrieved vault does not contain keys")
	}

	// TODO: Generalize to N keys
	privAndMnemonic := strings.Split(items[0].Value, separator)
	privInBytes, err := hex.DecodeString(privAndMnemonic[0])
	if err != nil {
		return err
	}

	unmarshalledPriv, err := crypto.UnmarshalEd25519PrivateKey(privInBytes)
	if err != nil {
		return err
	}

	if err := s.keychain.ImportExistingKeyPair(unmarshalledPriv, privAndMnemonic[1]); err != nil {
		return err
	}

	if err := s.tc.RestoreDB(ctx); err != nil {
		s.keychain.DeleteKeypair()
		return err
	}
	return nil
}

// Uses the vault service to securely store the current keypair
func (s *Space) BackupKeysByPassphrase(ctx context.Context, uuid string, pass string, backupType string) error {
	tokens, err := s.GetAPISessionTokens(ctx)
	if err != nil {
		return err
	}

	priv, _, err := s.keychain.GetStoredKeyPairInLibP2PFormat()
	if err != nil {
		return err
	}

	privInBytes, err := priv.Raw()
	if err != nil {
		return err
	}

	mnemonic, err := s.keychain.GetStoredMnemonic()
	if err != nil {
		return err
	}

	// TODO: Generalize to item array once we support multiple keys
	item := vault.VaultItem{
		ItemType: vault.PrivateKeyWithMnemonic,
		Value:    hex.EncodeToString(privInBytes) + separator + mnemonic,
	}

	items := []vault.VaultItem{item}

	if _, err := s.vault.Store(uuid, pass, backupType, tokens.ServicesToken, items); err != nil {
		return err
	}

	return nil
}

// Tests a passphrase without storing anything to check if the passphrase is correct
func (s *Space) TestPassphrase(ctx context.Context, uuid string, pass string) error {
	items, err := s.vault.Retrieve(uuid, pass)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		return errors.New("Retrieved vault does not contain keys")
	}

	return nil
}
