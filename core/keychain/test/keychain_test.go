package keychain_test

import (
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/99designs/keyring"
	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tyler-smith/go-bip39"
)

var (
	mockStore   *mocks.Store
	mockKeyRing *mocks.Keyring
)

func initTestKeychain(t *testing.T) keychain.Keychain {
	mockStore = new(mocks.Store)
	mockStore.On("IsOpen").Return(true)

	mockKeyRing = new(mocks.Keyring)

	kc := keychain.New(keychain.WithStore(mockStore), keychain.WithKeyring(mockKeyRing))

	return kc
}

func TestKeychain_GenerateAndRestore(t *testing.T) {
	kc := initTestKeychain(t)

	mockStore.On("Set", mock.Anything, mock.Anything).Return(nil)
	mockKeyRing.On("Set", mock.Anything).Return(nil)
	pub, priv, _ := kc.GenerateKeyPairWithForce()

	privKeyItem := keyring.Item{
		Key:   keychain.PrivateKeyStoreKey,
		Data:  []byte(hex.EncodeToString(priv) + "___"),
		Label: "Space App",
	}

	mockStore.On("Get", []byte(keychain.PublicKeyStoreKey)).Return(pub, nil)
	mockKeyRing.On("Get", keychain.PrivateKeyStoreKey).Return(privKeyItem, nil)
	mockKeyRing.On("GetMetadata", mock.Anything).Return(keyring.Metadata{}, nil)

	libp2pPriv, _, _ := kc.GetStoredKeyPairInLibP2PFormat()

	// Reset mock store for assertions
	kc = initTestKeychain(t)
	mockStore.AssertNotCalled(t, "Set", []byte(keychain.PublicKeyStoreKey), pub)
	mockStore.On("Set", mock.Anything, mock.Anything).Return(nil)
	mockKeyRing.On("Set", mock.Anything).Return(nil)

	kc.ImportExistingKeyPair(libp2pPriv, "")

	mockStore.AssertCalled(t, "Set", []byte(keychain.PublicKeyStoreKey), pub)
	mockKeyRing.AssertCalled(t, "Set", privKeyItem)
}

func TestKeychain_GenerateMnemonicKey(t *testing.T) {
	kc := initTestKeychain(t)

	mockStore.On("Set", mock.Anything, mock.Anything).Return(nil)
	mockStore.On("Get", []byte(keychain.PublicKeyStoreKey)).Return(nil, nil)
	mockKeyRing.On("Set", mock.Anything).Return(nil)
	mockKeyRing.On("GetMetadata", mock.Anything).Return(keyring.Metadata{}, nil)

	val, err := kc.GenerateKeyFromMnemonic()
	words := strings.Split(val, " ")

	assert.Nil(t, err)
	assert.NotNil(t, val)
	assert.Equal(t, 12, len(words))
	mockStore.AssertCalled(t, "Set", []byte(keychain.PublicKeyStoreKey), mock.Anything)
	mockKeyRing.AssertCalled(t, "Set", mock.Anything)
}

func TestKeychain_RestoreMnemonicKey(t *testing.T) {
	kc := initTestKeychain(t)

	mockStore.On("Set", mock.Anything, mock.Anything).Return(nil)
	mockStore.On("Get", []byte(keychain.PublicKeyStoreKey)).Return(nil, nil)
	mockKeyRing.On("Set", mock.Anything).Return(nil)
	mockKeyRing.On("GetMetadata", mock.Anything).Return(keyring.Metadata{}, nil)

	mnemonic := "clog chalk blame black uncover frame before decide tuition maple crowd uncle"
	pubFromMnemonic, _ := hex.DecodeString("bbfa792cbf0453dde84947e5733c734b1bc11592190517d579ab589ae8107907")
	privAsHex := "6f0938b7f2beb6f1715aaad71f578a94c51cc8ebd2cb221063e28c8a2efcabb6bbfa792cbf0453dde84947e5733c734b1bc11592190517d579ab589ae8107907"

	val, err := kc.GenerateKeyFromMnemonic(keychain.WithMnemonic(mnemonic))
	assert.Nil(t, err)
	assert.NotNil(t, val)
	assert.Equal(t, mnemonic, val)
	mockStore.AssertCalled(t, "Set", []byte(keychain.PublicKeyStoreKey), pubFromMnemonic)
	mockKeyRing.AssertCalled(t, "Set", keyring.Item{
		Key:   keychain.PrivateKeyStoreKey,
		Data:  []byte(privAsHex + "___" + mnemonic),
		Label: "Space App",
	})
}

func TestKeychain_RestoreMnemonicKeyOnOverrideErr(t *testing.T) {
	kc := initTestKeychain(t)

	mockStore.On("Set", mock.Anything, mock.Anything).Return(nil)
	mockKeyRing.On("Set", mock.Anything).Return(nil)
	mockKeyRing.On("GetMetadata", mock.Anything).Return(keyring.Metadata{}, nil)

	mnemonic := "clog chalk blame black uncover frame before decide tuition maple crowd uncle"
	pubFromMnemonic, _ := hex.DecodeString("a29d5030556f55f32d82b71618e97bfe976ebebc713592122b124881b4da6191")

	mockStore.On("Get", []byte(keychain.PublicKeyStoreKey)).Return(pubFromMnemonic, nil)

	_, err := kc.GenerateKeyFromMnemonic(keychain.WithMnemonic(mnemonic))
	assert.NotNil(t, err)
	assert.Equal(t, errors.New("Error while executing GenerateKeyFromMnemonic. Key pair already exists."), err)
}

