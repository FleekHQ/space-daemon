package services

import (
	"context"
	"errors"
	"time"

	"github.com/FleekHQ/space-daemon/core/textile/hub"
	"github.com/FleekHQ/space-daemon/core/vault"
	"golang.org/x/sync/errgroup"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/env"
	node "github.com/FleekHQ/space-daemon/core/ipfs/node"
	"github.com/FleekHQ/space-daemon/core/keychain"
	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/core/store"
	"github.com/FleekHQ/space-daemon/core/textile"
)

// Implementation for space.Service
type Space struct {
	store    store.Store
	cfg      config.Config
	env      env.SpaceEnv
	tc       textile.Client
	sync     Syncer
	keychain keychain.Keychain
	vault    vault.Vault
	hub      hub.HubAuth
	ipfsNode *node.IpfsNode
	buckd    textile.Buckd
	aeg      *errgroup.Group
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

func NewSpace(
	st store.Store,
	tc textile.Client,
	syncer Syncer,
	cfg config.Config,
	env env.SpaceEnv,
	kc keychain.Keychain,
	v vault.Vault,
	h hub.HubAuth,
	i *node.IpfsNode,
	b textile.Buckd,
	aeg *errgroup.Group,
) *Space {
	return &Space{
		store:    st,
		cfg:      cfg,
		env:      env,
		tc:       tc,
		sync:     syncer,
		keychain: kc,
		vault:    v,
		hub:      h,
		ipfsNode: i,
		buckd:    b,
		aeg:      aeg,
	}
}

var textileClientInitTimeout = time.Second * 60
var textileClientHubTimeout = time.Second * 60 * 3

// Waits for textile client to be initialized before returning.
func (s *Space) waitForTextileInit(ctx context.Context) error {
	if s.tc.IsInitialized() {
		return nil
	}

	select {
	case <-time.After(textileClientInitTimeout):
		return errors.New("textile client not initialized in expected time")
	case <-s.tc.WaitForInitialized():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Waits for textile client to be healthy (initialized and connected to hub) before returning.
// If it exceeds the max amount of retries, it returns an error.
func (s *Space) waitForTextileHub(ctx context.Context) error {
	if s.tc.IsHealthy() {
		return nil
	}

	select {
	case err := <-s.tc.WaitForHealthy():
		// This returns error if there were 3 failed attempts to connect
		if err != nil {
			return err
		}
		return nil
	case <-time.After(textileClientHubTimeout):
		return errors.New("textile client not initialized in expected time")
	case <-ctx.Done():
		return ctx.Err()
	}

}
