package sync

import (
	"context"
	"io"
	"strings"

	"github.com/FleekHQ/space-daemon/core/textile/bucket"
	"github.com/FleekHQ/space-daemon/core/textile/utils"
	"github.com/FleekHQ/space-daemon/log"
)

func (s *synchronizer) uploadFileToRemote(ctx context.Context, bucket, path string) error {
	mirrorBucket, err := s.getMirrorBucket(ctx, bucket)
	if err != nil {
		return err
	}

	localBucket, err := s.getBucket(ctx, bucket)
	if err != nil {
		return err
	}

	return s.uploadFileToBucket(ctx, localBucket, mirrorBucket, path)
}

func (s *synchronizer) uploadFileToBucket(ctx context.Context, sourceBucket, targetBucket bucket.BucketInterface, path string) error {

	pipeReader, pipeWriter := io.Pipe()
	defer pipeReader.Close()

	errc := make(chan error, 1)
	// go routine for piping
	go func() {
		defer close(errc)
		defer pipeWriter.Close()

		if err := sourceBucket.GetFile(ctx, path, pipeWriter); err != nil {
			errc <- err
			return
		}

		errc <- nil
	}()

	if _, _, err := targetBucket.UploadFile(ctx, path, pipeReader); err != nil {
		return err
	}

	if err := <-errc; err != nil {
		return err
	}

	if err := s.addCurrentUserAsFileOwner(ctx, targetBucket.Slug(), path); err != nil {
		// not returning since we dont want to halt the whole process
		// also acl will still work since they are the owner
		// of the thread so this is more for showing members view
		log.Error("Unable to push path access roles for owner", err)
	}

	return nil
}

// backup all files in a bucket
func (s *synchronizer) uploadAllFilesInPath(ctx context.Context, bucket, path string) error {
	localBucket, err := s.getBucket(ctx, bucket)
	if err != nil {
		return err
	}

	dir, err := localBucket.ListDirectory(ctx, path)
	if err != nil {
		return err
	}

	for _, item := range dir.Item.Items {
		if utils.IsMetaFileName(item.Name) {
			continue
		}

		if item.IsDir {
			p := strings.Join([]string{path, item.Name}, "/")

			err := s.uploadAllFilesInPath(ctx, bucket, p)
			if err != nil {
				return err
			}

			continue
		}

		// If the current item is a file, we add it to the queue so that it both gets pinned and synced
		s.NotifyItemAdded(bucket, path)
	}

	return nil
}

func (s *synchronizer) deleteFileFromRemote(ctx context.Context, bucket, path string) (err error) {
	mirrorBucket, err := s.getMirrorBucket(ctx, bucket)
	if err != nil {
		return err
	}

	_, err = mirrorBucket.DeleteDirOrFile(ctx, path)
	if err != nil {
		return err
	}

	return nil
}

func (s *synchronizer) deleteAllFilesInPath(ctx context.Context, bucket, path string) error {
	localBucket, err := s.getBucket(ctx, bucket)
	if err != nil {
		return err
	}

	dir, err := localBucket.ListDirectory(ctx, path)
	if err != nil {
		return err
	}

	for _, item := range dir.Item.Items {
		if utils.IsMetaFileName(item.Name) {
			continue
		}

		if item.IsDir {
			p := strings.Join([]string{path, item.Name}, "/")

			err := s.deleteAllFilesInPath(ctx, bucket, p)
			if err != nil {
				return err
			}

			continue
		}

		// If the current item is a file, we add it to the queue so that it both gets pinned and synced
		s.NotifyItemRemoved(bucket, path)
	}

	return nil
}
