package sync

import (
	"context"

	"github.com/FleekHQ/space-daemon/core/events"
	"github.com/FleekHQ/space-daemon/core/textile/bucket"
	"github.com/FleekHQ/space-daemon/core/textile/utils"
	"github.com/FleekHQ/space-daemon/log"
	api_buckets_pb "github.com/textileio/textile/v2/api/buckets/pb"
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

		item, err := mirrorBucket.ListDirectory(ctx, itemPath)
		if s.eventNotifier != nil && err == nil {
			info := utils.MapDirEntryToFileInfo(api_buckets_pb.ListPathResponse(*item), itemPath)
			info.BackedUp = true
			info.LocallyAvailable = exists
			info.RestoreInProgress = true
			s.eventNotifier.SendFileEvent(events.NewFileEvent(info, events.FileRestoring))
		}

		s.NotifyFileRestore(bucketSlug, itemPath)
		return nil
	}

	if _, err = mirrorBucket.Each(ctx, "", iterator, true); err != nil {
		return err
	}

	return nil
}