func TestKeychain_RestoreMnemonicKeyExistsButNotInKeyring(t *testing.T) {
	kc := initTestKeychain(t)

	mockStore.On("Set", mock.Anything, mock.Anything).Return(nil)
	mockKeyRing.On("Set", mock.Anything).Return(nil)
	mockKeyRing.On("GetMetadata", mock.Anything).Return(keyring.Metadata{}, keyring.ErrKeyNotFound)
	mockStore.On("Get", []byte(keychain.PublicKeyStoreKey)).Return(nil, nil)

	mnemonic := "clog chalk blame black uncover frame before decide tuition maple crowd uncle"
	pubFromMnemonic, _ := hex.DecodeString("bbfa792cbf0453dde84947e5733c734b1bc11592190517d579ab589ae8107907")
	privAsHex := "6f0938b7f2beb6f1715aaad71f578a94c51cc8ebd2cb221063e28c8a2efcabb6bbfa792cbf0453dde84947e5733c734b1bc11592190517d579ab589ae8107907"

	val, err := kc.GenerateKeyFromMnemonic(keychain.WithMnemonic(mnemonic))
	assert.Nil(t, err)
	assert.NotNil(t, val)
	assert.Equal(t, mnemonic, val)
	mockStore.AssertCalled(t, "Set", []byte(keychain.PublicKeyStoreKey), pubFromMnemonic)
	mockKeyRing.AssertCalled(t, "Set", keyring.Item{
		Key:   keychain.PrivateKeyStoreKey,
		Data:  []byte(privAsHex + "___" + mnemonic),
		Label: "Space App",
	})
}

func TestKeychain_RestoreMnemonicKeyMnemonicErr(t *testing.T) {
	kc := initTestKeychain(t)

	mockStore.On("Set", mock.Anything, mock.Anything).Return(nil)
	mockStore.On("Get", []byte(keychain.PublicKeyStoreKey)).Return(nil, nil)
	mockKeyRing.On("Set", mock.Anything).Return(nil)
	mockKeyRing.On("GetMetadata", mock.Anything).Return(keyring.Metadata{}, nil)

	mnemonic := "clog chalk blame black uncover frame before decide tuition maple crowd"

	_, err := kc.GenerateKeyFromMnemonic(keychain.WithMnemonic(mnemonic))
	assert.NotNil(t, err)
	assert.Equal(t, bip39.ErrInvalidMnemonic, err)
}

func TestKeychain_RestoreMnemonicKeyOnOverrideSuccess(t *testing.T) {
	kc := initTestKeychain(t)

	mockStore.On("Set", mock.Anything, mock.Anything).Return(nil)
	mockKeyRing.On("Set", mock.Anything).Return(nil)
	mockKeyRing.On("GetMetadata", mock.Anything).Return(keyring.Metadata{}, nil)

	mnemonic := "clog chalk blame black uncover frame before decide tuition maple crowd uncle"
	pubFromMnemonic, _ := hex.DecodeString("bbfa792cbf0453dde84947e5733c734b1bc11592190517d579ab589ae8107907")
	privAsHex := "6f0938b7f2beb6f1715aaad71f578a94c51cc8ebd2cb221063e28c8a2efcabb6bbfa792cbf0453dde84947e5733c734b1bc11592190517d579ab589ae8107907"

	mockStore.On("Get", []byte(keychain.PublicKeyStoreKey)).Return(pubFromMnemonic, nil)

	val, err := kc.GenerateKeyFromMnemonic(keychain.WithMnemonic(mnemonic), keychain.WithOverride())
	assert.Nil(t, err)
	assert.NotNil(t, val)
	assert.Equal(t, mnemonic, val)
	mockStore.AssertCalled(t, "Set", []byte(keychain.PublicKeyStoreKey), pubFromMnemonic)
	mockKeyRing.AssertCalled(t, "Set", keyring.Item{
		Key:   keychain.PrivateKeyStoreKey,
		Data:  []byte(privAsHex + "___" + mnemonic),
		Label: "Space App",
	})
}

func TestKeychain_GetStoredMnemonic(t *testing.T) {
	kc := initTestKeychain(t)

	mnemonic := "clog chalk blame black uncover frame before decide tuition maple crowd uncle"
	privAsHex := "6f0938b7f2beb6f1715aaad71f578a94c51cc8ebd2cb221063e28c8a2efcabb6bbfa792cbf0453dde84947e5733c734b1bc11592190517d579ab589ae8107907"

	mockKeyRing.On("Get", keychain.PrivateKeyStoreKey).Return(keyring.Item{
		Key:   keychain.PrivateKeyStoreKey,
		Data:  []byte(privAsHex + "___" + mnemonic),
		Label: "Space App",
	}, nil)

	mnemonic2, err := kc.GetStoredMnemonic()

	assert.Nil(t, err)
	assert.Equal(t, mnemonic, mnemonic2)
}
