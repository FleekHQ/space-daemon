package model

import (
	"context"

	"github.com/FleekHQ/space-daemon/core/search"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/core/store"
	"github.com/FleekHQ/space-daemon/core/textile/hub"
	"github.com/FleekHQ/space-daemon/core/textile/utils"
	threadsClient "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/db"
	nc "github.com/textileio/go-threads/net/api/client"
)

const metaThreadName = "metathreadV1"

type model struct {
	st                 store.Store
	kc                 keychain.Keychain
	threads            *threadsClient.Client
	hubAuth            hub.HubAuth
	cfg                config.Config
	netc               *nc.Client
	hnetc              *nc.Client
	ht                 *threadsClient.Client
	shouldForceRestore bool
	fsearch            search.FilesSearchEngine
}

type Model interface {
	CreateBucket(ctx context.Context, bucketSlug, dbID string) (*BucketSchema, error)
	UpsertBucket(ctx context.Context, bucketSlug, dbID string) (*BucketSchema, error)
	BucketBackupToggle(ctx context.Context, bucketSlug string, backup bool) (*BucketSchema, error)
	FindBucket(ctx context.Context, bucketSlug string) (*BucketSchema, error)
	ListBuckets(ctx context.Context) ([]*BucketSchema, error)
	CreateReceivedFileViaInvitation(
		ctx context.Context,
		file domain.FullPath,
		invitationId string,
		accepted bool,
		key []byte,
	) (*ReceivedFileSchema, error)
	CreateReceivedFileViaPublicLink(
		ctx context.Context,
		ipfsHash string,
		password string,
		filename string,
		filesize string,
		accepted bool,
	) (*ReceivedFileSchema, error)
	CreateSentFileViaInvitation(
		ctx context.Context,
		file domain.FullPath,
		invitationId string,
		key []byte,
	) (*SentFileSchema, error)
	FindReceivedFile(ctx context.Context, remoteDbID, bucket, path string) (*ReceivedFileSchema, error)
	FindPublicLinkReceivedFile(ctx context.Context, ipfsHash string) (*ReceivedFileSchema, error)
	FindSentFile(ctx context.Context, remoteDbID, bucket, path string) (*SentFileSchema, error)
	CreateSharedPublicKey(ctx context.Context, pubKey string) (*SharedPublicKeySchema, error)
	ListSharedPublicKeys(ctx context.Context) ([]*SharedPublicKeySchema, error)
	CreateMirrorBucket(ctx context.Context, bucketSlug string, mirrorBucket *MirrorBucketSchema) (*BucketSchema, error)
	FindMirrorFileByPathAndBucketSlug(ctx context.Context, path, bucketSlug string) (*MirrorFileSchema, error)
	CreateMirrorFile(ctx context.Context, mirrorFile *domain.MirrorFile) (*MirrorFileSchema, error)
	UpdateMirrorFile(ctx context.Context, mirrorFile *MirrorFileSchema) (*MirrorFileSchema, error)
	ListReceivedFiles(ctx context.Context, accepted bool, seek string, limit int) ([]*ReceivedFileSchema, error)
	ListSentFiles(ctx context.Context, seek string, limit int) ([]*SentFileSchema, error)
	ListReceivedPublicFiles(ctx context.Context, cidHash string, accepted bool) ([]*ReceivedFileSchema, error)
	FindMirrorFileByPaths(ctx context.Context, paths []string) (map[string]*MirrorFileSchema, error)
	FindReceivedFilesByIds(ctx context.Context, ids []string) ([]*ReceivedFileSchema, error)
	InitSearchIndexCollection(ctx context.Context) error
	UpdateSearchIndexRecord(
		ctx context.Context,
		name, path string,
		itemType SearchItemType,
		bucketSlug, dbId string,
	) (*SearchIndexRecord, error)
	QuerySearchIndex(ctx context.Context, query string) ([]*SearchIndexRecord, error)
	DeleteSearchIndexRecord(ctx context.Context, name, path, bucketSlug, dbId string) error
}

func New(
	st store.Store,
	kc keychain.Keychain,
	threads *threadsClient.Client,
	ht *threadsClient.Client,
	hubAuth hub.HubAuth,
	cfg config.Config,
	netc *nc.Client,
	hnetc *nc.Client,
	shouldForceRestore bool,
	search search.FilesSearchEngine,
) *model {
	return &model{
		st:                 st,
		kc:                 kc,
		threads:            threads,
		hubAuth:            hubAuth,
		cfg:                cfg,
		netc:               netc,
		hnetc:              hnetc,
		ht:                 ht,
		shouldForceRestore: shouldForceRestore,
		fsearch:            search,
	}
}

func (m *model) findOrCreateMetaThreadID(ctx context.Context) (*thread.ID, error) {
	return utils.FindOrCreateDeterministicThread(
		ctx,
		utils.MetathreadThreadVariant,
		metaThreadName,
		m.kc,
		m.st,
		m.threads,
		m.cfg,
		m.netc,
		m.hnetc,
		m.hubAuth,
		m.shouldForceRestore,
		GetAllCollectionConfigs(),
	)
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

func GetAllCollectionConfigs() []db.CollectionConfig {
	return []db.CollectionConfig{
		GetBucketCollectionConfig(),
		GetMirrorFileCollectionConfig(),
		GetReceivedFileCollectionConfig(),
		GetSentFileCollectionConfig(),
		GetSharedPublicKeyCollectionConfig(),
	}
}
