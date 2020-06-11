package services

import (
	"context"
	"github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/core/env"
	"github.com/FleekHQ/space-poc/core/space/domain"
	"github.com/FleekHQ/space-poc/core/store"
	"github.com/FleekHQ/space-poc/core/textile"
)

// Implementation for space.Service
type Space struct {
	store     store.Store
	cfg       config.Config
	env       env.SpaceEnv
	tc        textile.Client
	watchFile AddFileWatchFunc
}

type AddFileWatchFunc = func(addFileInfo domain.AddWatchFile) error

func (s *Space) RegisterAddFileWatchFunc(watchFileFunc AddFileWatchFunc) {
	s.watchFile = watchFileFunc
}

func (s *Space) GetConfig(ctx context.Context) domain.AppConfig {
	return domain.AppConfig{
		FolderPath: s.cfg.GetString(config.SpaceFolderPath, ""),
		Port:       s.cfg.GetInt(config.SpaceServerPort, "-1"),
		AppPath:    s.env.WorkingFolder(),
	}

}

func NewSpace(st store.Store, tc textile.Client, cfg config.Config, env env.SpaceEnv, watchFile AddFileWatchFunc) *Space {
	return &Space{
		store:     st,
		cfg:       cfg,
		env:       env,
		tc:        tc,
		watchFile: watchFile,
	}
}
