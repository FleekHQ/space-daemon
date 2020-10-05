package sync

import (
	"context"
	"encoding/hex"
	"errors"

	"github.com/FleekHQ/space-daemon/core/events"
	"github.com/FleekHQ/space-daemon/core/textile/utils"
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
		if err := s.setMirrorFileBackup(ctx, path, bucket, true); err != nil {
			return err
		}
	}

	if s.eventNotifier != nil {
		s.eventNotifier.SendFileEvent(events.NewFileEvent(path, bucket, events.FileBackupInProgress, nil))
	}

	pft := newTask(pinFileTask, []string{bucket, path})
	pft.Parallelizable = true
	s.enqueueTask(pft, s.filePinningQueue)

	s.notifySyncNeeded()

	return nil
}

func (s *synchronizer) processRemoveItem(ctx context.Context, task *Task) error {
	if err := checkTaskType(task, removeItemTask); err != nil {
		return err
	}

	bucket := task.Args[0]
	path := task.Args[1]

	uft := newTask(unpinFileTask, []string{bucket, path})
	uft.Parallelizable = true
	s.enqueueTask(uft, s.filePinningQueue)

	s.notifySyncNeeded()

	if err := s.unsetMirrorFileBackup(ctx, path, bucket); err != nil {
		return err
	}

	err := s.deleteFileFromRemote(ctx, bucket, path)

	return err
}

func (s *synchronizer) processPinFile(ctx context.Context, task *Task) error {
	if err := checkTaskType(task, pinFileTask); err != nil {
		return err
	}

	bucket := task.Args[0]
	path := task.Args[1]

	err := s.uploadFileToRemote(ctx, bucket, path)
	s.setMirrorFileBackup(ctx, path, bucket, false)

	if s.eventNotifier != nil {
		s.eventNotifier.SendFileEvent(events.NewFileEvent(path, bucket, events.FileBackupReady, nil))
	}

	return err
}

func (s *synchronizer) processUnpinFile(ctx context.Context, task *Task) error {
	if err := checkTaskType(task, unpinFileTask); err != nil {
		return err
	}

	bucket := task.Args[0]
	path := task.Args[1]

	err := s.deleteFileFromRemote(ctx, bucket, path)

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

func (s *synchronizer) processBucketBackupOn(ctx context.Context, task *Task) error {
	if err := checkTaskType(task, bucketBackupOnTask); err != nil {
		return err
	}

	bucket := task.Args[0]

	bucketModel, err := s.model.FindBucket(ctx, bucket)
	if err != nil {
		return err
	}

	dbID, err := utils.ParseDbIDFromString(bucketModel.DbID)
	if err != nil {
		return err
	}

	if err := s.replicateThreadToHub(ctx, dbID); err != nil {
		return err
	}

	return s.uploadAllFilesInPath(ctx, bucket, "")
}

func (s *synchronizer) processBucketBackupOff(ctx context.Context, task *Task) error {
	if err := checkTaskType(task, bucketBackupOffTask); err != nil {
		return err
	}

	bucket := task.Args[0]

	bucketModel, err := s.model.FindBucket(ctx, bucket)
	if err != nil {
		return err
	}

	dbID, err := utils.ParseDbIDFromString(bucketModel.DbID)
	if err != nil {
		return err
	}

	if err := s.dereplicateThreadFromHub(ctx, dbID); err != nil {
		return err
	}

	return s.deleteAllFilesInPath(ctx, bucket, "")
}
