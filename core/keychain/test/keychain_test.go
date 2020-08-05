package keychain_test

import (
	"testing"

	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/mocks"
	"github.com/stretchr/testify/mock"
)

var (
	mockStore *mocks.Store
)

func initTestKeychain(t *testing.T) keychain.Keychain {
	mockStore = new(mocks.Store)
	mockStore.On("IsOpen").Return(true)

	kc := keychain.New(mockStore)

	return kc
}

func TestKeychain_GenerateAndRestore(t *testing.T) {
	kc := initTestKeychain(t)

	mockStore.On("Set", mock.Anything, mock.Anything).Return(nil)
	pub, priv, _ := kc.GenerateKeyPairWithForce()

	mockStore.On("Get", []byte(keychain.PublicKeyStoreKey)).Return(pub, nil)
	mockStore.On("Get", []byte(keychain.PrivateKeyStoreKey)).Return(priv, nil)

	libp2pPriv, _, _ := kc.GetStoredKeyPairInLibP2PFormat()

	// Reset mock store for assertions
	kc = initTestKeychain(t)
	mockStore.AssertNotCalled(t, "Set", []byte(keychain.PublicKeyStoreKey), pub)
	mockStore.On("Set", mock.Anything, mock.Anything).Return(nil)

	kc.ImportExistingKeyPair(libp2pPriv)

	mockStore.AssertCalled(t, "Set", []byte(keychain.PublicKeyStoreKey), pub)
	mockStore.AssertCalled(t, "Set", []byte(keychain.PrivateKeyStoreKey), priv)
}
