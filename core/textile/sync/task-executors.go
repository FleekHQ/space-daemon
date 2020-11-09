package sync

import (
	"context"
	"encoding/hex"
	"errors"

	"path"

	"golang.org/x/sync/errgroup"

	"github.com/FleekHQ/space-daemon/core/textile/model"
	api_buckets_pb "github.com/textileio/textile/v2/api/buckets/pb"

	"github.com/FleekHQ/space-daemon/log"

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

	localBucket, err := s.getBucket(ctx, bucket)
	if err != nil {
		return err
	}

	item, err := localBucket.ListDirectory(ctx, path)
	if s.eventNotifier != nil && err == nil {
		info := utils.MapDirEntryToFileInfo(api_buckets_pb.ListPathResponse(*item), path)
		info.LocallyAvailable = true
		info.BackupInProgress = true
		s.eventNotifier.SendFileEvent(events.NewFileEvent(info, events.FileBackupInProgress, bucket, bucketModel.DbID))
	}

	pft := newTask(pinFileTask, []string{bucket, path})
	s.enqueueTask(pft, s.filePinningQueue)
	s.notifySyncNeeded()

	s.NotifyIndexItemAdded(bucket, path, "")

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

	rIndexTask := newTask(removeIndexItemTask, []string{bucket, path, ""})
	s.enqueueTask(rIndexTask, s.taskQueue)

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

	localBucket, err := s.getBucket(ctx, bucket)
	if err != nil {
		return err
	}

	bucketModel, err := s.model.FindBucket(ctx, bucket)
	if err != nil {
		return err
	}

	item, err := localBucket.ListDirectory(ctx, path)
	if s.eventNotifier != nil && err == nil {
		info := utils.MapDirEntryToFileInfo(api_buckets_pb.ListPathResponse(*item), path)
		info.LocallyAvailable = true
		info.BackedUp = true
		s.eventNotifier.SendFileEvent(events.NewFileEvent(info, events.FileBackupReady, bucket, bucketModel.DbID))
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

	newerBucket, err := s.newerBucketPath(ctx, localBucket, mirrorBucket, path)
	if err != nil {
		return err
	}

	if newerBucket == localBucket {
		// do not overwrite: mirror is not newer
		return nil
	}

	// TODO: use timestamp or CID for check

	if err = s.downloadFile(ctx, mirrorBucket, localBucket, path); err != nil {
		return err
	}

	bucketModel, err := s.model.FindBucket(ctx, bucket)
	if err != nil {
		return err
	}

	item, err := mirrorBucket.ListDirectory(ctx, path)
	if s.eventNotifier != nil && err == nil {
		info := utils.MapDirEntryToFileInfo(api_buckets_pb.ListPathResponse(*item), path)
		info.LocallyAvailable = true
		info.BackedUp = true
		s.eventNotifier.SendFileEvent(events.NewFileEvent(info, events.FileRestored, bucket, bucketModel.DbID))
	}

	return err
}

func (s *synchronizer) processAddIndexItemTask(ctx context.Context, task *Task) error {
	if err := checkTaskType(task, addIndexItemTask); err != nil {
		return err
	}

	bucket := task.Args[0]
	itemPath := task.Args[1]
	dbId := task.Args[2]

	if dbId != "" {
		// handle shared file instances
		file, err := s.model.FindReceivedFile(ctx, dbId, itemPath, bucket)
		if err != nil {
			log.Error(
				"ProcessIndexItemTask: unable to find shared file",
				err,
				"dbId:"+dbId, "itemPath:"+itemPath, "bucket:"+bucket,
			)
			return err
		}

		_, err = s.model.UpdateSearchIndexRecord(ctx, file.FileName, file.Path, model.FileItem, file.Bucket, dbId)
		if err != nil {
			log.Error(
				"ProcessIndexItemTask: failed to index shared file",
				err,
			)
			return err
		}
	} else {
		erg, ctx := errgroup.WithContext(ctx)
		// index file
		erg.Go(func() error {
			fileName := path.Base(itemPath)
			_, err := s.model.UpdateSearchIndexRecord(ctx, fileName, itemPath, model.FileItem, bucket, "")
			if err != nil {
				log.Error(
					"ProcessIndexItemTask: failed to index file",
					err,
				)
				return err
			}

			return nil
		})

		// index parent dir
		erg.Go(func() error {
			parentPath := path.Dir(itemPath)
			if parentPath == "/" {
				return nil
			}

			dirName := path.Base(parentPath)
			_, err := s.model.UpdateSearchIndexRecord(ctx, dirName, parentPath, model.DirectoryItem, bucket, "")
			if err != nil {
				log.Error(
					"ProcessIndexItemTask: failed to index directory",
					err,
				)
				return err
			}

			return nil
		})

		err := erg.Wait()
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *synchronizer) processRemoveIndexItemTask(ctx context.Context, task *Task) error {
	if err := checkTaskType(task, removeIndexItemTask); err != nil {
		return err
	}

	bucket := task.Args[0]
	itemPath := task.Args[1]
	dbId := task.Args[2]
	fileName := path.Base(itemPath)

	return s.model.DeleteSearchIndexRecord(ctx, fileName, itemPath, bucket, dbId)
}
