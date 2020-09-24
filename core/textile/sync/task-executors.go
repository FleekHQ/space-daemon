package sync

import (
	"context"
	"encoding/hex"
	"errors"
)

func checkTaskType(t *Task, tp taskType) error {
	if tp != t.Type {
		return errors.New("expected different task type at Textile synchronizer")
	}

	return nil
}

func (s *synchronizer) processAddItem(ctx context.Context, task *Task) error {
	if err := checkTaskType(task, addItemTask); err != nil {
		return err
	}

	bucket := task.Args[0]
	path := task.Args[1]

	bucketModel, err := s.model.FindBucket(ctx, bucket)
	if err != nil {
		return err
	}

	mirrorFile, err := s.model.FindMirrorFileByPathAndBucketSlug(ctx, path, bucket)

	if bucketModel.Backup && mirrorFile == nil {
		if err := s.setMirrorFileBackup(ctx, path, bucket); err != nil {
			return err
		}
		if err := s.addCurrentUserAsFileOwner(ctx, bucket, path); err != nil {
			return err
		}
	}

	return nil
}

func (s *synchronizer) processRemoveItem(ctx context.Context, task *Task) error {
	if err := checkTaskType(task, removeItemTask); err != nil {
		return err
	}

	// TODO: Implement this
	return nil
}

func (s *synchronizer) processPinFile(ctx context.Context, task *Task) error {
	if err := checkTaskType(task, pinFileTask); err != nil {
		return err
	}

	bucket := task.Args[0]
	path := task.Args[1]

	err := s.uploadFileToHub(ctx, bucket, path)

	return err
}

func (s *synchronizer) processCreateBucket(ctx context.Context, task *Task) error {
	if err := checkTaskType(task, createBucketTask); err != nil {
		return err
	}

	bucket := task.Args[0]
	enckey, err := hex.DecodeString(task.Args[1])
	if err != nil {
		return err
	}

	mirror, err := s.createMirrorBucket(ctx, bucket, enckey)
	if mirror != nil {
		_, err = s.model.CreateMirrorBucket(ctx, bucket, mirror)
	}

	return err
}
