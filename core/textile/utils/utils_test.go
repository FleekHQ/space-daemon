package utils_test

import (
	"encoding/hex"
	"testing"

	"github.com/FleekHQ/space-daemon/core/textile/utils"
	"github.com/FleekHQ/space-daemon/mocks"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/stretchr/testify/assert"
)

var (
	mockStore    *mocks.Store
	mockKeychain *mocks.Keychain
	mockPubKey   crypto.PubKey
	mockPrivKey  crypto.PrivKey
)

func initMocks(t *testing.T) {
	mockStore = new(mocks.Store)
	mockStore.On("IsOpen").Return(true)

	mockKeychain = new(mocks.Keychain)

	mockPubKeyHex := "67730a6678566ead5911d71304854daddb1fe98a396551a4be01de65da01f3a9"
	mockPrivKeyHex := "dd55f8921f90fdf31c6ef9ad86bd90605602fd7d32dc8ea66ab72deb6a82821c67730a6678566ead5911d71304854daddb1fe98a396551a4be01de65da01f3a9"

	pubKeyBytes, _ := hex.DecodeString(mockPubKeyHex)
	privKeyBytes, _ := hex.DecodeString(mockPrivKeyHex)
	mockPubKey, _ = crypto.UnmarshalEd25519PublicKey(pubKeyBytes)
	mockPrivKey, _ = crypto.UnmarshalEd25519PrivateKey(privKeyBytes)
}

func TestUtils_NewDeterministicThreadID(t *testing.T) {
	initMocks(t)

	mockKeychain.On(
		"GetStoredPublicKey",
	).Return(mockPubKey, nil)

	threadID, err := utils.NewDeterministicThreadID(mockKeychain, utils.MetathreadThreadVariant)
	assert.Nil(t, err)
	threadIDCopy, err := utils.NewDeterministicThreadID(mockKeychain, utils.MetathreadThreadVariant)
	assert.Nil(t, err)

	// Generate a thread ID from a different private key (changed the last char)
	mockPubKeyHex := "67730a6678566ead5911d71304854daddb1fe98a396551a4be01de65da01f3a8"
	pubKeyBytes, _ := hex.DecodeString(mockPubKeyHex)
	diffPubKey, _ := crypto.UnmarshalEd25519PublicKey(pubKeyBytes)
	newMockKeychain := new(mocks.Keychain)
	newMockKeychain.On(
		"GetStoredPublicKey",
	).Return(diffPubKey, nil)

	diffThreadID, err := utils.NewDeterministicThreadID(newMockKeychain, utils.MetathreadThreadVariant)
	assert.Nil(t, err)

	assert.Equal(t, threadID, threadIDCopy)
	assert.NotEqual(t, threadID, diffThreadID)
}
