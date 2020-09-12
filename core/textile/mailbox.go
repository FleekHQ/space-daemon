package textile

import (
	"context"
	"encoding/json"
	"errors"
	"os/user"
	"path/filepath"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/log"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/textile/api/users/client"
	"github.com/textileio/textile/cmd"
	mail "github.com/textileio/textile/mail/local"
)

type GrpcMailboxNotifier interface {
	SendNotificationEvent(notif *domain.Notification)
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
	Identity() thread.Identity
}

func (tc *textileClient) parseMessage(ctx context.Context, msg client.Message) (*domain.Notification, error) {
	p, err := msg.Open(ctx, tc.mb.Identity())
	if err != nil {
		return nil, err
	}

	b := &domain.MessageBody{}
	err = json.Unmarshal(p, b)

	if err != nil {
		log.Error("Error parsing message into MessageBody type", err)

		// returning generic notification since body was not able to be parsed
		n := &domain.Notification{
			ID:        msg.ID,
			Body:      string(p),
			CreatedAt: msg.CreatedAt.Unix(),
			ReadAt:    msg.ReadAt.Unix(),
		}

		return n, nil
	}

	n := &domain.Notification{
		ID:               msg.ID,
		Body:             string(p),
		NotificationType: (*b).Type,
		CreatedAt:        msg.CreatedAt.Unix(),
		ReadAt:           msg.ReadAt.Unix(),
	}

	switch (*b).Type {
	case domain.INVITATION:
		i := &domain.Invitation{}
		err := json.Unmarshal((*b).Body, i)
		if err != nil {
			return nil, err
		}

		n.InvitationValue = *i
		n.RelatedObject = *i
	case domain.USAGEALERT:
		u := &domain.UsageAlert{}
		err := json.Unmarshal((*b).Body, u)

		if err != nil {
			return nil, err
		}
		n.UsageAlertValue = *u
		n.RelatedObject = *u
	default:
	}

	return n, nil
}

func (tc *textileClient) SendMessage(ctx context.Context, recipient crypto.PubKey, body []byte) (*client.Message, error) {
	if err := tc.requiresHubConnection(); err != nil {
		return nil, err
	}

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
	if err := tc.requiresHubConnection(); err != nil {
		return nil, err
	}

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
		notif, err := tc.parseMessage(ctx, n)
		if err != nil {
			return nil, err
		}

		ns = append(ns, notif)
	}

	return ns, nil
}

type handleMessage func(context.Context, interface{}) error

func (tc *textileClient) listenForMessages(ctx context.Context) error {
	if tc.mbNotifier == nil {
		return errors.New("no mailbox notifier, run AttachMailboxNotifier first")
	}
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

				// need to fetch the message again because the event
				// payload doesn't have the full deets, will remove
				// once its fixed on txl end
				msg, err := tc.mb.ListInboxMessages(ctx, client.WithSeek(e.MessageID.String()), client.WithLimit(1))
				if err != nil {
					return
				}

				p, err := tc.parseMessage(ctx, msg[0])
				if err != nil {
					log.Error("Unable to parse incoming message: ", err)
				}

				tc.mbNotifier.SendNotificationEvent(p)
			case mail.MessageRead:
				// handle message read (inbox only)
			case mail.MessageDeleted:
				// handle message deleted
			}
		}
	}()

	// Start watching (the third param indicates we want to keep watching when offline)
	go func() {
		state, err := tc.mb.WatchInbox(ctx, tc.mailEvents, true)
		if err != nil {
			log.Error("Unable to watch mailbox, ", err)
			return
		}

		// TODO: handle connectivity state if needed
		for s := range state {
			log.Info("received inbox watch state: " + s.State.String())
		}
	}()

	return nil
}

// Attachs a handler for mailbox notification events
func (tc *textileClient) AttachMailboxNotifier(notif GrpcMailboxNotifier) {
	tc.mbNotifier = notif
}

func (tc *textileClient) createMailBox(ctx context.Context, maillib *mail.Mail, mbpath string) (*mail.Mailbox, error) {
	// create
	priv, _, err := tc.kc.GetStoredKeyPairInLibP2PFormat()
	if err != nil {
		return nil, err
	}

	id := thread.NewLibp2pIdentity(priv)

	mailbox, err := maillib.NewMailbox(ctx, mail.Config{
		Path:      mbpath,
		Identity:  id,
		APIKey:    tc.cfg.GetString(config.TextileUserKey, ""),
		APISecret: tc.cfg.GetString(config.TextileUserSecret, ""),
	})
	if err != nil {
		return nil, err
	}
	tc.store.Set([]byte(mailboxSetupFlagStoreKey), []byte("true"))
	return mailbox, nil
}

func (tc *textileClient) setupOrCreateMailBox(ctx context.Context) (*mail.Mailbox, error) {
	maillib := mail.NewMail(cmd.NewClients(tc.cfg.GetString(config.TextileHubTarget, ""), true), mail.DefaultConfConfig())

	usr, _ := user.Current()
	mbpath := filepath.Join(usr.HomeDir, ".fleek-space/textile/mail")

	var mailbox *mail.Mailbox
	dbid, err := tc.store.Get([]byte(mailboxSetupFlagStoreKey))
	if err == nil && len(dbid) > 0 {
		// restore
		mailbox, err = maillib.GetLocalMailbox(ctx, mbpath)
		if err != nil {
			return nil, err
		}
	} else {
		mailbox, err = tc.createMailBox(ctx, maillib, mbpath)
		if err != nil {
			return nil, err
		}
	}

	mid := mailbox.Identity()
	log.Info("Mailbox identity: " + mid.GetPublic().String())
	return mailbox, nil
}
