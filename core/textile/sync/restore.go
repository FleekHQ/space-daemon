package sync

import (
	"context"

	"github.com/FleekHQ/space-daemon/log"
)

// replicate a local thread on the hub
func (s *synchronizer) restoreBucket(ctx context.Context, bucket string) error {

	b, err := s.getBucket(ctx, bucket)
	if err != nil {
		log.Error("Error in getBucket", err)
		return err
	}

	dir, err := b.ListDirectory(ctx, "")
	if err != nil {
		log.Error("Error in ListDirectory", err)
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
		s.NotifyFileRestore(bucket, m.Path)
	}

	return nil
}
