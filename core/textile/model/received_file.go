package model

import (
	"context"
	"errors"
	"time"

	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/textileio/go-threads/api/client"
	core "github.com/textileio/go-threads/core/db"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/db"
	"github.com/textileio/go-threads/util"
)

type ReceivedFileSchema struct {
	ID            core.InstanceID `json:"_id"`
	DbID          string          `json:"dbId"`
	Bucket        string          `json:"bucket"`
	Path          string          `json:"path"`
	InvitationId  string          `json:"invitationId"`
	Accepted      bool            `json:"accepted"`
	BucketKey     string          `json:"bucketKey`
	EncryptionKey []byte          `json:"encryptionKey`
	CreatedAt     int64           `json:"created_at"`
}

const receivedFileModelName = "ReceivedFile"

var errReceivedFileNotFound = errors.New("Received file not found")

// Creates the metadata for a file that has been shared to the user
func (m *model) CreateReceivedFile(
	ctx context.Context,
	file domain.FullPath,
	invitationID string,
	accepted bool,
	key []byte,
) (*ReceivedFileSchema, error) {
	log.Debug("Model.CreateReceivedFile: Storing received file " + file.Path)
	if existingFile, err := m.FindReceivedFile(ctx, file.DbId, file.Bucket, file.Path); err == nil {
		log.Debug("Model.CreateReceivedFile: Bucket already in collection")
		return existingFile, nil
	}

	log.Debug("Model.CreateReceivedFile: Initializing db")
	metaCtx, metaDbID, err := m.initReceivedFileModel(ctx)
	if err != nil && metaDbID == nil {
		return nil, err
	}

	now := time.Now().UnixNano()

	newInstance := &ReceivedFileSchema{
		ID:            "",
		DbID:          file.DbId,
		Bucket:        file.Bucket,
		Path:          file.Path,
		InvitationId:  invitationID,
		Accepted:      accepted,
		BucketKey:     file.BucketKey,
		EncryptionKey: key,
		CreatedAt:     now,
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
		ID:           core.InstanceID(id),
		DbID:         newInstance.DbID,
		Bucket:       newInstance.Bucket,
		Path:         newInstance.Path,
		InvitationId: newInstance.InvitationId,
		Accepted:     newInstance.Accepted,
		CreatedAt:    newInstance.CreatedAt,
	}, nil
}

func (m *model) FindReceivedFilesByIds(ctx context.Context, ids []string) ([]*ReceivedFileSchema, error) {
	metaCtx, dbID, err := m.initReceivedFileModel(ctx)
	if err != nil || dbID == nil {
		return nil, err
	}

	var qry *db.Query
	for i, id := range ids {
		if i == 0 {
			qry = db.Where("invitationId").Eq(id)
		} else {
			qry = qry.Or(db.Where("invitationId").Eq(id))
		}
	}

	fileSchemasRaw, err := m.threads.Find(metaCtx, *dbID, receivedFileModelName, qry, &ReceivedFileSchema{})
	if err != nil {
		return nil, err
	}

	fileSchemas := fileSchemasRaw.([]*ReceivedFileSchema)

	return fileSchemas, nil
}

// Finds the metadata of a file that has been shared to the user
func (m *model) FindReceivedFile(ctx context.Context, remoteDbID, bucket, path string) (*ReceivedFileSchema, error) {
	metaCtx, dbID, err := m.initReceivedFileModel(ctx)
	if err != nil || dbID == nil {
		return nil, err
	}

	rawFiles, err := m.threads.Find(metaCtx, *dbID, receivedFileModelName, db.Where("dbId").Eq(remoteDbID).And("bucket").Eq(bucket).And("path").Eq(path), &ReceivedFileSchema{})
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

// Lists the metadata of files received by the user
// use accepted bool to look up for either accepted or rejected files
// If seek == "", will start looking from the beginning. If it's an existing ID it will start looking from that ID.
func (m *model) ListReceivedFiles(ctx context.Context, accepted bool, seek string, limit int) ([]*ReceivedFileSchema, error) {
	metaCtx, dbID, err := m.initReceivedFileModel(ctx)
	if err != nil || dbID == nil {
		return nil, err
	}

	query := db.Where("accepted").Eq(accepted).LimitTo(limit)

	if seek != "" {
		query = query.SeekID(core.InstanceID(seek))
	}

	rawFiles, err := m.threads.Find(metaCtx, *dbID, receivedFileModelName, query, &ReceivedFileSchema{})
	if err != nil {
		return nil, err
	}

	if rawFiles == nil {
		return []*ReceivedFileSchema{}, nil
	}

	files := rawFiles.([]*ReceivedFileSchema)
	return files, nil
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
