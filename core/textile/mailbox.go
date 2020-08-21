package textile

import (
	"context"
	"errors"

	"github.com/FleekHQ/space-daemon/core/space/domain"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/textile/api/users/client"
)

// Seek:      args.seek, string
// Limit:     int64(args.limit),
// Ascending: args.ascending, bool
// Status:    pb.ListInboxMessagesRequest_Status(args.status),

// type ListInboxMessagesRequest_Status int32

// const (
// 	ListInboxMessagesRequest_ALL    ListInboxMessagesRequest_Status = 0
// 	ListInboxMessagesRequest_READ   ListInboxMessagesRequest_Status = 1
// 	ListInboxMessagesRequest_UNREAD ListInboxMessagesRequest_Status = 2
// )

type UsersClient interface {
	ListInboxMessages(ctx context.Context, opts ...client.ListOption) ([]client.Message, error)
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

func (tc *textileClient) GetMailAsNotifications(ctx context.Context, seek string, limit int64) ([]domain.Notification, error) {
	// call sortmail and format each message
	return []domain.Notification{}, nil
}

func (tc *textileClient) sortMail() {}

type handleMessage func(context.Context, interface{}) error

func (tc *textileClient) ListenForMessages(ctx context.Context, handler handleMessage) error {
	return errors.New("Not Implemented")
}
