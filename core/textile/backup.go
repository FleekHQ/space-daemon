package textile

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/FleekHQ/space-daemon/log"

	"github.com/textileio/go-threads/core/thread"
)

// backup a single file with an io.Reader provided
func (tc *textileClient) BackupFileWithReader(ctx context.Context, bucket Bucket, path string, reader io.Reader) (err error) {

	if _, _, err = tc.UploadFileToHub(ctx, bucket, path, reader); err != nil {
		log.Error(fmt.Sprintf("error backuping up file with reader (path=%+v b.Slug=%+v)", path, bucket.Slug()), err)
		return err
	}

	if _, err = tc.setMirrorFileBackup(ctx, path, bucket.Slug()); err != nil {
		log.Error(fmt.Sprintf("error setting mirror file as backup (path=%+v b.Slug=%+v)", path, bucket.Slug()), err)
		return err
	}

	return nil
}

// backup a single file with an io.Reader provided
func (tc *textileClient) backupFile(ctx context.Context, bucket Bucket, path string) error {

	errc := make(chan error, 1)
	pipeReader, pipeWriter := io.Pipe()

	// go routine for piping
	go func() {
		defer close(errc)
		defer pipeWriter.Close()

		err := bucket.GetFile(ctx, path, pipeWriter)
		if err != nil {
			errc <- err
			log.Error(fmt.Sprintf("error getting file (path=%+v b.Slug=%+v)", path, bucket.Slug()), err)
			return
		}
	}()
	if err := <-errc; err != nil {
		return err
	}

	if err := tc.BackupFileWithReader(ctx, bucket, path, pipeReader); err != nil {
		log.Error(fmt.Sprintf("error backing up file (path=%+v b.Slug=%+v)", path, bucket.Slug()), err)
		return err
	}

	return nil
}

// backup all files in a bucket
func (tc *textileClient) backupBucketFiles(ctx context.Context, bucket Bucket, path string) (int, error) {
	var wg sync.WaitGroup
	var count int

	// XXX: we ignore errc (no atomicity at all) for now but we should return
	// XXX: the errors, perhaps in a separate async call
	errc := make(chan error)

	dir, err := bucket.ListDirectory(ctx, path)
	if err != nil {
		return 0, err
	}

	wg.Add(len(dir.Item.Items))

	for _, item := range dir.Item.Items {
		if item.IsDir {
			p := strings.Join([]string{path, item.Name}, "/")

			n, err := tc.backupBucketFiles(ctx, bucket, p)
			if err != nil {
				return 0, err
			}

			count += n
			continue
		}

		// parallelize the backups
		go func(path string) {
			defer wg.Done()

			if err = tc.backupFile(ctx, bucket, path); err != nil {
				errc <- err
				return
			}

			count += 1
		}(item.Path)
	}

	wg.Wait()

	return count, nil
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
		if err = tc.ReplicateThreadToHub(ctx, &dbId); err != nil {
			return 0, err
		}

		replicatedDbIds = append(replicatedDbIds, dbId)
	}

	return len(replicatedDbIds), nil
}

// backup the bucket
func (tc *textileClient) BackupBucket(ctx context.Context, bucket Bucket) (count int, err error) {

	count, err = tc.backupBucketFiles(ctx, bucket, "")
	if err != nil {
		return 0, nil
	}

	if _, err = tc.replicateThreadsToHub(ctx, bucket); err != nil {
		return 0, nil
	}

	return count, nil
}

// unbackup a single file
func (tc *textileClient) unbackupFile(ctx context.Context, bucket Bucket, path string) (err error) {

	if err = tc.unsetMirrorFileBackup(ctx, path, bucket.Slug()); err != nil {
		log.Error(fmt.Sprintf("error unsetting mirror file as backup (path=%+v b.Slug=%+v)", path, bucket.Slug()), err)
		return err
	}

	if err = tc.deleteFileFromHub(ctx, bucket, path); err != nil {
		log.Error(fmt.Sprintf("error backuping up file with reader (path=%+v b.Slug=%+v)", path, bucket.Slug()), err)
		return err
	}

	return nil
}

// unbackup all files in a bucket
func (tc *textileClient) unbackupBucketFiles(ctx context.Context, bucket Bucket, path string) (int, error) {
	var wg sync.WaitGroup
	var count int

	dir, err := bucket.ListDirectory(ctx, "")
	if err != nil {
		return 0, err
	}

	wg.Add(len(dir.Item.Items))

	for _, item := range dir.Item.Items {
		if item.IsDir {
			p := strings.Join([]string{path, item.Name}, "/")

			n, err := tc.unbackupBucketFiles(ctx, bucket, p)
			if err != nil {
				return 0, err
			}

			count += n
			continue
		}

		go func(path string) {
			defer wg.Done()

			if tc.isSharedFile(ctx, bucket, path) {
				return
			}
			if err = tc.unbackupFile(ctx, bucket, path); err != nil {
				return
			}
		}(item.Path)
	}

	wg.Wait()
	return count, nil
}

// dereplicate meta, key and shared with me threads
func (tc *textileClient) dereplicateThreadsToHub(ctx context.Context, bucket Bucket) (int, error) {
	var count int

	userThreadIds, err := tc.userThreadIds(ctx, bucket)
	if err != nil {
		return 0, nil
	}

	for _, dbId := range userThreadIds {
		if err = tc.DereplicateThreadFromHub(ctx, &dbId); err != nil {
			return 0, err
		}

		count += 1
	}

	return count, nil
}

// unbackup the bucket
func (tc *textileClient) UnbackupBucket(ctx context.Context, bucket Bucket) (count int, err error) {

	count, err = tc.unbackupBucketFiles(ctx, bucket, "")
	if err != nil {
		return 0, nil
	}

	if _, err = tc.dereplicateThreadsToHub(ctx, bucket); err != nil {
		return 0, nil
	}

	return count, nil
}
