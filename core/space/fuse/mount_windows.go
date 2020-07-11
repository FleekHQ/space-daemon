package fuse

import (
	"context"
	"errors"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/spacefs"
)

var errNotImplemented = errors.New("fuse not implemented for windows")

func pathExists(path string) bool {
	return false
}

func getMountPath(cfg config.Config) (string, error) {
	return "", errNotImplemented
}

func initVFS(ctx context.Context, sfs spacefs.FSOps) VFS {
	return &dummyVFS{}
}

// dummyVFS acts a placeholder vfs for windows pending the actual implementation
type dummyVFS struct{}

func (d dummyVFS) Mount(mountPath, fsName string) error {
	return errNotImplemented
}

func (d dummyVFS) IsMounted() bool {
	return false
}

func (d dummyVFS) Serve() error {
	return errNotImplemented
}

func (d dummyVFS) Unmount() error {
	return errNotImplemented
}
