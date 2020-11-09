package keychain

import (
	"errors"

	"github.com/99designs/keyring"
	"github.com/FleekHQ/space-daemon/core/permissions"
)

const AppTokenStoreKey = "appToken"
const MasterAppTokenStoreKey = "masterAppToken"

var ErrMasterTokenAlreadyExists = errors.New("master app token already exists")

func (kc *keychain) StoreAppToken(tok *permissions.AppToken) error {
	ring, err := kc.getKeyRing()
	if err != nil {
		return err
	}

	// Prevent overriding existing master key
	key, _ := kc.st.Get([]byte(getMasterTokenStKey()))
	if key != nil && tok.IsMaster {
		return ErrMasterTokenAlreadyExists
	}

	// Prevents overriding even if user logged out and logged back in (which clears the store)
	_, err = ring.Get(getMasterTokenStKey())
	if err == nil && tok.IsMaster {
		return ErrMasterTokenAlreadyExists
	}

	err = ring.Set(keyring.Item{
		Key:   AppTokenStoreKey + "_" + tok.Key,
		Data:  []byte(permissions.MarshalFullToken(tok)),
		Label: "Space App - App Token",
	})
	if err != nil {
		return err
	}

	if tok.IsMaster {
		if err := kc.st.Set([]byte(getMasterTokenStKey()), []byte(tok.Key)); err != nil {
			return err
		}

		if err := ring.Set(keyring.Item{
			Key:   getMasterTokenStKey(),
			Data:  []byte(permissions.MarshalFullToken(tok)),
			Label: "Space App - Master App Token",
		}); err != nil {
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

	token, err := ring.Get(AppTokenStoreKey + "_" + key)
	if err != nil {
		return nil, err
	}

	return permissions.UnmarshalFullToken(string(token.Data))
}

func getMasterTokenStKey() string {
	return AppTokenStoreKey + "_" + MasterAppTokenStoreKey
}
