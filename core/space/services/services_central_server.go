package services

import (
	"context"

	"github.com/FleekHQ/space-daemon/core/space/domain"
)

// Return session token for central services authenticated access
func (s *Space) GetAPISessionTokens(ctx context.Context) (*domain.APISessionTokens, error) {
	tokens, err := s.hub.GetTokensWithCache(ctx)
	if err != nil {
		return nil, err
	}

	return &domain.APISessionTokens{
		HubToken:      tokens.HubToken,
		ServicesToken: tokens.AppToken,
	}, nil
}
