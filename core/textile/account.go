package textile

import (
	"context"

	"github.com/FleekHQ/space-daemon/core/textile/utils"
)

func (tc *textileClient) DeleteAccount(ctx context.Context) error {
	if err := tc.requiresRunning(); err != nil {
		return err
	}

	// delete local buckets
	bucks, err := tc.ListBuckets(ctx)
	if err != nil {
		return err
	}

	for _, b := range bucks {
		bs, err := tc.GetModel().FindBucket(ctx, b.Slug())
		if err != nil {
			return err
		}
		dbid, err := b.GetThreadID(ctx)
		if err != nil {
			return err
		}
		ctx, _, err = tc.getBucketContext(ctx, utils.CastDbIDToString(*dbid), b.Slug(), false, bs.EncryptionKey)
		err = tc.bucketsClient.Remove(ctx, b.Key())
		if err != nil {
			return err
		}

		ctx, _, err = tc.getBucketContext(ctx, bs.RemoteDbID, b.Slug(), true, bs.EncryptionKey)
		err = tc.hb.Remove(ctx, bs.RemoteBucketKey)
		if err != nil {
			return err
		}
	}

	// disable sync
	tc.DisableSync()

	// stop backgroundjobs
	tc.sync.Shutdown()

	// stop listener
	tc.DeleteListeners(ctx)

	return nil
}
