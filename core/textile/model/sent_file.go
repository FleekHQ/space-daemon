package model

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/textileio/go-threads/api/client"
	core "github.com/textileio/go-threads/core/db"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/db"
	"github.com/textileio/go-threads/util"
)

type SentFileViaInvitationSchema struct {
	DbID          string `json:"dbId"`
	Bucket        string `json:"bucket"`
	Path          string `json:"path"`
	InvitationId  string `json:"invitationId"`
	BucketKey     string `json:"bucketKey"`
	EncryptionKey []byte `json:"encryptionKey"`
}

// SentFileSchema represents data of files shared by the user
type SentFileSchema struct {
	ID        core.InstanceID `json:"_id"`
	CreatedAt int64           `json:"created_at"`
	SentFileViaInvitationSchema
}

const sentFileModelName = "SentFile"

var errSentFileNotFound = errors.New("Sent file not found")

// Creates the metadata for a file that has been shared by the user
func (m *model) CreateSentFileViaInvitation(
	ctx context.Context,
	file domain.FullPath,
	invitationID string,
	key []byte,
) (*SentFileSchema, error) {
	log.Debug(fmt.Sprintf("Model.CreateSentFileViaInvitation: Storing sent file file=%+v", file))

	if existingFile, err := m.FindSentFile(ctx, file.DbId, file.Bucket, file.Path); err == nil {
		log.Debug("Model.CreateSentFileViaInvitation: file already in the collection")
		return existingFile, nil
	}

	newInstance := &SentFileSchema{
		ID: "",
		SentFileViaInvitationSchema: SentFileViaInvitationSchema{
			DbID:          file.DbId,
			Bucket:        file.Bucket,
			Path:          file.Path,
			InvitationId:  invitationID,
			BucketKey:     file.BucketKey,
			EncryptionKey: key,
		},
		CreatedAt: time.Now().UnixNano(),
	}

	return m.createSentFile(ctx, newInstance)
}

func (m *model) createSentFile(ctx context.Context, instance *SentFileSchema) (*SentFileSchema, error) {
	metaCtx, metaDbID, err := m.initSentFileModel(ctx)
	if err != nil && metaDbID == nil {
		return nil, err
	}

	instances := client.Instances{instance}
	res, err := m.threads.Create(metaCtx, *metaDbID, sentFileModelName, instances)
	if err != nil {
		return nil, err
	}

	log.Debug(fmt.Sprintf("Model.createSentFile: stored sent file res=%+v", res))

	id := res[0]
	return &SentFileSchema{
		ID:                          core.InstanceID(id),
		SentFileViaInvitationSchema: instance.SentFileViaInvitationSchema,
		CreatedAt:                   instance.CreatedAt,
	}, nil
}

// Finds the metadata of a file that has been shared by the user
func (m *model) FindSentFile(ctx context.Context, remoteDbID, bucket, path string) (*SentFileSchema, error) {
	metaCtx, dbID, err := m.initSentFileModel(ctx)
	if err != nil || dbID == nil {
		return nil, err
	}

	rawFiles, err := m.threads.Find(metaCtx, *dbID, sentFileModelName, db.Where("dbId").Eq(remoteDbID).And("bucket").Eq(bucket).And("path").Eq(path), &SentFileSchema{})
	if err != nil {
		return nil, err
	}

	if rawFiles == nil {
		return nil, errSentFileNotFound
	}

	files := rawFiles.([]*SentFileSchema)
	if len(files) == 0 {
		return nil, errSentFileNotFound
	}

	log.Debug(fmt.Sprintf("Model.FindSentFile: returning files=%+v", files))

	return files[0], nil
}

// Lists the metadata of files sent by the user
// If seek == "", will start looking from the beginning. If it's an existing ID it will start looking from that ID.
func (m *model) ListSentFiles(ctx context.Context, seek string, limit int) ([]*SentFileSchema, error) {
	metaCtx, dbID, err := m.initSentFileModel(ctx)
	if err != nil || dbID == nil {
		return nil, err
	}

	query := db.OrderByIDDesc().LimitTo(limit)

	if seek != "" {
		query = query.SeekID(core.InstanceID(seek))
	}

	rawFiles, err := m.threads.Find(metaCtx, *dbID, sentFileModelName, query, &SentFileSchema{})
	if err != nil {
		return nil, err
	}

	if rawFiles == nil {
		return []*SentFileSchema{}, nil
	}

	files := rawFiles.([]*SentFileSchema)
	return files, nil
}

// XXX: this is to reuse the builders in the sharing.go
func (sf *SentFileSchema) ReceivedFileSchema() *ReceivedFileSchema {
	return &ReceivedFileSchema{
		ID:        sf.ID,
		CreatedAt: sf.CreatedAt,
		ReceivedFileViaInvitationSchema: ReceivedFileViaInvitationSchema{
			DbID:          sf.DbID,
			Bucket:        sf.Bucket,
			Path:          sf.Path,
			InvitationId:  sf.InvitationId,
			BucketKey:     sf.BucketKey,
			EncryptionKey: sf.EncryptionKey,
		},
	}
}

func (m *model) initSentFileModel(ctx context.Context) (context.Context, *thread.ID, error) {
	metaCtx, dbID, err := m.getMetaThreadContext(ctx)
	if err != nil {
		return nil, nil, err
	}

	if err := m.threads.NewCollection(metaCtx, *dbID, GetSentFileCollectionConfig()); err != nil {
		log.Debug("initSentFileModel: collection already exists")
	}

	return metaCtx, dbID, nil
}

func GetSentFileCollectionConfig() db.CollectionConfig {
	return db.CollectionConfig{
		Name:   sentFileModelName,
		Schema: util.SchemaFromInstance(&SentFileSchema{}, false),
	}
}
