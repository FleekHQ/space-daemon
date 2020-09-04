package services

import (
	"context"

	"github.com/FleekHQ/space-daemon/core/space/domain"
)

func (s *Space) GetNotifications(ctx context.Context, seek string, limit int) ([]*domain.Notification, error) {
	r, err := s.tc.GetMailAsNotifications(ctx, seek, limit)
	if err != nil {
		return nil, err
	}
	return r, nil
}
