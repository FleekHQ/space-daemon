package space

import (
	"context"
	"errors"
	"fmt"
	"github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/core/env"
	"github.com/FleekHQ/space-poc/core/space/domain"
	"github.com/FleekHQ/space-poc/core/space/services"
	"github.com/FleekHQ/space-poc/core/store"
	tc "github.com/FleekHQ/space-poc/core/textile/client"
	"log"
)

// Service Layer should not depend on gRPC dependencies
type Service interface {
	OpenFile(ctx context.Context, path string, bucketSlug string) (domain.OpenFileInfo, error)
	GetConfig(ctx context.Context) domain.AppConfig
	ListDir(ctx context.Context) ([]domain.FileInfo, error)
	GetPathInfo(ctx context.Context, path string) (domain.FileInfo, error)
	GenerateKeyPair(ctx context.Context, useForce bool) (domain.KeyPair, error)
	CreateFolder(ctx context.Context, path string) error
	AddItems(ctx context.Context, sourcePaths []string, targetPath string) error
}



type serviceOptions struct {
	cfg       config.Config
	env       env.SpaceEnv
	watchFile services.AddFileWatchFunc
}

var defaultOptions = serviceOptions{}

type ServiceOption func(o *serviceOptions)

func NewService(store store.Store, tc tc.Client, cfg config.Config, opts ...ServiceOption) (Service, error) {
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

	if o.watchFile == nil {
		o.watchFile = defaultWatchFile
	}

	sv := services.NewSpace(store, tc, cfg, o.env, o.watchFile)

	return sv, nil
}

func defaultWatchFile(path string) error {
	log.Println(fmt.Sprintf("WARNING: using default watch file func to add path %s. File will not be watched", path))
	return nil
}

func WithEnv(env env.SpaceEnv) ServiceOption {
	return func(o *serviceOptions) {
		if env != nil {
			o.env = env
		}
	}
}

func WithAddWatchFileFunc(fileFunc services.AddFileWatchFunc) ServiceOption {
	return func(o *serviceOptions) {
		if fileFunc != nil {
			o.watchFile = fileFunc
		}
	}
}
