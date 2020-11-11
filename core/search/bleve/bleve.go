package bleve

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os/user"
	"path/filepath"

	"github.com/blevesearch/bleve/mapping"

	"github.com/FleekHQ/space-daemon/log"

	"github.com/FleekHQ/space-daemon/core/util"

	"github.com/FleekHQ/space-daemon/core/search"
	"github.com/blevesearch/bleve"
)

const DbFileName = "filesIndex.bleve"

type bleveSearchOption struct {
	dbPath string
}

type Option func(o *bleveSearchOption)

// bleveFilesSearchEngine is a files search engine that is backed by bleve
type bleveFilesSearchEngine struct {
	opts bleveSearchOption
	idx  bleve.Index
}

// Creates a new Bleve backed search engine for files and folders
func NewSearchEngine(opts ...Option) *bleveFilesSearchEngine {
	usr, _ := user.Current()

	searchOptions := bleveSearchOption{
		dbPath: filepath.Join(usr.HomeDir, ".fleek-space"),
	}

	for _, opt := range opts {
		opt(&searchOptions)
	}

	return &bleveFilesSearchEngine{
		opts: searchOptions,
	}
}

func (b *bleveFilesSearchEngine) Start() error {
	path := filepath.Join(b.opts.dbPath, DbFileName)

	var (
		idx bleve.Index
		err error
	)

	if util.DirEntryExists(path) {
		log.Debug("Opening existing search index")
		idx, err = bleve.Open(path)
	} else {
		log.Debug("Creating and opening new search index")
		indexMapping, err := getSearchIndexMapping()

		if err != nil {
			return err
		}

		idx, err = bleve.New(path, indexMapping)
	}

	if err != nil {
		return err
	}

	b.idx = idx

	return nil
}

func getSearchIndexMapping() (*mapping.IndexMappingImpl, error) {
	indexMapping := bleve.NewIndexMapping()
	indexMapping.DefaultAnalyzer = CustomerAnalyzerName

	filesMapping := bleve.NewDocumentMapping()

	// index the following fields
	nameFm := bleve.NewTextFieldMapping()
	filesMapping.AddFieldMappingsAt("ItemName", nameFm)
	extFm := bleve.NewTextFieldMapping()
	filesMapping.AddFieldMappingsAt("ItemExtension", extFm)
	pathFm := bleve.NewTextFieldMapping()
	filesMapping.AddFieldMappingsAt("ItemPath", pathFm)

	// ignore indexing the following fields of IndexRecord
	idFm := bleve.NewTextFieldMapping()
	idFm.Index = false
	filesMapping.AddFieldMappingsAt("Id", idFm)

	bucketFm := bleve.NewTextFieldMapping()
	bucketFm.Index = false
	filesMapping.AddFieldMappingsAt("BucketSlug", bucketFm)

	dbIdFm := bleve.NewTextFieldMapping()
	dbIdFm.Index = false
	filesMapping.AddFieldMappingsAt("DbId", dbIdFm)

	itemTypeFm := bleve.NewTextFieldMapping()
	itemTypeFm.Index = false
	filesMapping.AddFieldMappingsAt("ItemType", itemTypeFm)

	indexMapping.AddDocumentMapping("files", filesMapping)
	indexMapping.DefaultType = "files"

	return indexMapping, nil
}

func (b *bleveFilesSearchEngine) InsertFileData(
	ctx context.Context,
	data *search.InsertIndexRecord,
) (*search.IndexRecord, error) {
	indexId := generateIndexId(data.ItemName, data.ItemPath, data.BucketSlug, data.DbId)
	record := search.IndexRecord{
		Id:            indexId,
		ItemName:      data.ItemName,
		ItemExtension: data.ItemExtension,
		ItemPath:      data.ItemPath,
		ItemType:      data.ItemType,
		BucketSlug:    data.BucketSlug,
		DbId:          data.DbId,
	}

	if err := b.idx.Index(indexId, record); err != nil {
		return nil, err
	}

	return &record, nil
}

func (b *bleveFilesSearchEngine) DeleteFileData(
	ctx context.Context,
	data *search.DeleteIndexRecord,
) error {
	indexId := generateIndexId(data.ItemName, data.ItemPath, data.BucketSlug, data.DbId)
	return b.idx.Delete(indexId)
}

func (b *bleveFilesSearchEngine) QueryFileData(
	ctx context.Context,
	query string,
	limit int,
) ([]*search.IndexRecord, error) {
	matchQuery := bleve.NewMatchQuery(query)
	matchQuery.Fuzziness = 2

	prefixQuery := bleve.NewPrefixQuery(query)
	infixRegexQuery := bleve.NewRegexpQuery(fmt.Sprintf(".*%s.*", query)) // TODO: think of escaping invalid regex in query

	searchQuery := bleve.NewDisjunctionQuery(matchQuery, prefixQuery, infixRegexQuery)
	searchRequest := bleve.NewSearchRequest(searchQuery)
	searchRequest.Size = limit
	searchRequest.Fields = []string{"*"}

	searchResults, err := b.idx.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	records := make([]*search.IndexRecord, len(searchResults.Hits))
	for i, hit := range searchResults.Hits {
		records[i] = &search.IndexRecord{
			Id:            hit.Fields["Id"].(string),
			ItemName:      hit.Fields["ItemName"].(string),
			ItemExtension: hit.Fields["ItemExtension"].(string),
			ItemPath:      hit.Fields["ItemPath"].(string),
			ItemType:      hit.Fields["ItemType"].(string),
			BucketSlug:    hit.Fields["BucketSlug"].(string),
			DbId:          hit.Fields["DbId"].(string),
		}
	}

	return records, nil
}

func (b *bleveFilesSearchEngine) Shutdown() error {
	return b.idx.Close()
}

func generateIndexId(name, path, bucketSlug, dbId string) string {
	bytes := sha256.Sum256([]byte(name + path + bucketSlug + dbId))
	return fmt.Sprintf("%x", bytes)
}
