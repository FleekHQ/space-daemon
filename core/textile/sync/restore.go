package sync

import (
	"context"

	"github.com/FleekHQ/space-daemon/core/events"
	"github.com/FleekHQ/space-daemon/core/textile/bucket"
	"github.com/FleekHQ/space-daemon/log"
)

// return the targetBucket if path is newer there, srcBucket otherwise
func (s *synchronizer) newerBucketPath(ctx context.Context, srcBucket, targetBucket bucket.BucketInterface, path string) (bucket.BucketInterface, error) {
	targetUpdatedAt, err := targetBucket.UpdatedAt(ctx, path)
	if err != nil {
		return nil, err
	}

	srcUpdatedAt, err := srcBucket.UpdatedAt(ctx, path)
	if err != nil {
		// Path might not exist in src bucket
		return targetBucket, nil
	}

	if srcUpdatedAt >= targetUpdatedAt {
		return srcBucket, nil
	}

	return targetBucket, nil
}

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
		exists, _ := localBucket.FileExists(c, itemPath)
		// if err != nil && err.Error() != "rpc error: code = DeadlineExceeded desc = context deadline exceeded" {
		// 	// deadline exceeded can be interpreted as the file not being present
		// 	return err
		// }

		if exists {
			newerBucket, err := s.newerBucketPath(c, localBucket, mirrorBucket, itemPath)
			if err != nil {
				return err
			}

			if newerBucket == localBucket {
				// do not overwrite: mirror is not newer
				return nil
			}
		}

		if s.eventNotifier != nil {
			s.eventNotifier.SendFileEvent(events.NewFileEvent(itemPath, bucketSlug, events.FileRestoring, nil))
		}

		s.NotifyFileRestore(bucketSlug, itemPath)
		return nil
	}

	if _, err = mirrorBucket.Each(ctx, "", iterator, true); err != nil {
		return err
	}

	return nil
}
