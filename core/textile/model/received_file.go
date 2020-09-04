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

type ReceivedFileSchema struct {
	ID     core.InstanceID `json:"_id"`
	DbID   string          `json:"dbId"`
	Bucket string          `json:"bucket"`
	Path   string          `json:"path"`
}

const receivedFileModelName = "ReceivedFile"

var errReceivedFileNotFound = errors.New("Received file not found")

// Creates the metadata for a file that has been shared to the user
func (m *model) CreateReceivedFile(ctx context.Context, file domain.FullPath) (*ReceivedFileSchema, error) {
	log.Debug("Model.CreateReceivedFile: Storing received file " + file.Path)
	if existingFile, err := m.FindReceivedFile(ctx, file); err == nil {
		log.Debug("Model.CreateReceivedFile: Bucket already in collection")
		return existingFile, nil
	}

	log.Debug("Model.CreateReceivedFile: Initializing db")
	metaCtx, metaDbID, err := m.initReceivedFileModel(ctx)
	if err != nil && metaDbID == nil {
		return nil, err
	}

	newInstance := &ReceivedFileSchema{
		ID:     "",
		DbID:   file.DbId,
		Bucket: file.Bucket,
		Path:   file.Path,
	}

	instances := client.Instances{newInstance}
	log.Debug("Model.CreateReceivedFile: Creating instance")

	res, err := m.threads.Create(metaCtx, *metaDbID, receivedFileModelName, instances)
	if err != nil {
		return nil, err
	}
	log.Debug("Model.CreateReceivedFile: stored received file with path " + newInstance.Path)

	id := res[0]
	return &ReceivedFileSchema{
		ID:     core.InstanceID(id),
		DbID:   newInstance.DbID,
		Bucket: newInstance.Bucket,
		Path:   newInstance.Path,
	}, nil
}

// Finds the metadata of a file that has been shared to the user
func (m *model) FindReceivedFile(ctx context.Context, file domain.FullPath) (*ReceivedFileSchema, error) {
	metaCtx, dbID, err := m.initReceivedFileModel(ctx)
	if err != nil || dbID == nil {
		return nil, err
	}

	rawFiles, err := m.threads.Find(metaCtx, *dbID, receivedFileModelName, db.Where("dbId").Eq(file.DbId).And("bucket").Eq(file.Bucket).And("path").Eq(file.Path), &ReceivedFileSchema{})
	if err != nil {
		return nil, err
	}

	if rawFiles == nil {
		return nil, errReceivedFileNotFound
	}

	files := rawFiles.([]*ReceivedFileSchema)
	if len(files) == 0 {
		return nil, errReceivedFileNotFound
	}
	log.Debug("Model.FindReceivedFile: returning file with path " + files[0].Path)
	return files[0], nil
}

func (m *model) initReceivedFileModel(ctx context.Context) (context.Context, *thread.ID, error) {
	metaCtx, dbID, err := m.getMetaThreadContext(ctx)
	if err != nil {
		return nil, nil, err
	}

	if err = m.threads.NewDB(metaCtx, *dbID); err != nil {
		log.Debug("initReceivedFileModel: db already exists")
	}
	if err := m.threads.NewCollection(metaCtx, *dbID, db.CollectionConfig{
		Name:   receivedFileModelName,
		Schema: util.SchemaFromInstance(&ReceivedFileSchema{}, false),
	}); err != nil {
		log.Debug("initReceivedFileModel: collection already exists")
	}

	return metaCtx, dbID, nil
}
