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

	if err := s.uploadFileToRemote(ctx, bucket, path); err != nil {
		return err
	}

	s.setMirrorFileBackup(ctx, path, bucket, false)

	if s.eventNotifier != nil {
		s.eventNotifier.SendFileEvent(events.NewFileEvent(path, bucket, events.FileBackupReady, nil))
	}

	return nil
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
		if err != nil {
			return err
		}
	}

	if err := s.addBucketListener(ctx, bucket); err != nil {
		return err
	}

	return nil
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

	// race
	if bucketModel.Backup == false {
		return nil
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

	// race
	if bucketModel.Backup == true {
		return nil
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

func (s *synchronizer) processBucketRestoreTask(ctx context.Context, task *Task) error {
	if err := checkTaskType(task, bucketRestoreTask); err != nil {
		return err
	}

	bucket := task.Args[0]

	if err := s.restoreBucket(ctx, bucket); err != nil {
		return err
	}

	return nil
}

func (s *synchronizer) processRestoreFile(ctx context.Context, task *Task) error {
	if err := checkTaskType(task, restoreFileTask); err != nil {
		return err
	}

	bucket := task.Args[0]
	path := task.Args[1]

	localBucket, err := s.getBucket(ctx, bucket)
	if err != nil {
		return err
	}

	mirrorBucket, err := s.getMirrorBucket(ctx, bucket)
	if err != nil {
		return err
	}

	// TODO: use timestamp or CID for check

	if err = s.uploadFileToBucket(ctx, mirrorBucket, localBucket, path); err != nil {
		return err
	}

	if s.eventNotifier != nil {
		s.eventNotifier.SendFileEvent(events.NewFileEvent(path, bucket, events.FileRestored, nil))
	}

	return err
}
