package keychain

import (
	"errors"

	"github.com/99designs/keyring"
	"github.com/FleekHQ/space-daemon/core/permissions"
)

const appTokenStoreKey = "appToken"

func (kc *keychain) StoreAppToken(tok *permissions.AppToken) error {
	ring, err := kc.getKeyRing()
	if err != nil {
		return err
	}

	// Prevent overriding existing master key
	key, _ := kc.st.Get(getMasterTokenStKey())
	if key != nil && tok.IsMaster {
		return errors.New("master key already exists")
	}

	err = ring.Set(keyring.Item{
		Key:   tok.Key,
		Data:  []byte(permissions.MarshalFullToken(tok)),
		Label: "Space App - App Token",
	})
	if err != nil {
		return err
	}

	if tok.IsMaster {
		if err := kc.st.Set(getMasterTokenStKey(), []byte(tok.Key)); err != nil {
			return err
		}
	}

	return nil
}

func (kc *keychain) GetAppToken(key string) (*permissions.AppToken, error) {
	ring, err := kc.getKeyRing()
	if err != nil {
		return nil, err
	}

	token, err := ring.Get(appTokenStoreKey + "_" + key)
	if err != nil {
		return nil, err
	}

	return permissions.UnmarshalFullToken(string(token.Data))
}

func getMasterTokenStKey() []byte {
	return []byte(appTokenStoreKey + "_master")
}
