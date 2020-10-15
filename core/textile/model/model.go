package model

import (
	"context"

	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/core/store"
	"github.com/FleekHQ/space-daemon/core/textile/hub"
	"github.com/FleekHQ/space-daemon/core/textile/utils"
	threadsClient "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
)

const metaThreadName = "metathreadV1"

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
		key []byte,
	) (*ReceivedFileSchema, error)
	FindReceivedFile(ctx context.Context, remoteDbID, bucket, path string) (*ReceivedFileSchema, error)
	CreateSharedPublicKey(ctx context.Context, pubKey string) (*SharedPublicKeySchema, error)
	ListSharedPublicKeys(ctx context.Context) ([]*SharedPublicKeySchema, error)
	CreateMirrorBucket(ctx context.Context, bucketSlug string, mirrorBucket *MirrorBucketSchema) (*BucketSchema, error)
	FindMirrorFileByPathAndBucketSlug(ctx context.Context, path, bucketSlug string) (*MirrorFileSchema, error)
	CreateMirrorFile(ctx context.Context, mirrorFile *domain.MirrorFile) (*MirrorFileSchema, error)
	UpdateMirrorFile(ctx context.Context, mirrorFile *MirrorFileSchema) (*MirrorFileSchema, error)
	ListReceivedFiles(ctx context.Context, accepted bool, seek string, limit int) ([]*ReceivedFileSchema, error)
	FindMirrorFileByPaths(ctx context.Context, paths []string) (map[string]*MirrorFileSchema, error)
	FindReceivedFilesByIds(ctx context.Context, ids []string) ([]*ReceivedFileSchema, error)
}

func New(st store.Store, kc keychain.Keychain, threads *threadsClient.Client, hubAuth hub.HubAuth) *model {
	return &model{
		st:      st,
		kc:      kc,
		threads: threads,
		hubAuth: hubAuth,
	}
}

func (m *model) findOrCreateMetaThreadID(ctx context.Context) (*thread.ID, error) {
	return utils.FindOrCreateDeterministicThreadID(ctx, utils.MetathreadThreadVariant, metaThreadName, m.kc, m.st, m.threads)
}

func (m *model) getMetaThreadContext(ctx context.Context) (context.Context, *thread.ID, error) {
	var err error

	var dbID *thread.ID
	if dbID, err = m.findOrCreateMetaThreadID(ctx); err != nil {
		return nil, nil, err
	}

	metathreadCtx, err := utils.GetThreadContext(ctx, metaThreadName, *dbID, false, m.kc, m.hubAuth, m.threads)
	if err != nil {
		return nil, nil, err
	}

	return metathreadCtx, dbID, nil
}
