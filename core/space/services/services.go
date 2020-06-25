package services

import (
	"context"

	"github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/core/env"
	"github.com/FleekHQ/space-poc/core/keychain"
	"github.com/FleekHQ/space-poc/core/space/domain"
	"github.com/FleekHQ/space-poc/core/store"
	"github.com/FleekHQ/space-poc/core/textile"
)

// Implementation for space.Service
type Space struct {
	store    store.Store
	cfg      config.Config
	env      env.SpaceEnv
	tc       textile.Client
	sync     Syncer
	keychain keychain.Keychain
}

type Syncer interface {
	AddFileWatch(addFileInfo domain.AddWatchFile) error
	GetOpenFilePath(bucketSlug string, bucketPath string) (string, bool)
}

type AddFileWatchFunc = func(addFileInfo domain.AddWatchFile) error

func (s *Space) RegisterSyncer(sync Syncer) {
	s.sync = sync
}

func (s *Space) GetConfig(ctx context.Context) domain.AppConfig {
	return domain.AppConfig{
		Port:                 s.cfg.GetInt(config.SpaceServerPort, "-1"),
		AppPath:              s.env.WorkingFolder(),
		TextileHubTarget:     s.cfg.GetString(config.TextileHubTarget, ""),
		TextileThreadsTarget: s.cfg.GetString(config.TextileThreadsTarget, ""),
	}

}

func NewSpace(st store.Store, tc textile.Client, syncer Syncer, cfg config.Config, env env.SpaceEnv, kc keychain.Keychain) *Space {
	return &Space{
		store:    st,
		cfg:      cfg,
		env:      env,
		tc:       tc,
		sync:     syncer,
		keychain: kc,
	}
}
