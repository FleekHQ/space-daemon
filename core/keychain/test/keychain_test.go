package keychain_test

import (
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/mocks"
	"github.com/stretchr/testify/assert"
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

func TestKeychain_GenerateMnemonicKey(t *testing.T) {
	kc := initTestKeychain(t)

	mockStore.On("Set", mock.Anything, mock.Anything).Return(nil)
	mockStore.On("Get", []byte(keychain.PublicKeyStoreKey)).Return(nil, nil)

	val, err := kc.GenerateKeyFromMnemonic()
	words := strings.Split(val, " ")

	assert.Nil(t, err)
	assert.NotNil(t, val)
	assert.Equal(t, 24, len(words))
	mockStore.AssertCalled(t, "Set", []byte(keychain.PublicKeyStoreKey), mock.Anything)
	mockStore.AssertCalled(t, "Set", []byte(keychain.PrivateKeyStoreKey), mock.Anything)
}

func TestKeychain_RestoreMnemonicKey(t *testing.T) {
	kc := initTestKeychain(t)

	mockStore.On("Set", mock.Anything, mock.Anything).Return(nil)
	mockStore.On("Get", []byte(keychain.PublicKeyStoreKey)).Return(nil, nil)

	mnemonic := "pioneer powder icon lemon pulse struggle title jealous stamp sausage interest govern fault pumpkin fever glove dust buzz skin diesel purse answer pitch cave"
	pubFromMnemonic, _ := hex.DecodeString("401e174cc48028539a1bc687aaf7aae1bb5320bd3a48a5d69733b211ad88e93d")
	privFromMnemonic, _ := hex.DecodeString("d7680cca9c019e96b1f0371d504772eb803e1d63c90ad6074ba410400ee7de20401e174cc48028539a1bc687aaf7aae1bb5320bd3a48a5d69733b211ad88e93d")

	val, err := kc.GenerateKeyFromMnemonic(keychain.WithMnemonic(mnemonic))
	assert.Nil(t, err)
	assert.NotNil(t, val)
	assert.Equal(t, mnemonic, val)
	mockStore.AssertCalled(t, "Set", []byte(keychain.PublicKeyStoreKey), pubFromMnemonic)
	mockStore.AssertCalled(t, "Set", []byte(keychain.PrivateKeyStoreKey), privFromMnemonic)
}

func TestKeychain_RestoreMnemonicKeyOnOverrideErr(t *testing.T) {
	kc := initTestKeychain(t)

	mockStore.On("Set", mock.Anything, mock.Anything).Return(nil)

	mnemonic := "pioneer powder icon lemon pulse struggle title jealous stamp sausage interest govern fault pumpkin fever glove dust buzz skin diesel purse answer pitch cave"
	pubFromMnemonic, _ := hex.DecodeString("401e174cc48028539a1bc687aaf7aae1bb5320bd3a48a5d69733b211ad88e93d")

	mockStore.On("Get", []byte(keychain.PublicKeyStoreKey)).Return(pubFromMnemonic, nil)

	_, err := kc.GenerateKeyFromMnemonic(keychain.WithMnemonic(mnemonic))
	assert.NotNil(t, err)
	assert.Equal(t, errors.New("Error while executing GenerateKeyFromMnemonic. Key pair already exists."), err)
}

func TestKeychain_RestoreMnemonicKeyOnOverrideSuccess(t *testing.T) {
	kc := initTestKeychain(t)

	mockStore.On("Set", mock.Anything, mock.Anything).Return(nil)

	mnemonic := "pioneer powder icon lemon pulse struggle title jealous stamp sausage interest govern fault pumpkin fever glove dust buzz skin diesel purse answer pitch cave"
	pubFromMnemonic, _ := hex.DecodeString("401e174cc48028539a1bc687aaf7aae1bb5320bd3a48a5d69733b211ad88e93d")
	privFromMnemonic, _ := hex.DecodeString("d7680cca9c019e96b1f0371d504772eb803e1d63c90ad6074ba410400ee7de20401e174cc48028539a1bc687aaf7aae1bb5320bd3a48a5d69733b211ad88e93d")

	mockStore.On("Get", []byte(keychain.PublicKeyStoreKey)).Return(pubFromMnemonic, nil)

	val, err := kc.GenerateKeyFromMnemonic(keychain.WithMnemonic(mnemonic), keychain.WithOverride())
	assert.Nil(t, err)
	assert.NotNil(t, val)
	assert.Equal(t, mnemonic, val)
	mockStore.AssertCalled(t, "Set", []byte(keychain.PublicKeyStoreKey), pubFromMnemonic)
	mockStore.AssertCalled(t, "Set", []byte(keychain.PrivateKeyStoreKey), privFromMnemonic)
}
