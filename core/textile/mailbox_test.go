package textile_test

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"

	tc "github.com/FleekHQ/space-daemon/core/textile"
	"github.com/FleekHQ/space-daemon/mocks"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/textileio/go-threads/core/thread"
	uc "github.com/textileio/textile/api/users/client"
)

var (
	cfg         *mocks.Config
	st          *mocks.Store
	mockUc      *mocks.UsersClient
	mockKc      *mocks.Keychain
	mockPubKey  crypto.PubKey
	mockPrivKey crypto.PrivKey
	mockMb      *mocks.Mailbox
	mockHubAuth *mocks.HubAuth
)

type TearDown func()

func initTestMailbox(t *testing.T) (tc.Client, TearDown) {
	st = new(mocks.Store)
	mockKc = new(mocks.Keychain)
	mockHubAuth = new(mocks.HubAuth)
	mockUc = new(mocks.UsersClient)
	mockMb = new(mocks.Mailbox)
	client := tc.NewClient(st, mockKc, mockHubAuth, mockUc, mockMb)

	mockPubKeyHex := "67730a6678566ead5911d71304854daddb1fe98a396551a4be01de65da01f3a9"
	mockPrivKeyHex := "dd55f8921f90fdf31c6ef9ad86bd90605602fd7d32dc8ea66ab72deb6a82821c67730a6678566ead5911d71304854daddb1fe98a396551a4be01de65da01f3a9"

	pubKeyBytes, _ := hex.DecodeString(mockPubKeyHex)
	privKeyBytes, _ := hex.DecodeString(mockPrivKeyHex)
	mockPubKey, _ = crypto.UnmarshalEd25519PublicKey(pubKeyBytes)
	mockPrivKey, _ = crypto.UnmarshalEd25519PrivateKey(privKeyBytes)

	tearDown := func() {
		st = nil
		client = nil
		mockUc = nil
		mockKc = nil
	}

	return client, tearDown
}

func TestSendMessage(t *testing.T) {
	tc, tearDown := initTestMailbox(t)
	defer tearDown()

	assert.NotNil(t, tc)

	_, rp, _ := crypto.GenerateEd25519Key(nil)

	mockKc.On(
		"GetStoredKeyPairInLibP2PFormat",
	).Return(mockPrivKey, mockPubKey, nil)

	msg := uc.Message{
		ID: "testid",
	}

	mockMb.On("SendMessage", mock.Anything, mock.Anything, mock.Anything).Return(msg, nil)
	mockHubAuth.On("GetHubContext", mock.Anything).Return(context.Background(), nil)
	body := "mockbody"
	rmsg, err := tc.SendMessage(context.Background(), rp, []byte(body))

	assert.NotNil(t, rmsg)
	assert.Nil(t, err)
	mockMb.AssertCalled(t, "SendMessage", context.Background(), thread.NewLibp2pPubKey(rp), []byte(body))
	assert.Equal(t, msg.ID, rmsg.ID)
}

// func TestSendMessageFailGettingSenderKey(t *testing.T) {
// 	tc, tearDown := initTestMailbox(t)
// 	defer tearDown()

// 	assert.NotNil(t, tc)

// 	_, rp, _ := crypto.GenerateEd25519Key(nil)

// 	mockKc.On(
// 		"GetStoredKeyPairInLibP2PFormat",
// 	).Return(nil, nil, keychain.ErrKeyPairNotFound)

// 	msg := uc.Message{
// 		ID: "testid",
// 	}

// mockHubAuth.On("GetHubContext", mock.Anything).Return(context.Background(), nil)
// mockMb.On("SendMessage", mock.Anything, mock.Anything, mock.Anything).Return(msg, nil)
// body := "mockbody"
// rmsg, err := tc.SendMessage(context.Background(), rp, []byte(body))

// 	assert.Nil(t, rmsg)
// 	assert.NotNil(t, err)
// 	assert.Equal(t, keychain.ErrKeyPairNotFound, err)
// 	mockMb.AssertNotCalled(t, "SendMessage", mock.Anything, mock.Anything, mock.Anything)
// }

func TestSendMessageFailureOnHub(t *testing.T) {
	tc, tearDown := initTestMailbox(t)
	defer tearDown()

	assert.NotNil(t, tc)

	_, rp, _ := crypto.GenerateEd25519Key(nil)

	errToRet := errors.New("failed sending message at the hub")

	mockKc.On(
		"GetStoredKeyPairInLibP2PFormat",
	).Return(mockPrivKey, mockPubKey, nil)

	msg := uc.Message{}

	mockMb.On("SendMessage", mock.Anything, mock.Anything, mock.Anything).Return(msg, errToRet)
	mockHubAuth.On("GetHubContext", mock.Anything).Return(context.Background(), nil)
	body := "mockbody"
	rmsg, err := tc.SendMessage(context.Background(), rp, []byte(body))

	assert.Nil(t, rmsg)
	assert.NotNil(t, err)
	assert.Equal(t, errToRet, err)
}
