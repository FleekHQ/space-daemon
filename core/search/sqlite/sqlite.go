package sqlite

import (
	"context"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"gorm.io/gorm/logger"

	"github.com/FleekHQ/space-daemon/core/search"

	"github.com/pkg/errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const DbFileName = "filesIndex.db"

type sqliteSearchOption struct {
	dbPath   string
	logLevel logger.LogLevel
}

type Option func(o *sqliteSearchOption)

// sqliteFilesSearchEngine is a files search engine that is backed by sqlite
type sqliteFilesSearchEngine struct {
	db   *gorm.DB
	opts sqliteSearchOption
}

// Creates a new SQLite backed search engine for files and folders
func NewSearchEngine(opts ...Option) *sqliteFilesSearchEngine {
	usr, _ := user.Current()

	searchOptions := sqliteSearchOption{
		dbPath: filepath.Join(usr.HomeDir, ".fleek-space"),
	}

	for _, opt := range opts {
		opt(&searchOptions)
	}

	return &sqliteFilesSearchEngine{
		db:   nil,
		opts: searchOptions,
	}
}

func (s *sqliteFilesSearchEngine) Start() error {
	dsn := filepath.Join(s.opts.dbPath, DbFileName)

	if db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(s.opts.logLevel),
	}); err != nil {
		return errors.Wrap(err, "failed to open database")
	} else {
		s.db = db
	}

	return s.db.AutoMigrate(&SearchIndexRecord{})
}

func (s *sqliteFilesSearchEngine) InsertFileData(ctx context.Context, data *search.InsertIndexRecord) (*search.IndexRecord, error) {
	record := SearchIndexRecord{
		ItemName:      data.ItemName,
		ItemExtension: data.ItemExtension,
		ItemPath:      data.ItemPath,
		ItemType:      data.ItemPath,
		BucketSlug:    data.BucketSlug,
		DbId:          data.DbId,
	}
	result := s.db.Create(&record)

	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "UNIQUE constraint failed") {
			return nil, errors.New("a similar file has already been inserted")
		}
		return nil, result.Error
	}

	return modelToIndexRecord(&record), nil
}

func (s *sqliteFilesSearchEngine) DeleteFileData(ctx context.Context, data *search.DeleteIndexRecord) error {
	stmt := s.db.Where(
		"item_name = ? AND item_path = ? AND bucket_slug = ?",
		data.ItemName,
		data.ItemPath,
		data.BucketSlug,
	)
	if data.DbId != "" {
		stmt = stmt.Where("dbId = ?", data.DbId)
	}

	result := stmt.Delete(&SearchIndexRecord{})

	return result.Error
}

func (s *sqliteFilesSearchEngine) QueryFileData(ctx context.Context, query string, limit int) ([]*search.IndexRecord, error) {
	var records []*SearchIndexRecord
	result := s.db.Where(
		"LOWER(item_name) LIKE ? OR LOWER(item_extension) = ?",
		"%"+strings.ToLower(query)+"%",
		strings.ToLower(query),
	).Limit(limit).Find(&records)

	if result.Error != nil {
		return nil, result.Error
	}

	searchResults := make([]*search.IndexRecord, len(records))
	for i, record := range records {
		searchResults[i] = modelToIndexRecord(record)
	}

	return searchResults, nil
}

func (s *sqliteFilesSearchEngine) Shutdown() error {
	db, err := s.db.DB()
	if err != nil {
		return err
	}

	return db.Close()
}

func modelToIndexRecord(model *SearchIndexRecord) *search.IndexRecord {
	return &search.IndexRecord{
		Id:            strconv.Itoa(int(model.ID)),
		ItemName:      model.ItemName,
		ItemExtension: model.ItemExtension,
		ItemPath:      model.ItemPath,
		ItemType:      model.ItemType,
		BucketSlug:    model.BucketSlug,
		DbId:          model.DbId,
	}
}
