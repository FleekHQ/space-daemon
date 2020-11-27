package fuse

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/FleekHQ/space-daemon/core/space/fuse/installer"

	"github.com/FleekHQ/space-daemon/core/spacefs"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/store"
	"github.com/FleekHQ/space-daemon/log"
)

// Controller is the space domain controller for managing the VFS.
// It is used by the grpc server and app/daemon generally
type Controller struct {
	cfg       config.Config
	vfs       VFS
	store     store.Store
	install   installer.FuseInstaller
	isServed  bool
	mountLock sync.RWMutex
	mountPath string
}

var DefaultFuseDriveName = "Space"

func NewController(
	ctx context.Context,
	cfg config.Config,
	store store.Store,
	sfs *spacefs.SpaceFS,
	install installer.FuseInstaller,
) *Controller {
	vfs := initVFS(ctx, sfs)

	return &Controller{
		cfg:       cfg,
		store:     store,
		vfs:       vfs,
		install:   install,
		isServed:  false,
		mountLock: sync.RWMutex{},
	}
}

// ShouldMount check the store and config to determine if the VFS drive was previously mounted
func (s *Controller) ShouldMount() bool {
	if s.cfg.GetString(config.MountFuseDrive, "false") == "true" {
		return true
	}

	mountFuseDrive, err := s.store.Get([]byte(config.MountFuseDrive))
	if err == nil {
		log.Debug("Persisted mountFuseDrive", fmt.Sprintf("state=%s", string(mountFuseDrive)))
		return string(mountFuseDrive) == "true"
	} else {
		log.Debug("No persisted mountFuseDrive state found")
	}

	return false
}

// Mount mounts the vfs drive and immediately serves the handler.
// It starts the Fuse Server in the background
func (s *Controller) Mount() error {
	s.mountLock.Lock()
	defer s.mountLock.Unlock()

	if s.vfs.IsMounted() {
		return nil
	}

	mountPath, err := getMountPath(s.cfg)
	if err != nil {
		return err
	}

	s.mountPath = mountPath

	err = s.vfs.Mount(
		mountPath,
		s.cfg.GetString(config.FuseDriveName, DefaultFuseDriveName),
	)

	if err != nil {
		if !strings.Contains(err.Error(), "exit status 64") {
			return err
		}

		// a drive mount error, so we try unmounting first and retry mounting
		_ = s.vfs.Unmount()
		s.removeMountedPath()
		err = s.vfs.Mount(
			mountPath,
			s.cfg.GetString(config.FuseDriveName, DefaultFuseDriveName),
		)
		if err != nil {
			return err
		}
	}

	// persist mount state to store to trigger remount on restart
	if err := s.store.Set([]byte(config.MountFuseDrive), []byte("true")); err != nil {
		return err
	}

	s.serve()
	return nil
}

func (s *Controller) GetMountPath() string {
	if !s.IsMounted() {
		return ""
	}

	path, _ := getMountPath(s.cfg)
	return path
}

func (s *Controller) serve() {
	if s.isServed {
		return
	}

	go func() {
		s.isServed = true
		defer func() {
			s.isServed = false
		}()

		// this blocks and unblocks when vfs.Unmount() is called
		// or some external thing happens like user unmounting the drive
		err := s.vfs.Serve()
		if err != nil {
			log.Error("error ending fuse server", err)
		}
		log.Info("FUSE Controller server ended")
	}()
}

func (s *Controller) IsMounted() bool {
	s.mountLock.RLock()
	defer s.mountLock.RUnlock()
	return s.vfs.IsMounted()
}

func (s *Controller) Unmount() error {
	s.mountLock.Lock()
	defer s.mountLock.Unlock()
	if !s.vfs.IsMounted() {
		return nil
	}

	// persist unmount state to store to prevent remount on restart
	if err := s.store.Set([]byte(config.MountFuseDrive), []byte("false")); err != nil {
		return err
	}

	err := s.vfs.Unmount()

	return err
}

func (s *Controller) removeMountedPath() {
	if s.mountPath != "" {
		// try unmounting via os
		err := exec.Command("umount", s.mountPath).Run()
		log.Error("Failed to run unmount command", err)
		err = os.RemoveAll(s.mountPath)
		log.Error("Failed to delete mount directory on unmount", err)
	}
}

func (s *Controller) Shutdown() error {
	return s.Unmount()
}
