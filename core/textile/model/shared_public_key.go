package model

import (
	"context"
	"errors"
	"time"

	"github.com/FleekHQ/space-daemon/log"
	"github.com/textileio/go-threads/api/client"
	core "github.com/textileio/go-threads/core/db"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/db"
	"github.com/textileio/go-threads/util"
)

type SharedPublicKeySchema struct {
	ID        core.InstanceID `json:"_id"`
	DbID      string          `json:"dbId"`
	PublicKey string          `json:"public_key"`
	UpdatedAt int64           `json:"updated_at"`
	CreatedAt int64           `json:"created_at"`
}

const sharedPublicKeyModel = "SharedPublicKey"

var errSharedPublicKeyNotFound = errors.New("Shared public key not found")

// Creates the metadata for a shared public key
func (m *model) CreateSharedPublicKey(ctx context.Context, pubKey string) (*SharedPublicKeySchema, error) {
	log.Debug("Model.CreateSharedPublicKey: Storing shared public key " + pubKey)
	if existingPublicKey, err := m.FindSharedPublicKey(ctx, pubKey); err == nil {
		log.Debug("Model.CreateSharedPublicKey: Shared public key already in collection")
		return existingPublicKey, nil
	}

	log.Debug("Model.CreateSharedPublicKey: Initializing db")
	metaCtx, metaDbID, err := m.initSharedPublicKey(ctx)
	if err != nil && metaDbID == nil {
		return nil, err
	}

	now := time.Now().UnixNano()

	newInstance := &SharedPublicKeySchema{
		ID:        "",
		PublicKey: pubKey,
		UpdatedAt: now,
		CreatedAt: now,
	}

	instances := client.Instances{newInstance}
	log.Debug("Model.CreateSharedPublicKey: Creating instance")

	res, err := m.threads.Create(metaCtx, *metaDbID, sharedPublicKeyModel, instances)
	if err != nil {
		return nil, err
	}
	log.Debug("Model.CreateSharedPublicKey: stored shared public key " + newInstance.PublicKey)

	id := res[0]
	return &SharedPublicKeySchema{
		ID:        core.InstanceID(id),
		DbID:      newInstance.DbID,
		PublicKey: newInstance.PublicKey,
		UpdatedAt: newInstance.UpdatedAt,
		CreatedAt: newInstance.CreatedAt,
	}, nil
}

// Finds the metadata of a shared public key
func (m *model) FindSharedPublicKey(ctx context.Context, pubKey string) (*SharedPublicKeySchema, error) {
	metaCtx, dbID, err := m.initReceivedFileModel(ctx)
	if err != nil || dbID == nil {
		return nil, err
	}

	rawKeys, err := m.threads.Find(metaCtx, *dbID, sharedPublicKeyModel, db.Where("public_key").Eq(pubKey), &SharedPublicKeySchema{})
	if err != nil {
		return nil, err
	}

	if rawKeys == nil {
		return nil, errReceivedFileNotFound
	}

	files := rawKeys.([]*SharedPublicKeySchema)
	if len(files) == 0 {
		return nil, errReceivedFileNotFound
	}
	log.Debug("Model.FindReceivedFile: returning shared public key " + files[0].PublicKey)
	return files[0], nil
}

func (m *model) initSharedPublicKey(ctx context.Context) (context.Context, *thread.ID, error) {
	metaCtx, dbID, err := m.getMetaThreadContext(ctx)
	if err != nil {
		return nil, nil, err
	}

	managedKey, err := m.kc.GetManagedThreadKey(metaThreadName)
	if err != nil {
		log.Error("error getting managed thread key", err)
		return nil, nil, err
	}

	if err = m.threads.NewDB(metaCtx, *dbID, db.WithNewManagedThreadKey(managedKey)); err != nil {
		log.Debug("initSharedPublicKey: db already exists")
	}
	if err := m.threads.NewCollection(metaCtx, *dbID, db.CollectionConfig{
		Name:   sharedPublicKeyModel,
		Schema: util.SchemaFromInstance(&SharedPublicKeySchema{}, false),
	}); err != nil {
		log.Debug("initSharedPublicKey: collection already exists")
	}

	return metaCtx, dbID, nil
}

const listSharedPublicKeysLimit = 128

func (m *model) ListSharedPublicKeys(ctx context.Context) ([]*SharedPublicKeySchema, error) {
	metaCtx, dbID, err := m.initSharedPublicKey(ctx)
	if err != nil && dbID == nil {
		return nil, err
	}

	query := &db.Query{}
	query.Limit = listSharedPublicKeysLimit
	query.Sort.FieldPath = "CreatedAt"
	query.Sort.Desc = false

	rawKeys, err := m.threads.Find(metaCtx, *dbID, sharedPublicKeyModel, query, &SharedPublicKeySchema{})
	if rawKeys == nil {
		return []*SharedPublicKeySchema{}, nil
	}
	keys := rawKeys.([]*SharedPublicKeySchema)
	return keys, nil
}
