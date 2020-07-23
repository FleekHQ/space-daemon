package textile

import (
	"context"
	"errors"

	"github.com/FleekHQ/space-daemon/core/space/domain"
)

func (tc *textileClient) SendInviteMessage(ctx context.Context, recipient string, ti *domain.ThreadInfo, msg *string) error {
	return errors.New("Not implemented")
}
