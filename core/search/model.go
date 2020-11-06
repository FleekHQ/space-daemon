package search

type IndexRecord struct {
	Id            string
	ItemName      string
	ItemExtension string
	ItemPath      string
	ItemType      string
	// Metadata here
	BucketSlug string
	DbId       string
}

type InsertIndexRecord struct {
	ItemName      string
	ItemExtension string
	ItemPath      string
	ItemType      string
	BucketSlug    string
	DbId          string
}

type DeleteIndexRecord struct {
	ItemName   string
	ItemPath   string
	BucketSlug string
	DbId       string // DbId is only required for shared content
}
