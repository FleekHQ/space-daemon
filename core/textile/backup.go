package textile

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/FleekHQ/space-daemon/log"

	"github.com/textileio/go-threads/core/thread"
)

// backup a single file with an io.Reader provided
func (tc *textileClient) BackupFileWithReader(ctx context.Context, bucket Bucket, path string, reader io.Reader) (err error) {

	_, _, err = tc.UploadFileToHub(ctx, bucket, path, reader)
	if err != nil {
		log.Error(fmt.Sprintf("error backuping up file with reader (path=%+v b.Slug=%+v)", path, bucket.Slug()), err)
		return err
	}

	_, err = tc.SetMirrorFileBackup(ctx, path, bucket.Slug())
	if err != nil {
		log.Error(fmt.Sprintf("error setting mirror file as backup (path=%+v b.Slug=%+v)", path, bucket.Slug()), err)
		return err
	}

	return nil
}

// backup a single file with an io.Reader provided
func (tc *textileClient) backupFile(ctx context.Context, bucket Bucket, path string) (err error) {

	pipeReader, pipeWriter := io.Pipe()

	go func() {
		err = bucket.GetFile(ctx, path, pipeWriter)
		if err != nil {
			log.Error(fmt.Sprintf("error getting file (path=%+v b.Slug=%+v)", path, bucket.Slug()), err)
			return
		}

		pipeWriter.Close()
	}()

	err = tc.BackupFileWithReader(ctx, bucket, path, pipeReader)
	if err != nil {
		log.Error(fmt.Sprintf("error backing up file (path=%+v b.Slug=%+v)", path, bucket.Slug()), err)
		return err
	}

	return nil
}

// backup all files in a bucket
func (tc *textileClient) backupBucketFiles(ctx context.Context, bucket Bucket) (count int, err error) {

	var wg sync.WaitGroup

	dir, err := bucket.ListDirectory(ctx, "")
	if err != nil {
		return 0, err
	}

	wg.Add(len(dir.Item.Items))

	for _, item := range dir.Item.Items {
		go func() {
			defer wg.Done()

			err = tc.backupFile(ctx, bucket, item.Path)
			if err != nil {
				return
			}
		}()
	}

	wg.Wait()
	return 0, nil
}

// meta, key and share with me thread ids
func (tc *textileClient) userThreadIds(ctx context.Context, bucket Bucket) ([]thread.ID, error) {
	dbIds := make([]thread.ID, 0)

	// key
	bucketThreadID, err := bucket.GetThreadID(ctx)
	if err != nil {
		return nil, err
	}
	dbIds = append(dbIds, *bucketThreadID)

	// shared with me
	publicShareId, err := tc.getPublicShareThread(ctx)
	if err != nil {
		return nil, err
	}
	dbIds = append(dbIds, publicShareId)

	return dbIds, nil
}

// replicate meta, key and shared with me threads
func (tc *textileClient) replicateThreadsToHub(ctx context.Context, bucket Bucket) (count int, err error) {
	replicatedDbIds := make([]thread.ID, 0)

	// poor man's transactionality
	defer func() {
		if err != nil {
			for _, dbId := range replicatedDbIds {
				err = tc.DereplicateThreadFromHub(ctx, &dbId)
				if err != nil {
					log.Error(fmt.Sprintf("failed to dereplicate thread (dbId=%+v b.Slug=%+v)", dbId, bucket.Slug()), err)
				}
			}
		}
	}()

	userThreadIds, err := tc.userThreadIds(ctx, bucket)
	if err != nil {
		return 0, err
	}

	for _, dbId := range userThreadIds {
		err = tc.ReplicateThreadToHub(ctx, &dbId)
		if err != nil {
			return 0, err
		}

		replicatedDbIds = append(replicatedDbIds, dbId)
	}

	return len(replicatedDbIds), nil
}

// backup the bucket
func (tc *textileClient) BackupBucket(ctx context.Context, bucket Bucket) (count int, err error) {

	count, err = tc.backupBucketFiles(ctx, bucket)
	if err != nil {
		return 0, nil
	}

	_, err = tc.replicateThreadsToHub(ctx, bucket)
	if err != nil {
		return 0, nil
	}

	return 0, nil
}

// unbackup a single file
func (tc *textileClient) unbackupFile(ctx context.Context, bucket Bucket, path string) (err error) {

	err = tc.UnsetMirrorFileBackup(ctx, path, bucket.Slug())
	if err != nil {
		log.Error(fmt.Sprintf("error unsetting mirror file as backup (path=%+v b.Slug=%+v)", path, bucket.Slug()), err)
		return err
	}

	err = tc.deleteFileFromHub(ctx, bucket, path)
	if err != nil {
		log.Error(fmt.Sprintf("error backuping up file with reader (path=%+v b.Slug=%+v)", path, bucket.Slug()), err)
		return err
	}

	return nil
}

// return false if file was shared
func (tc *textileClient) isSharedFile(ctx context.Context, bucket Bucket, path string) bool {
	sbc := NewSecureBucketsClient(tc.hb, bucket.Slug())

	roles, err := sbc.PullPathAccessRoles(ctx, bucket.Key(), path)
	if err != nil {
		// TEMP: returning empty members list until we
		// fix it on textile side
		return false
	}

	pk, err := tc.kc.GetStoredPublicKey()
	if err != nil {
		return false
	}

	tpk := thread.NewLibp2pPubKey(pk)

	delete(roles, tpk.String())

	return len(roles) > 0
}

// unbackup all files in a bucket
func (tc *textileClient) unbackupBucketFiles(ctx context.Context, bucket Bucket) (count int, err error) {

	var wg sync.WaitGroup

	dir, err := bucket.ListDirectory(ctx, "")
	if err != nil {
		return 0, err
	}

	wg.Add(len(dir.Item.Items))

	for _, item := range dir.Item.Items {
		go func() {
			defer wg.Done()

			if tc.isSharedFile(ctx, bucket, item.Path) {
				return
			}

			err = tc.unbackupFile(ctx, bucket, item.Path)
			if err != nil {
				return
			}
		}()
	}

	wg.Wait()
	return 0, nil
}

// dereplicate meta, key and shared with me threads
func (tc *textileClient) dereplicateThreadsToHub(ctx context.Context, bucket Bucket) (count int, err error) {

	userThreadIds, err := tc.userThreadIds(ctx, bucket)
	if err != nil {
		return 0, nil
	}

	for _, dbId := range userThreadIds {
		err = tc.DereplicateThreadFromHub(ctx, &dbId)
		if err != nil {
			return 0, err
		}

		count += 1
	}

	return count, nil
}

// unbackup the bucket
func (tc *textileClient) UnbackupBucket(ctx context.Context, bucket Bucket) (count int, err error) {

	count, err = tc.unbackupBucketFiles(ctx, bucket)
	if err != nil {
		return 0, nil
	}

	_, err = tc.dereplicateThreadsToHub(ctx, bucket)
	if err != nil {
		return 0, nil
	}

	return 0, nil
}
