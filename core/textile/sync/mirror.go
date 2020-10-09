package sync

import (
	"context"
	"fmt"

	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/core/textile/model"
	"github.com/FleekHQ/space-daemon/core/textile/utils"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/db"
	bucketsClient "github.com/textileio/textile/api/buckets/client"
	"github.com/textileio/textile/buckets"
)

const mirrorThreadKeyName = "mirrorV1"

func (s *synchronizer) setMirrorFileBackup(ctx context.Context, path, bucketSlug string, isInProgress bool) error {
	mf, err := s.model.FindMirrorFileByPathAndBucketSlug(ctx, path, bucketSlug)
	if err != nil {
		return err
	}
	if mf != nil {
		// update
		mf.Backup = true
		mf.BackupInProgress = isInProgress

		_, err = s.model.UpdateMirrorFile(ctx, mf)
		if err != nil {
			return err
		}
	} else {
		// create
		mf := &domain.MirrorFile{
			Path:             path,
			BucketSlug:       bucketSlug,
			Backup:           !isInProgress,
			BackupInProgress: isInProgress,
			Shared:           false,
		}

		_, err := s.model.CreateMirrorFile(ctx, mf)
		if err != nil {
			return err
		}
	}

	return nil
}

// unset mirror file as backup
func (s *synchronizer) unsetMirrorFileBackup(ctx context.Context, path, bucketSlug string) error {
	mf, err := s.model.FindMirrorFileByPathAndBucketSlug(ctx, path, bucketSlug)
	if err != nil {
		return err
	}
	if mf == nil {
		log.Warn(fmt.Sprintf("mirror file (path=%+v bucketSlug=%+v) does not exist", path, bucketSlug))
		return nil
	}

	// do not delete the instance because it might be shared
	mf.Backup = false
	mf.BackupInProgress = false

	if _, err = s.model.UpdateMirrorFile(ctx, mf); err != nil {
		return err
	}

	return nil
}

func (s *synchronizer) addCurrentUserAsFileOwner(ctx context.Context, bucket, path string) error {
	bucketModel, err := s.model.FindBucket(ctx, bucket)
	if err != nil {
		return err
	}

	roles := make(map[string]buckets.Role)
	pk, err := s.kc.GetStoredPublicKey()
	if err != nil {
		return err
	}
	tpk := thread.NewLibp2pPubKey(pk)
	roles[tpk.String()] = buckets.Admin

	mirror, err := s.getMirrorBucket(ctx, bucket)
	if err != nil {
		return err
	}

	bucketsClient := mirror.GetClient()
	bucketCtx, _, err := s.getBucketCtx(ctx, bucketModel.RemoteDbID, bucketModel.RemoteBucketSlug, true, bucketModel.EncryptionKey)
	if err != nil {
		return err
	}

	return bucketsClient.PushPathAccessRoles(bucketCtx, mirror.GetData().Key, path, roles)
}

// Creates a mirror bucket.
func (s *synchronizer) createMirrorBucket(ctx context.Context, slug string, enckey []byte) (*model.MirrorBucketSchema, error) {
	newSlug := slug + "_mirror"
	log.Debug("Creating a new mirror bucket with slug " + newSlug)
	dbID, err := s.createMirrorThread(ctx, newSlug)
	if err != nil {
		return nil, err
	}

	hubCtx, _, err := s.getBucketCtx(ctx, utils.CastDbIDToString(*dbID), newSlug, true, enckey)
	if err != nil {
		return nil, err
	}

	b, err := s.hubBuckets.Create(hubCtx, bucketsClient.WithName(newSlug), bucketsClient.WithPrivate(true))
	if err != nil {
		return nil, err
	}

	return &model.MirrorBucketSchema{
		RemoteDbID:       utils.CastDbIDToString(*dbID),
		RemoteBucketKey:  b.Root.Key,
		RemoteBucketSlug: newSlug,
		HubAddr:          s.cfg.GetString(config.TextileHubTarget, ""),
	}, nil
}

// Creates a remote hub thread for the mirror bucket
func (s *synchronizer) createMirrorThread(ctx context.Context, slug string) (*thread.ID, error) {
	log.Debug("createMirrorThread: Generating a new threadID ...")
	var err error
	ctx, err = s.hubAuth.GetHubContext(ctx)
	if err != nil {
		return nil, err
	}

	dbID := thread.NewIDV1(thread.Raw, 32)

	managedKey, err := s.kc.GetManagedThreadKey(mirrorThreadKeyName + "_" + slug)
	if err != nil {
		log.Error("error getting managed thread key", err)
		return nil, err
	}

	log.Debug("createMirrorThread: Creating Thread DB for bucket at db " + dbID.String())
	if err := s.hubThreads.NewDB(ctx, dbID, db.WithNewManagedThreadKey(managedKey)); err != nil {
		return nil, err
	}
	log.Debug("createMirrorThread: Thread DB Created")
	return &dbID, nil
}
