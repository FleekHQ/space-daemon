package services

import (
	"context"

	"github.com/FleekHQ/space-daemon/core/space/domain"
)

func (s *Space) GetNotifications(ctx context.Context, seek string, limit int64) ([]domain.Notification, int64, error) {
	return []domain.Notification{}, 0, nil
}
