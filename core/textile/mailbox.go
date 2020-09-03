package textile

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/FleekHQ/space-daemon/core/events"
	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/log"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/textile/api/users/client"
	"github.com/textileio/textile/cmd"
	mail "github.com/textileio/textile/mail/local"
)

type GrpcMailboxNotifier interface {
	SendNotificationEvent(event events.NotificationEvent)
}

const mailboxSetupFlagStoreKey = "mailboxSetupFlag"

type UsersClient interface {
	ListInboxMessages(ctx context.Context, opts ...client.ListOption) ([]client.Message, error)
	SendMessage(ctx context.Context, from thread.Identity, to thread.PubKey, body []byte) (msg client.Message, err error)
	SetupMailbox(ctx context.Context) (mailbox thread.ID, err error)
}

type Mailbox interface {
	ListInboxMessages(ctx context.Context, opts ...client.ListOption) ([]client.Message, error)
	SendMessage(ctx context.Context, to thread.PubKey, body []byte) (msg client.Message, err error)
	WatchInbox(ctx context.Context, mevents chan<- mail.MailboxEvent, offline bool) (<-chan cmd.WatchState, error)
}

func parseMessage(msg client.Message) (*domain.Notification, error) {
	var b *domain.MessageBody
	err := json.Unmarshal([]byte(msg.Body), b)

	if err != nil {
		return nil, err
	}

	n := &domain.Notification{
		ID:               msg.ID,
		Body:             string(msg.Body),
		NotificationType: (*b).Type,
		CreatedAt:        msg.CreatedAt.Unix(),
		ReadAt:           msg.ReadAt.Unix(),
	}

	switch (*b).Type {
	case domain.INVITATION:
		var i *domain.Invitation
		err := json.Unmarshal((*b).Body, i)
		if err != nil {
			return nil, err
		}

		n.InvitationValue = *i
		n.RelatedObject = i
	case domain.USAGEALERT:
		var u *domain.UsageAlert
		err := json.Unmarshal((*b).Body, u)

		if err != nil {
			return nil, err
		}
		n.UsageAlertValue = *u
		n.RelatedObject = u
	default:
		return nil, errors.New("Unsupported message type")
	}

	return n, nil
}

func (tc *textileClient) SendMessage(ctx context.Context, recipient crypto.PubKey, body []byte) (*client.Message, error) {
	var err error
	ctx, err = tc.getHubCtx(ctx)
	if err != nil {
		return nil, err
	}
	msg, err := tc.mb.SendMessage(ctx, thread.NewLibp2pPubKey(recipient), body)
	if err != nil {
		return nil, err
	}

	return &msg, nil
}

func (tc *textileClient) GetMailAsNotifications(ctx context.Context, seek string, limit int) ([]*domain.Notification, error) {
	var err error
	ns := []*domain.Notification{}

	ctx, err = tc.getHubCtx(ctx)
	if err != nil {
		return nil, err
	}

	notifs, err := tc.mb.ListInboxMessages(ctx, client.WithSeek(seek), client.WithLimit(limit))
	if err != nil {
		return nil, err
	}

	for _, n := range notifs {
		notif, err := parseMessage(n)
		if err != nil {
			return nil, err
		}

		ns = append(ns, notif)
	}

	return ns, nil
}

type handleMessage func(context.Context, interface{}) error

func (tc *textileClient) ListenForMessages(ctx context.Context, srv GrpcMailboxNotifier) error {
	log.Info("Starting to listen for mailbox messages")

	var err error
	ctx, err = tc.getHubCtx(ctx)
	if err != nil {
		return err
	}

	// Handle mailbox events as they arrive
	go func() {
		for e := range tc.mailEvents {
			switch e.Type {
			case mail.NewMessage:
				// handle new message
				log.Info("Received mail: " + e.MessageID.String())

				p, err := parseMessage(e.Message)
				if err != nil {
					log.Error("Unable to parse incoming message: ", err)
				}

				i := events.NotificationEvent{
					Body:          p.Body,
					RelatedObject: p.RelatedObject,
					Type:          events.NotificationType(p.NotificationType),
					CreatedAt:     e.Message.CreatedAt.Unix(),
					ReadAt:        e.Message.ReadAt.Unix(),
				}

				srv.SendNotificationEvent(i)
			case mail.MessageRead:
				// handle message read (inbox only)
			case mail.MessageDeleted:
				// handle message deleted
			}
		}
	}()

	// Start watching (the third param indicates we want to keep watching when offline)
	_, err = tc.mb.WatchInbox(ctx, tc.mailEvents, true)
	if err != nil {
		return err
	}
	// TODO: handle connectivity state if needed
	// for s := range state {
	// 	// handle connectivity state
	// }
	return nil
}
