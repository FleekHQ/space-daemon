package services

import (
	"context"
	"encoding/hex"
	"github.com/FleekHQ/space-poc/core/keychain"
	"github.com/FleekHQ/space-poc/core/space/domain"
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