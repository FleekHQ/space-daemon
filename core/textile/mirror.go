package textile

import (
	"context"
	"fmt"

	"github.com/FleekHQ/space-daemon/log"
)

const mirrorThreadKeyName = "mirrorV1"

func (tc *textileClient) IsMirrorFile(ctx context.Context, path, bucketSlug string) bool {
	mirrorFile, _ := tc.GetModel().FindMirrorFileByPathAndBucketSlug(ctx, path, bucketSlug)
	if mirrorFile != nil {
		return true
	}

	return false
}

// set mirror file as backup

// return true if mirror file is a backup
func (tc *textileClient) isMirrorBackupFile(ctx context.Context, path, bucketSlug string) bool {
	mf, err := tc.GetModel().FindMirrorFileByPathAndBucketSlug(ctx, path, bucketSlug)
	if err != nil {
		log.Error(fmt.Sprintf("Error checking if path=%+v bucketSlug=%+v is a mirror backup file", path, bucketSlug), err)
		return false
	}
	if mf == nil {
		log.Warn(fmt.Sprintf("mirror file (path=%+v bucketSlug=%+v) does not exist", path, bucketSlug))
		return false
	}

	return mf.Backup == true
}

// XXX: public in the interface as the reverse of UploadFileToHub?
func (tc *textileClient) deleteFileFromHub(ctx context.Context, b Bucket, path string) (err error) {
	// XXX: locking?

	bucket, err := tc.GetModel().FindBucket(ctx, b.Slug())
	if err != nil {
		return err
	}

	hubCtx, _, err := tc.getBucketContext(ctx, bucket.RemoteDbID, b.Slug(), true, bucket.EncryptionKey)
	if err != nil {
		return err
	}

	bucketsClient := NewSecureBucketsClient(
		tc.hb,
		b.Slug(),
	)

	_, err = bucketsClient.RemovePath(hubCtx, bucket.RemoteBucketKey, path)
	if err != nil {
		return err
	}

	return nil
}
