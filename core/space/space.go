package space

import (
	"context"
	"errors"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/env"
	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/core/space/services"
	"github.com/FleekHQ/space-daemon/core/store"
	"github.com/FleekHQ/space-daemon/core/textile-new"
)

// Service Layer should not depend on gRPC dependencies
type Service interface {
	RegisterSyncer(sync services.Syncer)
	OpenFile(ctx context.Context, path string, bucketName string) (domain.OpenFileInfo, error)
	GetConfig(ctx context.Context) domain.AppConfig
	ListDirs(ctx context.Context, path string, bucketName string) ([]domain.FileInfo, error)
	ListDir(ctx context.Context, path string, bucketName string) ([]domain.FileInfo, error)
	GenerateKeyPair(ctx context.Context, useForce bool) (domain.KeyPair, error)
	CreateFolder(ctx context.Context, path string, bucketName string) error
	CreateBucket(ctx context.Context, slug string) (textile.Bucket, error)
	ListBuckets(ctx context.Context) ([]textile.Bucket, error)
	AddItems(ctx context.Context, sourcePaths []string, targetPath string, bucketName string) (<-chan domain.AddItemResult, domain.AddItemsResponse, error)
	CreateIdentity(ctx context.Context, username string) (*domain.Identity, error)
	GetIdentityByUsername(ctx context.Context, username string) (*domain.Identity, error)
	ShareBucket(ctx context.Context, slug string) (*domain.ThreadInfo, error)
	JoinBucket(ctx context.Context, slug string, threadinfo *domain.ThreadInfo) (bool, error)
}

type serviceOptions struct {
	cfg config.Config
	env env.SpaceEnv
}

var defaultOptions = serviceOptions{}

type ServiceOption func(o *serviceOptions)

func NewService(store store.Store, tc textile.Client, sync services.Syncer, cfg config.Config, kc keychain.Keychain, opts ...ServiceOption) (Service, error) {
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

	sv := services.NewSpace(store, tc, sync, cfg, o.env, kc)

	return sv, nil
}

func WithEnv(env env.SpaceEnv) ServiceOption {
	return func(o *serviceOptions) {
		if env != nil {
			o.env = env
		}
	}
}
