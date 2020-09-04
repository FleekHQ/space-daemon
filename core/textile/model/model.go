package model

import (
	"context"
	"errors"

	"github.com/FleekHQ/space-daemon/core/space/domain"

	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/store"
	"github.com/FleekHQ/space-daemon/core/textile/hub"
	"github.com/FleekHQ/space-daemon/core/textile/utils"
	"github.com/FleekHQ/space-daemon/log"
	threadsClient "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
)

const metaThreadName = "metathreadV1"
const threadIDStoreKey = "thread_id"

type model struct {
	st      store.Store
	kc      keychain.Keychain
	threads *threadsClient.Client
	hubAuth hub.HubAuth
}

type Model interface {
	CreateBucket(ctx context.Context, bucketSlug, dbID string) (*BucketSchema, error)
	UpsertBucket(ctx context.Context, bucketSlug, dbID string) (*BucketSchema, error)
	BucketBackupToggle(ctx context.Context, bucketSlug string, backup bool) (*BucketSchema, error)
	FindBucket(ctx context.Context, bucketSlug string) (*BucketSchema, error)
	ListBuckets(ctx context.Context) ([]*BucketSchema, error)
	CreateReceivedFile(
		ctx context.Context,
		file domain.FullPath,
		invitationId string,
		accepted bool,
	) (*ReceivedFileSchema, error)
	FindReceivedFile(ctx context.Context, file domain.FullPath) (*ReceivedFileSchema, error)
}

func New(st store.Store, kc keychain.Keychain, threads *threadsClient.Client, hubAuth hub.HubAuth) *model {
	return &model{
		st:      st,
		kc:      kc,
		threads: threads,
		hubAuth: hubAuth,
	}
}

// Returns the store key for a thread ID. It uses the keychain to obtain the public key, since the store key depends on it.
func getMetathreadStoreKey(kc keychain.Keychain) ([]byte, error) {
	pub, err := kc.GetStoredPublicKey()
	if err != nil {
		return nil, err
	}

	pubInBytes, err := pub.Raw()
	if err != nil {
		return nil, err
	}

	result := []byte(threadIDStoreKey + "_" + metaThreadName)
	result = append(result, pubInBytes...)

	return result, nil
}

func (m *model) findOrCreateMetaThreadID(ctx context.Context) (*thread.ID, error) {
	storeKey, err := getMetathreadStoreKey(m.kc)
	if err != nil {
		return nil, err
	}

	if val, _ := m.st.Get(storeKey); val != nil {
		// Cast the stored dbID from bytes to thread.ID
		if dbID, err := thread.Cast(val); err != nil {
			return nil, err
		} else {
			return &dbID, nil
		}
	}

	// thread id does not exist yet

	// We need to create an ID that's derived deterministically from the user private key
	// The reason for this is that the user needs to be able to restore the exact ID when moving across devices.
	// The only consideration is that we must try to avoid dbID collisions with other users.
	dbID, err := utils.NewDeterministicThreadID(m.kc, utils.MetathreadThreadVariant)
	if err != nil {
		return nil, err
	}

	dbIDInBytes := dbID.Bytes()

	log.Debug("Model.findOrCreateMetaThreadID: Created meta thread in db " + dbID.String())

	if err := m.threads.NewDB(ctx, dbID); err != nil {
		return nil, err
	}

	if err := m.st.Set(storeKey, dbIDInBytes); err != nil {
		newErr := errors.New("error while storing thread id: check your local space db accessibility")
		return nil, newErr
	}

	return &dbID, nil
}

func (m *model) getMetaThreadContext(ctx context.Context) (context.Context, *thread.ID, error) {
	var err error

	var dbID *thread.ID
	if dbID, err = m.findOrCreateMetaThreadID(ctx); err != nil {
		return nil, nil, err
	}

	metathreadCtx, err := utils.GetThreadContext(ctx, metaThreadName, *dbID, false, m.kc, m.hubAuth)
	if err != nil {
		return nil, nil, err
	}

	return metathreadCtx, dbID, nil
}
