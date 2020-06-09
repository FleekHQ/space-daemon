package services

import (
	"context"

	"github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/core/env"
	"github.com/FleekHQ/space-poc/core/space/domain"
	"github.com/FleekHQ/space-poc/core/store"
	tc "github.com/FleekHQ/space-poc/core/textile/client"
)

// Implementation for space.Service
type Space struct {
	store     store.Store
	cfg       config.Config
	env       env.SpaceEnv
	tc        tc.Client
	watchFile AddFileWatchFunc
}

type AddFileWatchFunc = func(path string) error

func (s *Space) GetConfig(ctx context.Context) domain.AppConfig {
	return domain.AppConfig{
		Port:                 s.cfg.GetInt(config.SpaceServerPort, "-1"),
		AppPath:              s.env.WorkingFolder(),
		TextileHubTarget:     s.cfg.GetString(config.TextileHubTarget, ""),
		TextileThreadsTarget: s.cfg.GetString(config.TextileThreadsTarget, ""),
	}

}

func NewSpace(st store.Store, tc tc.Client, cfg config.Config, env env.SpaceEnv, watchFile AddFileWatchFunc) *Space {
	return &Space{
		store:     st,
		cfg:       cfg,
		env:       env,
		tc:        tc,
		watchFile: watchFile,
	}
}
