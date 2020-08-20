package textile

import (
	"context"
	"errors"

	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/textile/api/users/client"
)

type UsersClient interface {
	SendMessage(ctx context.Context, from thread.Identity, to thread.PubKey, body []byte) (msg client.Message, err error)
	SetupMailbox(ctx context.Context) (mailbox thread.ID, err error)
}

func (tc *textileClient) SendMessage(ctx context.Context, recipient crypto.PubKey, body []byte) (*client.Message, error) {
	var privateKey crypto.PrivKey
	var err error
	if privateKey, _, err = tc.kc.GetStoredKeyPairInLibP2PFormat(); err != nil {
		return nil, err
	}
	id := thread.NewLibp2pIdentity(privateKey)

	msg, err := tc.uc.SendMessage(ctx, id, thread.NewLibp2pPubKey(recipient), body)
	if err != nil {
		return nil, err
	}

	return &msg, nil
}

type handleMessage func(context.Context, interface{}) error

func (tc *textileClient) ListenForMessages(ctx context.Context, handler handleMessage) error {
	return errors.New("Not Implemented")
}
