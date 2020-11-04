package model

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/textileio/go-threads/api/client"
	core "github.com/textileio/go-threads/core/db"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/db"
	"github.com/textileio/go-threads/util"
)

type MirrorFileSchema struct {
	ID                core.InstanceID `json:"_id"`
	Path              string          `json:"path"`
	BucketSlug        string          `json:"bucket_slug"`
	Backup            bool            `json:"backup"`
	Shared            bool            `json:"shared"`
	BackupInProgress  bool            `json:"backupInProgress"`
	RestoreInProgress bool            `json:"restoreInProgress"`
	DbID              string
}

type MirrorBucketSchema struct {
	RemoteDbID       string `json:"remoteDbId"`
	RemoteBucketKey  string `json:"remoteBucketKey"`
	HubAddr          string `json:"HubAddr"`
	RemoteBucketSlug string `json:"remoteBucketSlug"`
}

const mirrorFileModelName = "MirrorFile"

var errMirrorFileNotFound = errors.New("Mirror file not found")
var errMirrorFileAlreadyExists = errors.New("Mirror file already exists")

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
	bucket.RemoteBucketSlug = mirrorBucket.RemoteBucketSlug

	instances := client.Instances{bucket}

	err = m.threads.Save(metaCtx, *metaDbID, bucketModelName, instances)
	if err != nil {
		return nil, err
	}

	return bucket, nil
}

func (m *model) FindMirrorFileByPaths(ctx context.Context, paths []string) (map[string]*MirrorFileSchema, error) {
	metaCtx, dbID, err := m.initMirrorFileModel(ctx)
	if err != nil || dbID == nil {
		return nil, err
	}

	var qry *db.Query
	for i, path := range paths {
		if i == 0 {
			qry = db.Where("path").Eq(filepath.Clean(path))
		} else {
			qry = qry.Or(db.Where("path").Eq(filepath.Clean(path)))
		}
	}

	rawMirrorFiles, err := m.threads.Find(metaCtx, *dbID, mirrorFileModelName, qry, &MirrorFileSchema{})
	if err != nil {
		return nil, err
	}

	if rawMirrorFiles == nil {
		return nil, nil
	}

	mirror_files := rawMirrorFiles.([]*MirrorFileSchema)
	if len(mirror_files) == 0 {
		return nil, nil
	}

	mirror_map := make(map[string]*MirrorFileSchema)
	for _, mirror_file := range mirror_files {
		mirror_map[mirror_file.Path] = mirror_file
	}

	return mirror_map, nil
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
		return nil, nil
	}

	mirror_files := rawMirrorFiles.([]*MirrorFileSchema)
	if len(mirror_files) == 0 {
		return nil, nil
	}

	log.Debug("Model.FindMirrorFileByPathAndBucketSlug: returning mirror file with dbid " + mirror_files[0].DbID)
	return mirror_files[0], nil
}

// create a new mirror file
func (m *model) CreateMirrorFile(ctx context.Context, mirrorFile *domain.MirrorFile) (*MirrorFileSchema, error) {
	metaCtx, metaDbID, err := m.initMirrorFileModel(ctx)
	if err != nil && metaDbID == nil {
		return nil, err
	}

	mf, err := m.FindMirrorFileByPathAndBucketSlug(ctx, mirrorFile.Path, mirrorFile.BucketSlug)
	if err != nil {
		return nil, err
	}
	if mf != nil {
		return nil, errMirrorFileAlreadyExists
	}

	newInstance := &MirrorFileSchema{
		Path:              mirrorFile.Path,
		BucketSlug:        mirrorFile.BucketSlug,
		Backup:            mirrorFile.Backup,
		BackupInProgress:  mirrorFile.BackupInProgress,
		RestoreInProgress: mirrorFile.RestoreInProgress,
		Shared:            mirrorFile.Shared,
	}

	instances := client.Instances{newInstance}

	res, err := m.threads.Create(metaCtx, *metaDbID, mirrorFileModelName, instances)
	if err != nil {
		return nil, err
	}

	id := res[0]
	return &MirrorFileSchema{
		Path:              newInstance.Path,
		BucketSlug:        newInstance.BucketSlug,
		Backup:            newInstance.Backup,
		BackupInProgress:  newInstance.BackupInProgress,
		RestoreInProgress: newInstance.RestoreInProgress,
		Shared:            newInstance.Shared,
		ID:                core.InstanceID(id),
		DbID:              newInstance.DbID,
	}, nil
}

// update existing mirror file
func (m *model) UpdateMirrorFile(ctx context.Context, mirrorFile *MirrorFileSchema) (*MirrorFileSchema, error) {
	metaCtx, metaDbID, err := m.initMirrorFileModel(ctx)
	if err != nil && metaDbID == nil {
		return nil, err
	}

	mf, err := m.FindMirrorFileByPathAndBucketSlug(ctx, mirrorFile.Path, mirrorFile.BucketSlug)
	if err != nil {
		return nil, err
	}
	if mf == nil {
		return nil, errMirrorFileNotFound
	}

	existingInstance := mirrorFile
	instances := client.Instances{existingInstance}

	err = m.threads.Save(metaCtx, *metaDbID, mirrorFileModelName, instances)
	if err != nil {
		return nil, err
	}
	log.Debug(fmt.Sprintf("saved mirror file (%+v)", mirrorFile))

	return mf, nil
}

func (m *model) initMirrorFileModel(ctx context.Context) (context.Context, *thread.ID, error) {
	metaCtx, dbID, err := m.getMetaThreadContext(ctx)
	if err != nil {
		return nil, nil, err
	}

	m.threads.NewCollection(metaCtx, *dbID, GetMirrorFileCollectionConfig())

	// Migrates db by adding new fields between old version of the daemon and a new one
	m.threads.UpdateCollection(metaCtx, *dbID, db.CollectionConfig{
		Name:   mirrorFileModelName,
		Schema: util.SchemaFromInstance(&MirrorFileSchema{}, false),
	})

	return metaCtx, dbID, nil
}

func GetMirrorFileCollectionConfig() db.CollectionConfig {
	return db.CollectionConfig{
		Name:   mirrorFileModelName,
		Schema: util.SchemaFromInstance(&MirrorFileSchema{}, false),
		Indexes: []db.Index{{
			Path:   "path",
			Unique: true, // TODO: multicolumn index
		}},
	}
}
