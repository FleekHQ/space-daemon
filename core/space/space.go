package space

import (
	"context"
	"errors"
	"io"

	"github.com/FleekHQ/space-daemon/core/permissions"
	"github.com/FleekHQ/space-daemon/core/textile/hub"
	"github.com/FleekHQ/space-daemon/core/vault"
	crypto "github.com/libp2p/go-libp2p-crypto"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/env"
	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/core/space/services"
	"github.com/FleekHQ/space-daemon/core/store"
	"github.com/FleekHQ/space-daemon/core/textile"
)

// Service Layer should not depend on gRPC dependencies
type Service interface {
	RegisterSyncer(sync services.Syncer)
	OpenFile(ctx context.Context, path, bucketName, dbID string) (domain.OpenFileInfo, error)
	GetConfig(ctx context.Context) domain.AppConfig
	ListDirs(ctx context.Context, path string, bucketName string, listMembers bool) ([]domain.FileInfo, error)
	ListDir(ctx context.Context, path string, bucketName string, listMembers bool) ([]domain.FileInfo, error)
	GenerateKeyPair(ctx context.Context, useForce bool) (mnemonic string, err error)
	DeleteKeypair(ctx context.Context) error
	GetMnemonic(ctx context.Context) (mnemonic string, err error)
	RestoreKeyPairFromMnemonic(ctx context.Context, mnemonic string) error
	RecoverKeysByPassphrase(ctx context.Context, uuid string, pass string, backupType domain.KeyBackupType) error
	BackupKeysByPassphrase(ctx context.Context, uuid string, pass string, backupType domain.KeyBackupType) error
	TestPassphrase(ctx context.Context, uuid string, pass string) error
	GetPublicKey(ctx context.Context) (string, error)
	GetHubAuthToken(ctx context.Context) (string, error)
	CreateFolder(ctx context.Context, path string, bucketName string) error
	CreateBucket(ctx context.Context, slug string) (textile.Bucket, error)
	ListBuckets(ctx context.Context) ([]textile.Bucket, error)
	AddItems(ctx context.Context, sourcePaths []string, targetPath string, bucketName string) (<-chan domain.AddItemResult, domain.AddItemsResponse, error)
	AddItemWithReader(ctx context.Context, reader io.Reader, targetPath, bucketName string) (domain.AddItemResult, error)
	CreateIdentity(ctx context.Context, username string) (*domain.Identity, error)
	GetIdentityByUsername(ctx context.Context, username string) (*domain.Identity, error)
	GenerateFileSharingLink(ctx context.Context, encryptionPassword, path, bucketName, dbID string) (domain.FileSharingInfo, error)
	GenerateFilesSharingLink(ctx context.Context, encryptionPassword string, paths []string, bucketName, dbID string) (domain.FileSharingInfo, error)
	OpenSharedFile(ctx context.Context, cid, password, filename string) (domain.OpenFileInfo, error)
	ShareBucket(ctx context.Context, slug string) (*domain.ThreadInfo, error)
	JoinBucket(ctx context.Context, slug string, threadinfo *domain.ThreadInfo) (bool, error)
	CreateLocalKeysBackup(ctx context.Context, pathToKeyBackup string) error
	RecoverKeysByLocalBackup(ctx context.Context, pathToKeyBackup string) error
	GetNotifications(ctx context.Context, seek string, limit int) ([]*domain.Notification, error)
	ToggleBucketBackup(ctx context.Context, bucketSlug string, bucketBackup bool) error
	BucketBackupRestore(ctx context.Context, bucketSlug string) error
	ShareFilesViaPublicKey(ctx context.Context, paths []domain.FullPath, pubkeys []crypto.PubKey) error
	HandleSharedFilesInvitation(ctx context.Context, invitationId string, accept bool) error
	GetAPISessionTokens(ctx context.Context) (*domain.APISessionTokens, error)
	AddRecentlySharedPublicKeys(ctx context.Context, pubkeys []crypto.PubKey) error
	RecentlySharedPublicKeys(ctx context.Context) ([]crypto.PubKey, error)
	GetSharedWithMeFiles(ctx context.Context, seek string, limit int) ([]*domain.SharedDirEntry, string, error)
	GetSharedByMeFiles(ctx context.Context, seek string, limit int) ([]*domain.SharedDirEntry, string, error)
	SetNotificationsLastSeenAt(timestamp int64) error
	GetNotificationsLastSeenAt() (int64, error)
	TruncateData(ctx context.Context) error
	SearchFiles(ctx context.Context, query string) ([]domain.SearchFileEntry, error)
	InitializeMasterAppToken(ctx context.Context) (*permissions.AppToken, error)
	RemoveDirOrFile(ctx context.Context, path, bucketName string) error
}

type serviceOptions struct {
	cfg config.Config
	env env.SpaceEnv
}

var defaultOptions = serviceOptions{}

type ServiceOption func(o *serviceOptions)

func NewService(
	store store.Store,
	tc textile.Client,
	sync services.Syncer,
	cfg config.Config,
	kc keychain.Keychain,
	v vault.Vault,
	h hub.HubAuth,
	opts ...ServiceOption,
) (Service, error) {
	if !store.IsOpen() {
		return nil, errors.New("service expects an opened store to work")
	}
	o := defaultOptions
	for _, opt := range opts {
		opt(&o)
	}
	if o.env == nil {
		o.env = env.New()
	}

	sv := services.NewSpace(store, tc, sync, cfg, o.env, kc, v, h)

	return sv, nil
}

func WithEnv(env env.SpaceEnv) ServiceOption {
	return func(o *serviceOptions) {
		if env != nil {
			o.env = env
		}
	}
}
