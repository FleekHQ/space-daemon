package fuse

import (
	"context"
	"runtime"

	"github.com/FleekHQ/space-daemon/log"
)

type State string

const (
	UNSUPPORTED   State = "UNSUPPORTED"
	NOT_INSTALLED State = "NOT_INSTALLED"
	UNMOUNTED     State = "UNMOUNTED"
	MOUNTED       State = "MOUNTED"
	ERROR         State = "ERROR"
)

var supportedOs = map[string]bool{
	"linux":  true,
	"darwin": true,
}

func (s *Controller) GetFuseState(ctx context.Context) (State, error) {
	if !supportedOs[runtime.GOOS] {
		return UNSUPPORTED, nil
	}

	if s.IsMounted() {
		return MOUNTED, nil
	}

	// try and get if it is installed
	installed, err := s.install.IsInstalled(ctx)
	if err != nil {
		log.Error("unable to determine state of extension", err)
		return ERROR, err
	}

	if !installed {
		return NOT_INSTALLED, err
	}

	return UNMOUNTED, nil
}
