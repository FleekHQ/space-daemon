package model

import (
	"context"
	"path"
	"strings"

	"github.com/FleekHQ/space-daemon/core/search"

	"github.com/FleekHQ/space-daemon/log"
)

type SearchItemType string

const (
	FileItem                 SearchItemType = "FILE"
	DirectoryItem            SearchItemType = "DIRECTORY"
	DefaultSearchResultLimit int            = 20
)

type SearchIndexRecord search.IndexRecord

func (m *model) InitSearchIndexCollection(ctx context.Context) error {
	log.Debug("Model.InitSearchIndexCollection: Initializing db")
	return m.fsearch.Start()
}

func (m *model) UpdateSearchIndexRecord(
	ctx context.Context,
	name, itemPath string,
	itemType SearchItemType,
	bucketSlug, dbId string,
) (*SearchIndexRecord, error) {
	log.Debug("Model.UpdateSearchIndexRecord: Initializing db")
	if instance, err := m.fsearch.InsertFileData(ctx, &search.InsertIndexRecord{
		ItemName:      name,
		ItemExtension: strings.Replace(path.Ext(name), ".", "", -1),
		ItemPath:      itemPath,
		ItemType:      string(itemType),
		BucketSlug:    bucketSlug,
		DbId:          dbId,
	}); err != nil {
		return nil, err
	} else {
		return (*SearchIndexRecord)(instance), nil
	}
}

func (m *model) QuerySearchIndex(ctx context.Context, query string) ([]*SearchIndexRecord, error) {
	res, err := m.fsearch.QueryFileData(ctx, query, DefaultSearchResultLimit)
	if err != nil {
		return nil, err
	}

	result := make([]*SearchIndexRecord, len(res))
	for i, item := range res {
		result[i] = (*SearchIndexRecord)(item)
	}

	return result, nil
}

// DeleteSearchIndexRecords updates the fsearch index by deleting records that match the name and path.
func (m *model) DeleteSearchIndexRecord(ctx context.Context, name, path, bucketSlug, dbId string) error {
	return m.fsearch.DeleteFileData(ctx, &search.DeleteIndexRecord{
		ItemName:   name,
		ItemPath:   path,
		BucketSlug: bucketSlug,
		DbId:       dbId,
	})
}
