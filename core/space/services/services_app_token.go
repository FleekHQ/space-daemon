package services

import (
	"context"

	"github.com/FleekHQ/space-daemon/core/permissions"
)

func (s *Space) InitializeMasterAppToken(ctx context.Context) (*permissions.AppToken, error) {
	newAppToken, err := permissions.GenerateRandomToken(true, []string{})
	if err != nil {
		return nil, err
	}

	return newAppToken, s.keychain.StoreAppToken(newAppToken)
}
