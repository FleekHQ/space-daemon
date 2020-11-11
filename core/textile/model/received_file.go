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

type ReceivedFileViaPublicLinkSchema struct {
	PublicIpfsHash string `json:"publicIpfsHash"`
	FilePassword   string `json:"filePassword"`
	FileName       string `json:"fileName"`
	FileSize       string `json:"fileSize"`
}

type ReceivedFileViaInvitationSchema struct {
	DbID          string `json:"dbId"`
	Bucket        string `json:"bucket"`
	Path          string `json:"path"`
	InvitationId  string `json:"invitationId"`
	BucketKey     string `json:"bucketKey"`
	EncryptionKey []byte `json:"encryptionKey"`
}

// ReceivedFileSchema represents data of files shared with a user
// A file is shared with a user either by direct invite to the user or through a publicly accessible link
type ReceivedFileSchema struct {
	ID        core.InstanceID `json:"_id"`
	Accepted  bool            `json:"accepted"`
	CreatedAt int64           `json:"created_at"`
	ReceivedFileViaInvitationSchema
	ReceivedFileViaPublicLinkSchema
}

func (r ReceivedFileSchema) IsPublicLinkReceived() bool {
	return r.InvitationId == ""
}

const receivedFileModelName = "ReceivedFile"

var errReceivedFileNotFound = errors.New("Received file not found")

// Creates the metadata for a file that has been shared to the user
func (m *model) CreateReceivedFileViaInvitation(
	ctx context.Context,
	file domain.FullPath,
	invitationID string,
	accepted bool,
	key []byte,
) (*ReceivedFileSchema, error) {
	log.Debug("Model.CreateReceivedFileViaInvitation: Storing received file", "file:"+file.Path)
	if existingFile, err := m.FindReceivedFile(ctx, file.DbId, file.Bucket, file.Path); err == nil {
		log.Debug("Model.CreateReceivedFileViaInvitation: Bucket already in collection")
		return existingFile, nil
	}

	newInstance := &ReceivedFileSchema{
		ID: "",
		ReceivedFileViaInvitationSchema: ReceivedFileViaInvitationSchema{
			DbID:          file.DbId,
			Bucket:        file.Bucket,
			Path:          file.Path,
			InvitationId:  invitationID,
			BucketKey:     file.BucketKey,
			EncryptionKey: key,
		},
		Accepted:  accepted,
		CreatedAt: time.Now().UnixNano(),
	}

	return m.createReceivedFile(ctx, newInstance)
}

func (m *model) CreateReceivedFileViaPublicLink(
	ctx context.Context,
	ipfsHash string,
	password string,
	filename string,
	fileSize string,
	accepted bool,
) (*ReceivedFileSchema, error) {
	log.Debug(
		"Model.CreateReceivedFileViaPublicLink: Storing received file",
		"hash:"+ipfsHash,
		"filename:"+filename,
	)
	if existingFile, err := m.FindPublicLinkReceivedFile(ctx, ipfsHash); err == nil {
		log.Debug("Model.CreateReceivedFileViaPublicLink: similar file already shared with user")
		return existingFile, nil
	}

	newInstance := &ReceivedFileSchema{
		ReceivedFileViaPublicLinkSchema: ReceivedFileViaPublicLinkSchema{
			PublicIpfsHash: ipfsHash,
			FilePassword:   password,
			FileName:       filename,
			FileSize:       fileSize,
		},
		ReceivedFileViaInvitationSchema: ReceivedFileViaInvitationSchema{
			EncryptionKey: []byte(""),
		},
		Accepted:  accepted,
		CreatedAt: time.Now().UnixNano(),
	}

	return m.createReceivedFile(ctx, newInstance)
}

func (m *model) createReceivedFile(ctx context.Context, instance *ReceivedFileSchema) (*ReceivedFileSchema, error) {
	metaCtx, metaDbID, err := m.initReceivedFileModel(ctx)
	if err != nil && metaDbID == nil {
		return nil, err
	}

	instances := client.Instances{instance}
	res, err := m.threads.Create(metaCtx, *metaDbID, receivedFileModelName, instances)
	if err != nil {
		return nil, err
	}
	log.Debug("Model.createReceivedFile: stored received file", "instance_id:"+res[0])

	id := res[0]
	return &ReceivedFileSchema{
		ID:                              core.InstanceID(id),
		ReceivedFileViaInvitationSchema: instance.ReceivedFileViaInvitationSchema,
		ReceivedFileViaPublicLinkSchema: instance.ReceivedFileViaPublicLinkSchema,
		Accepted:                        instance.Accepted,
		CreatedAt:                       instance.CreatedAt,
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

func (m *model) FindPublicLinkReceivedFile(ctx context.Context, ipfsHash string) (*ReceivedFileSchema, error) {
	metaCtx, dbID, err := m.initReceivedFileModel(ctx)
	if err != nil || dbID == nil {
		return nil, err
	}

	rawFiles, err := m.threads.Find(
		metaCtx,
		*dbID,
		receivedFileModelName,
		db.Where("publicIpfsHash").Eq(ipfsHash),
		&ReceivedFileSchema{},
	)
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
	log.Debug("Model.findPublicLinkReceivedFile: returning file with hash " + files[0].PublicIpfsHash)
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

func (m *model) ListReceivedPublicFiles(
	ctx context.Context,
	cidHash string,
	accepted bool,
) ([]*ReceivedFileSchema, error) {
	metaCtx, dbID, err := m.initReceivedFileModel(ctx)
	if err != nil || dbID == nil {
		return nil, err
	}

	query := db.Where("accepted").Eq(accepted).And("publicIpfsHash").Eq(cidHash)

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

	if err := m.threads.NewCollection(metaCtx, *dbID, GetReceivedFileCollectionConfig()); err != nil {
		log.Debug("initReceivedFileModel: collection already exists")
	}

	return metaCtx, dbID, nil
}

func GetReceivedFileCollectionConfig() db.CollectionConfig {
	return db.CollectionConfig{
		Name:   receivedFileModelName,
		Schema: util.SchemaFromInstance(&ReceivedFileSchema{}, false),
	}
}
