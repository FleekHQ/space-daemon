package installer

import (
	"context"
	"errors"
)

type windowsFuseInstaller struct {
}

func NewFuseInstaller() *windowsFuseInstaller {
	return &windowsFuseInstaller{}
}

func (d *windowsFuseInstaller) IsInstalled(ctx context.Context) (bool, error) {
	return false, nil
}

func (d *windowsFuseInstaller) Install(ctx context.Context, args map[string]interface{}) error {
	return errors.New("not supported")
}
