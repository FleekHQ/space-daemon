package textile

import (
	"context"
	"errors"

	"github.com/FleekHQ/space-daemon/log"
	"github.com/textileio/go-threads/core/thread"
)

func (tc *textileClient) syncBucketThread(ctx context.Context, dbID thread.ID) error {
	dbInfo, err := tc.threads.GetDBInfo(ctx, dbID)
	if err != nil {
		return err
	}

	ctx, err = tc.getHubCtx(ctx)
	if err != nil {
		return err
	}

	for _, addr := range dbInfo.Addrs {
		// NOTE: dbInfo.key is dangerous to use here becuase
		// its the users key itself, need to resolve this before
		/// merging
		err = tc.ht.NewDBFromAddr(ctx, addr, dbInfo.Key)
		if err != nil {
			log.Warn("Unable to join thread")
		} else {
			// as long as one of the addresses work it's fine
			// might need to tweak later
			return nil
		}
	}

	return errors.New("enable to join thread on any addresses")
}
