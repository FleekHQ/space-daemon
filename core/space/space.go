package space

import (
	"context"
	"errors"

	"github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/core/env"
	"github.com/FleekHQ/space-poc/core/space/domain"
	"github.com/FleekHQ/space-poc/core/space/services"
	"github.com/FleekHQ/space-poc/core/store"
	"github.com/FleekHQ/space-poc/core/textile"
)

// Service Layer should not depend on gRPC dependencies
type Service interface {
	RegisterSyncer(sync services.Syncer)
	OpenFile(ctx context.Context, path string) (domain.OpenFileInfo, error)
	GetConfig(ctx context.Context) domain.AppConfig
	ListDirs(ctx context.Context, path string) ([]domain.FileInfo, error)
	ListDir(ctx context.Context, path string) ([]domain.FileInfo, error)
	GenerateKeyPair(ctx context.Context, useForce bool) (domain.KeyPair, error)
	CreateFolder(ctx context.Context, path string) error
	AddItems(ctx context.Context, sourcePaths []string, targetPath string) (<-chan domain.AddItemResult, domain.AddItemsResponse, error)
}

type serviceOptions struct {
	cfg config.Config
	env env.SpaceEnv
}

var defaultOptions = serviceOptions{}

type ServiceOption func(o *serviceOptions)

func NewService(store store.Store, tc textile.Client, sync services.Syncer, cfg config.Config, opts ...ServiceOption) (Service, error) {
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

	sv := services.NewSpace(store, tc, sync, cfg, o.env)

	return sv, nil
}

func WithEnv(env env.SpaceEnv) ServiceOption {
	return func(o *serviceOptions) {
		if env != nil {
			o.env = env
		}
	}
}
