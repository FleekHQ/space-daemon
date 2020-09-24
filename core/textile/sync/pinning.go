package sync

import (
	"context"
	"io"
)

func (s *synchronizer) uploadFileToHub(ctx context.Context, bucket, path string) error {
	mirror, err := s.getMirrorBucket(ctx, bucket)
	if err != nil {
		return err
	}

	remoteClient := mirror.GetClient()
	remoteCtx, _, err := mirror.GetContext(ctx)
	if err != nil {
		return err
	}

	localBucket, err := s.getBucket(ctx, bucket)
	if err != nil {
		return err
	}

	pipeReader, pipeWriter := io.Pipe()

	if err := localBucket.GetFile(ctx, path, pipeWriter); err != nil {
		return err
	}

	_, _, err = remoteClient.PushPath(remoteCtx, mirror.GetData().Key, path, pipeReader)
	if err != nil {
		return err
	}

	return nil
}
