package services

import (
	"context"
	"encoding/hex"

	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/core/textile/hub"
)

func (s *Space) GenerateKeyPair(ctx context.Context, useForce bool) (domain.KeyPair, error) {
	if useForce {
		return s.generateKeyPairWithForce(ctx)
	}

	kc := keychain.New(s.store)
	if pub, priv, err := kc.GenerateKeyPair(); err != nil {
		return domain.KeyPair{}, err
	} else {

		return domain.KeyPair{
			PublicKey:  hex.EncodeToString(pub),
			PrivateKey: hex.EncodeToString(priv),
		}, nil
	}
}

func (s *Space) generateKeyPairWithForce(ctx context.Context) (domain.KeyPair, error) {
	kc := keychain.New(s.store)
	if pub, priv, err := kc.GenerateKeyPairWithForce(); err != nil {
		return domain.KeyPair{}, err
	} else {

		return domain.KeyPair{
			PublicKey:  hex.EncodeToString(pub),
			PrivateKey: hex.EncodeToString(priv),
		}, nil
	}
}

func (s *Space) GetPublicKey(ctx context.Context) (string, error) {
	_, pub, err := s.keychain.GetStoredKeyPairInLibP2PFormat()
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
	return hub.GetHubToken(ctx, s.store, s.cfg)
}
