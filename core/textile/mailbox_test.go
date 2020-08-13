package textile_test

import (
	"context"
	"errors"
	"testing"

	"github.com/FleekHQ/space-daemon/core/keychain"
	tc "github.com/FleekHQ/space-daemon/core/textile"
	"github.com/FleekHQ/space-daemon/mocks"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/textileio/go-threads/core/thread"
	uc "github.com/textileio/textile/api/users/client"
)

var (
	cfg    *mocks.Config
	st     *mocks.Store
	mockUc *mocks.UsersClient
)

type TearDown func()

func initTestMailbox(t *testing.T) (tc.Client, keychain.Keychain, TearDown) {
	st = new(mocks.Store)
	client := tc.NewClient(st)
	mockUc = new(mocks.UsersClient)
	client.SetUc(mockUc)
	kc := keychain.New(st)
	tearDown := func() {
		st = nil
		kc = nil
		client = nil
		mockUc = nil
	}

	return client, kc, tearDown
}

func TestSendMessage(t *testing.T) {
	tc, kc, tearDown := initTestMailbox(t)
	defer tearDown()

	assert.NotNil(t, tc)

	_, rp, _ := crypto.GenerateEd25519Key(nil)

	st.On("Set", mock.Anything, mock.Anything).Return(nil)
	pub, priv, _ := kc.GenerateKeyPairWithForce()
	st.On("Get", []byte(keychain.PublicKeyStoreKey)).Return(pub, nil)
	st.On("Get", []byte(keychain.PrivateKeyStoreKey)).Return(priv, nil)

	msg := uc.Message{
		ID: "testid",
	}

	mockUc.On("SendMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(msg, nil)
	body := "mockbody"
	rb, _ := rp.Raw()
	rmsg, err := tc.SendMessage(context.Background(), string(rb), []byte(body))

	privateKey, _, err := kc.GetStoredKeyPairInLibP2PFormat()
	id := thread.NewLibp2pIdentity(privateKey)

	assert.NotNil(t, rmsg)
	assert.Nil(t, err)
	mockUc.AssertCalled(t, "SendMessage", context.Background(), id, thread.NewLibp2pPubKey(rp), []byte(body))
	assert.Equal(t, msg.ID, rmsg.ID)
}

func TestSendMessageInvalidKey(t *testing.T) {
	tc, kc, tearDown := initTestMailbox(t)
	defer tearDown()

	assert.NotNil(t, tc)

	st.On("Set", mock.Anything, mock.Anything).Return(nil)
	pub, priv, _ := kc.GenerateKeyPairWithForce()
	st.On("Get", []byte(keychain.PublicKeyStoreKey)).Return(pub, nil)
	st.On("Get", []byte(keychain.PrivateKeyStoreKey)).Return(priv, nil)

	msg := &uc.Message{
		ID: "testid",
	}

	rec := "invalidrecipientpubkey"
	body := "mockbody"

	mockUc.On("SendMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(msg, nil)

	_, err := tc.SendMessage(context.Background(), rec, []byte(body))

	assert.NotNil(t, err)
	mockUc.AssertNotCalled(t, "SendMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	assert.Equal(t, errors.New("expect ed25519 public key data size to be 32"), err)
}
