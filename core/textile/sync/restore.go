package sync

import (
	"context"

	"github.com/FleekHQ/space-daemon/core/textile/bucket"
	"github.com/FleekHQ/space-daemon/log"
)

// restore bucket by downloading files to the local from the mirror bucket
func (s *synchronizer) restoreBucket(ctx context.Context, bucketSlug string) error {

	localBucket, err := s.getBucket(ctx, bucketSlug)
	if err != nil {
		log.Error("Error in getBucket", err)
		return err
	}

	mirrorBucket, err := s.getMirrorBucket(ctx, bucketSlug)
	if err != nil {
		log.Error("Error in getMirrorBucket", err)
		return err
	}

	iterator := func(c context.Context, b *bucket.Bucket, itemPath string) error {
		exists, err := localBucket.FileExists(c, itemPath)
		if err != nil {
			return err
		}

		if exists {
			localUpdatedAt, err := localBucket.UpdatedAt(c, itemPath)
			if err != nil {
				return err
			}

			mirrorUpdatedAt, err := mirrorBucket.UpdatedAt(c, itemPath)
			if err != nil {
				return err
			}

			if localUpdatedAt >= mirrorUpdatedAt {
				// do not overwrite: mirror is not newer
				return nil
			}
		}

		s.NotifyFileRestore(bucketSlug, itemPath)
		return nil
	}

	if _, err = mirrorBucket.Each(ctx, "", iterator, true); err != nil {
		return err
	}

	return nil
}
