package model

import (
	"context"
	"errors"

	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/textileio/go-threads/api/client"
	core "github.com/textileio/go-threads/core/db"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/db"
	"github.com/textileio/go-threads/util"
)

type MirrorFileSchema struct {
	ID         core.InstanceID `json:"_id"`
	Path       string          `json:"path"`
	BucketSlug string          `json:"bucket_slug"`
	Backup     bool            `json:"backup"`
	Shared     bool            `json:"shared"`

	DbID string
}

type MirrorBucketSchema struct {
	RemoteDbID      string `json:"remoteDbId"`
	RemoteBucketKey string `json:"remoteBucketKey"`
	HubAddr         string `json:"HubAddr"`
}

const mirrorFileModelName = "MirrorFile"

var errMirrorFileNotFound = errors.New("Mirror file not found")

func (m *model) CreateMirrorBucket(ctx context.Context, bucketSlug string, mirrorBucket *MirrorBucketSchema) (*BucketSchema, error) {
	metaCtx, metaDbID, err := m.initBucketModel(ctx)
	if err != nil && metaDbID == nil {
		return nil, err
	}

	bucket, err := m.FindBucket(ctx, bucketSlug)
	if err != nil {
		return nil, err
	}

	bucket.RemoteDbID = mirrorBucket.RemoteDbID
	bucket.HubAddr = mirrorBucket.HubAddr
	bucket.RemoteBucketKey = mirrorBucket.RemoteBucketKey

	instances := client.Instances{bucket}

	err = m.threads.Save(metaCtx, *metaDbID, bucketModelName, instances)
	if err != nil {
		return nil, err
	}

	return bucket, nil
}

// Finds the metadata of a file that has been shared to the user
func (m *model) FindMirrorFileByPathAndBucketSlug(ctx context.Context, path, bucketSlug string) (*MirrorFileSchema, error) {
	metaCtx, dbID, err := m.initMirrorFileModel(ctx)
	if err != nil || dbID == nil {
		return nil, err
	}

	rawMirrorFiles, err := m.threads.Find(metaCtx, *dbID, mirrorFileModelName, db.Where("path").Eq(path), &MirrorFileSchema{})
	if err != nil {
		return nil, err
	}

	if rawMirrorFiles == nil {
		return &MirrorFileSchema{}, nil
	}

	mirror_files := rawMirrorFiles.([]*MirrorFileSchema)
	if len(mirror_files) == 0 {
		return &MirrorFileSchema{}, nil
	}

	log.Debug("Model.FindMirrorFileByPathAndBucketSlug: returning mirror file with dbid " + mirror_files[0].DbID)
	return mirror_files[0], nil
}

func (m *model) CreateMirrorFile(ctx context.Context, mirrorFile *domain.MirrorFile) (*MirrorFileSchema, error) {
	metaCtx, metaDbID, err := m.initMirrorFileModel(ctx)
	if err != nil && metaDbID == nil {
		return nil, err
	}

	_, err = m.FindMirrorFileByPathAndBucketSlug(ctx, mirrorFile.Path, mirrorFile.BucketSlug)
	if err != nil {
		return nil, err
	}

	newInstance := &MirrorFileSchema{
		Path:       mirrorFile.Path,
		BucketSlug: mirrorFile.BucketSlug,
		Backup:     mirrorFile.Backup,
		Shared:     mirrorFile.Shared,
	}

	instances := client.Instances{newInstance}

	res, err := m.threads.Create(metaCtx, *metaDbID, mirrorFileModelName, instances)
	if err != nil {
		return nil, err
	}
	log.Debug("stored mirror file with dbid " + newInstance.DbID)

	id := res[0]
	return &MirrorFileSchema{
		Path:       newInstance.Path,
		BucketSlug: newInstance.BucketSlug,
		Backup:     newInstance.Backup,
		Shared:     newInstance.Shared,
		ID:         core.InstanceID(id),
		DbID:       newInstance.DbID,
	}, nil
}

func (m *model) initMirrorFileModel(ctx context.Context) (context.Context, *thread.ID, error) {
	metaCtx, dbID, err := m.getMetaThreadContext(ctx)
	if err != nil {
		return nil, nil, err
	}

	if err = m.threads.NewDB(metaCtx, *dbID); err != nil {
		log.Debug("initMirrorFileModel: db already exists")
	}
	if err := m.threads.NewCollection(metaCtx, *dbID, db.CollectionConfig{
		Name:   mirrorFileModelName,
		Schema: util.SchemaFromInstance(&MirrorFileSchema{}, false),
		Indexes: []db.Index{{
			Path:   "path",
			Unique: true, // TODO: multicolumn index
		}},
	}); err != nil {
		log.Debug("initMirrorFileModel: collection already exists")
	}

	return metaCtx, dbID, nil
}
