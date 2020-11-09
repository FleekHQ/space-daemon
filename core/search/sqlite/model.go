package sqlite

import "gorm.io/gorm"

type SearchIndexRecord struct {
	gorm.Model
	ItemName      string `gorm:"index:idx_name_path_bucket,unique"`
	ItemExtension string `gorm:"size:10"`
	ItemPath      string `gorm:"index:idx_name_path_bucket,unique"`
	ItemType      string
	BucketSlug    string `gorm:"index:idx_name_path_bucket,unique"`
	DbId          string `gorm:"index"`
}
