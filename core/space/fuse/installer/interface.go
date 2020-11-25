package installer

import "context"

type FuseInstaller interface {
	IsInstalled(ctx context.Context) (bool, error)
	Install(ctx context.Context, args map[string]interface{}) error
	// TODO: UnInstall(ctx context.Context)
}
