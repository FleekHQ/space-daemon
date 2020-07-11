//+build !windows

package fuse

import (
	"context"
	"fmt"
	"os"
	s "strings"

	"github.com/FleekHQ/space-daemon/core/libfuse"

	"github.com/FleekHQ/space-daemon/core/spacefs"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/mitchellh/go-homedir"
)

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return os.IsExist(err)
}

func getMountPath(cfg config.Config) (string, error) {
	mountPath := cfg.GetString(config.FuseMountPath, "~/"+DefaultFuseDriveName)
	if home, err := homedir.Dir(); err == nil {
		// If the mount directory contains ~, we replace it with the actual home directory
		mountPath = s.TrimRight(
			s.Replace(mountPath, "~", home, -1),
			"/",
		)
	}

	// checks to ensure we are not mounting on an already existing path
	if pathExists(mountPath) {
		// loop through 10 suffixes till we find on that exists
		for i := 0; i < 10; i++ {
			newPath := fmt.Sprintf("%s%d", mountPath, i)
			if !pathExists(newPath) {
				mountPath = newPath
				break
			}
		}
	}

	return mountPath, nil
}

func initVFS(ctx context.Context, sfs spacefs.FSOps) VFS {
	return libfuse.NewVFileSystem(ctx, sfs)
}
