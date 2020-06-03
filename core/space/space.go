package space

import (
	"context"
	"errors"
	"github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/core/env"
	"github.com/FleekHQ/space-poc/core/space/domain"
	"github.com/FleekHQ/space-poc/core/space/services"
	"github.com/FleekHQ/space-poc/core/store"
)

// Service Layer should not depend on gRPC dependencies
type Service interface {
	GetConfig(ctx context.Context) domain.AppConfig
	ListDir(ctx context.Context) ([]domain.DirEntry, error)
	GetPathInfo(ctx context.Context, path string) (domain.PathInfo, error)
	GenerateKeyPair(ctx context.Context, useForce bool) (domain.KeyPair, error)
}

type serviceOptions struct {
	cfg config.Config
	env env.SpaceEnv
}

var defaultOptions = serviceOptions{}

type ServiceOption func(o *serviceOptions)

func NewService(store store.Store, cfg config.Config, opts ...ServiceOption) (Service, error) {
	if !store.IsOpen() {
		return nil, errors.New("service expects an opened store to work")
	}
	o := defaultOptions
	for _, opt := range opts {
		opt(&o)
	}
	if  o.env != nil {
		o.env = env.New()
	}
	sv := services.NewSpace(store, cfg, o.env)

	return sv, nil
}

func WithEnv(env env.SpaceEnv) ServiceOption {
	return func(o *serviceOptions) {
		if env != nil {
			o.env = env
		}
	}
}
