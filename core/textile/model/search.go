package model

import (
	"context"
	"path"

	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/util"

	"github.com/pkg/errors"
	"github.com/textileio/go-threads/db"

	"github.com/FleekHQ/space-daemon/log"
	"github.com/textileio/go-threads/api/client"
	core "github.com/textileio/go-threads/core/db"
)

type SearchItemType string

const (
	FileItem      SearchItemType = "FILE"
	DirectoryItem SearchItemType = "DIRECTORY"
)

type SearchIndexRecord struct {
	ID            core.InstanceID `json:"_id"`
	ItemName      string          `json:"itemName"`
	ItemExtension string          `json:"itemExtension"`
	ItemPath      string          `json:"itemPath"`
	ItemType      string          `json:"itemType"` // currently either FILE/DIRECTORY
	// Metadata here
	BucketSlug string `json:"bucketSlug"`
	DbId       string `json:"dbId"`
}

const searchIndexModelName = "SearchIndexMetadata"

func (m *model) InitSearchIndexCollection(ctx context.Context) error {
	log.Debug("Model.InitSearchIndexCollection: Initializing db")
	_, err := m.initSearchModel(ctx)
	return err
}

func (m *model) initSearchModel(ctx context.Context) (*thread.ID, error) {
	metaCtx, dbId, err := m.getMetaThreadContext(ctx)
	if err != nil || dbId == nil {
		return nil, err
	}

	_, err = m.threads.GetCollectionInfo(metaCtx, *dbId, searchIndexModelName)
	if err == nil {
		// collection already exists
		return dbId, nil
	}

	// create search collection
	if err := m.threads.NewCollection(metaCtx, *dbId, db.CollectionConfig{
		Name:   searchIndexModelName,
		Schema: util.SchemaFromInstance(&SearchIndexRecord{}, false),
		Indexes: []db.Index{
			{
				Path:   "itemName",
				Unique: false,
			},
			{
				Path:   "itemPath",
				Unique: false,
			},
			{
				Path:   "itemType",
				Unique: false,
			},
			{
				Path:   "itemExtension",
				Unique: false,
			},
		},
	}); err != nil {
		log.Warn("Creating Search collection failed", "error:"+err.Error())
		return nil, err
	}

	return dbId, nil
}

func (m *model) UpdateSearchIndexRecord(
	ctx context.Context,
	name, itemPath string,
	itemType SearchItemType,
	bucketSlug, dbId string,
) (*SearchIndexRecord, error) {
	log.Debug("Model.UpdateSearchIndexRecord: Initializing db")
	metaCtx, metaDbID, err := m.initBucketModel(ctx)
	if err != nil || metaDbID == nil {
		return nil, err
	}

	// if record already exists avoid duplication
	query := db.Where("itemName").Eq(name).And("itemPath").Eq(itemPath).And("bucketSlug").Eq(bucketSlug)
	if dbId != "" {
		query = query.And("dbId").Eq(dbId)
	}
	existingRecords, err := m.threads.Find(metaCtx, *metaDbID, searchIndexModelName, query, &SearchIndexRecord{})
	if err == nil && len(existingRecords.([]*SearchIndexRecord)) > 0 {
		return existingRecords.([]*SearchIndexRecord)[0], nil
	}

	newInstance := &SearchIndexRecord{
		ID:            "",
		ItemName:      name,
		ItemExtension: path.Ext(name),
		ItemPath:      itemPath,
		ItemType:      string(itemType),
		DbId:          dbId,
		BucketSlug:    bucketSlug,
	}

	instances := client.Instances{newInstance}
	log.Debug("Model.UpdateSearchIndexRecord: Creating instance")

	res, err := m.threads.Create(metaCtx, *metaDbID, searchIndexModelName, instances)
	if err != nil {
		return nil, err
	}
	log.Debug("Model.UpdateSearchIndexRecord: Instance creation successful", "instanceId:"+newInstance.ID.String())

	id := res[0]
	return &SearchIndexRecord{
		ID:         core.InstanceID(id),
		ItemName:   newInstance.ItemName,
		ItemPath:   newInstance.ItemPath,
		ItemType:   newInstance.ItemType,
		BucketSlug: newInstance.BucketSlug,
		DbId:       newInstance.DbId,
	}, nil
}

func (m *model) QuerySearchIndex(ctx context.Context, query string) ([]*SearchIndexRecord, error) {
	metaCtx, metaDbID, err := m.initBucketModel(ctx)
	if err != nil || metaDbID == nil {
		return nil, err
	}

	log.Debug("Model.QuerySearchIndex: start search", "query:"+query)
	res, err := m.threads.Find(
		metaCtx,
		*metaDbID,
		searchIndexModelName,
		db.Where("itemName").Eq(query).Or(db.Where("itemExtension").Eq(query)).LimitTo(20),
		&SearchIndexRecord{},
	)
	if err != nil {
		return nil, errors.Wrap(err, "search query failed")
	}

	return res.([]*SearchIndexRecord), nil
}

// DeleteSearchIndexRecords updates the search index by deleting records that match the name and path.
func (m *model) DeleteSearchIndexRecord(ctx context.Context, name, path, bucketSlug, dbId string) error {
	metaCtx, metaDbID, err := m.initBucketModel(ctx)
	if err != nil || metaDbID == nil {
		return err
	}

	query := db.Where("itemName").Eq(name).And("itemPath").Eq(path).And("bucketSlug").Eq(bucketSlug)
	if dbId != "" {
		query = query.And("dbId").Eq(dbId)
	}

	res, err := m.threads.Find(metaCtx, *metaDbID, searchIndexModelName, query, &SearchIndexRecord{})
	if err != nil {
		return err
	}
	records := res.([]*SearchIndexRecord)
	if len(records) == 0 {
		return nil
	}

	var instanceIds []string
	for _, record := range records {
		instanceIds = append(instanceIds, record.ID.String())
	}

	return m.threads.Delete(metaCtx, *metaDbID, searchIndexModelName, instanceIds)
}
