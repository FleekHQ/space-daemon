package installer

import (
	"context"
	"errors"
)

type linuxFuseInstaller struct {
}

func NewFuseInstaller() *linuxFuseInstaller {
	return &linuxFuseInstaller{}
}

func (d *linuxFuseInstaller) IsInstalled(ctx context.Context) (bool, error) {
	return true, nil // assume fuse is installed on recent linux builds
}

func (d *linuxFuseInstaller) Install(ctx context.Context, args map[string]interface{}) error {
	return errors.New("not supported")
}
