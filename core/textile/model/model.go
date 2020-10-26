package model

import (
	"context"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/core/store"
	"github.com/FleekHQ/space-daemon/core/textile/hub"
	"github.com/FleekHQ/space-daemon/core/textile/utils"
	"github.com/FleekHQ/space-daemon/log"
	threadsClient "github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
	nc "github.com/textileio/go-threads/net/api/client"
	"github.com/textileio/textile/v2/cmd"
)

const metaThreadName = "metathreadV1"

type model struct {
	st      store.Store
	kc      keychain.Keychain
	threads *threadsClient.Client
	hubAuth hub.HubAuth
	cfg     config.Config
	netc    *nc.Client
	ht      *threadsClient.Client
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
	FindReceivedFile(ctx context.Context, remoteDbID, bucket, path string) (*ReceivedFileSchema, error)
	FindPublicLinkReceivedFile(ctx context.Context, ipfsHash string) (*ReceivedFileSchema, error)
	CreateSharedPublicKey(ctx context.Context, pubKey string) (*SharedPublicKeySchema, error)
	ListSharedPublicKeys(ctx context.Context) ([]*SharedPublicKeySchema, error)
	CreateMirrorBucket(ctx context.Context, bucketSlug string, mirrorBucket *MirrorBucketSchema) (*BucketSchema, error)
	FindMirrorFileByPathAndBucketSlug(ctx context.Context, path, bucketSlug string) (*MirrorFileSchema, error)
	CreateMirrorFile(ctx context.Context, mirrorFile *domain.MirrorFile) (*MirrorFileSchema, error)
	UpdateMirrorFile(ctx context.Context, mirrorFile *MirrorFileSchema) (*MirrorFileSchema, error)
	ListReceivedFiles(ctx context.Context, accepted bool, seek string, limit int) ([]*ReceivedFileSchema, error)
	ListReceivedPublicFiles(ctx context.Context, cidHash string, accepted bool) ([]*ReceivedFileSchema, error)
	FindMirrorFileByPaths(ctx context.Context, paths []string) (map[string]*MirrorFileSchema, error)
	FindReceivedFilesByIds(ctx context.Context, ids []string) ([]*ReceivedFileSchema, error)
}

func New(st store.Store, kc keychain.Keychain, threads *threadsClient.Client, ht *threadsClient.Client, hubAuth hub.HubAuth, cfg config.Config, netc *nc.Client) *model {
	return &model{
		st:      st,
		kc:      kc,
		threads: threads,
		hubAuth: hubAuth,
		cfg:     cfg,
		netc:    netc,
		ht:      ht,
	}
}

func (m *model) findOrCreateMetaThreadID(ctx context.Context) (*thread.ID, error) {
	hubmaStr := m.cfg.GetString(config.TextileHubMa, "")
	hubma := cmd.AddrFromStr(hubmaStr)
	key, err := m.kc.GetManagedThreadKey(metaThreadName)
	if err != nil {
		return nil, err
	}

	threadID, err := utils.NewDeterministicThreadID(m.kc, utils.MetathreadThreadVariant)
	if err != nil {
		return nil, err
	}
	hubmaWithThreadID := hubmaStr + "/thread/" + threadID.String()

	// If we are here, then there's no replicated metathread yet
	if _, err := utils.FindOrCreateDeterministicThreadID(ctx, utils.MetathreadThreadVariant, metaThreadName, m.kc, m.st, m.threads); err != nil {
		return nil, err
	}

	// Try to join remote db if it was already replicated
	err = m.threads.NewDBFromAddr(ctx, cmd.AddrFromStr(hubmaWithThreadID), key)
	if err == nil || err.Error() == "rpc error: code = Unknown desc = db already exists" {
		return &threadID, nil
	}

	if _, err := m.netc.AddReplicator(ctx, threadID, hubma); err != nil {
		log.Error("error while replicating metathread", err)
		// Not returning error in case the user is offline (it should still work using local threads)
	}

	return &threadID, nil
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
