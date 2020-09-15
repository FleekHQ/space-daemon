package services

import (
	"context"
	"strconv"

	"github.com/FleekHQ/space-daemon/core/space/domain"
)

const notificationsLastSeenAtStoreKey = "notificationsLastSeenAt"

func (s *Space) GetNotifications(ctx context.Context, seek string, limit int) ([]*domain.Notification, error) {
	err := s.waitForTextileHub(ctx)
	if err != nil {
		return nil, err
	}

	r, err := s.tc.GetMailAsNotifications(ctx, seek, limit)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (s *Space) SetNotificationsLastSeenAt(timestamp int64) error {
	t := strconv.FormatInt(timestamp, 10)
	err := s.store.Set([]byte(notificationsLastSeenAtStoreKey), []byte(t))
	if err != nil {
		return err
	}
	return nil
}

func (s *Space) GetNotificationsLastSeenAt() (int64, error) {
	ts, err := s.store.Get([]byte(notificationsLastSeenAtStoreKey))
	if err != nil {
		return 0, err
	}

	i, err := strconv.ParseInt(string(ts), 10, 64)
	if err != nil {
		return 0, err
	}

	return i, nil
}
