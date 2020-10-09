package sync

import (
	"context"

	"github.com/FleekHQ/space-daemon/core/events"
	"github.com/FleekHQ/space-daemon/log"
)

// replicate a local thread on the hub
func (s *synchronizer) restoreBucket(ctx context.Context, bucketSlug string) error {

	bucket, err := s.getBucket(ctx, bucketSlug)
	if err != nil {
		log.Error("Error in ListDir", err)
		return err
	}

	dir, err := bucket.ListDirectory(ctx, "")
	if err != nil {
		log.Error("Error in ListDir", err)
		return err
	}

	dirPaths := make([]string, 0)
	for _, item := range dir.Item.Items {
		dirPaths = append(dirPaths, item.Path)
	}

	mirrorFiles, err := s.model.FindMirrorFileByPaths(ctx, dirPaths)
	if err != nil {
		log.Error("Error fetching mirror files", err)
		return err
	}

	for _, m := range mirrorFiles {
		if err = s.downloadFileFromRemote(ctx, m.BucketSlug, m.Path); err != nil {
			log.Error("Error downloading file", err)
			return err
		}

		if s.eventNotifier != nil {
			s.eventNotifier.SendFileEvent(events.NewFileEvent(m.Path, m.BucketSlug, events.FileRestoring, nil))
		}
	}

	return nil
}
