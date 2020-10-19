package fsds

import (
	"fmt"
	"os"

	"github.com/FleekHQ/space-daemon/core/space"
)

var DefaultBucketName = "personal"

type dataSourceConfig struct {
	tlfSources []*TLFDataSource
}

type FSDataSourceConfig func(config *dataSourceConfig)

func WithTLFDataSource(source *TLFDataSource) FSDataSourceConfig {
	return func(config *dataSourceConfig) {
		config.tlfSources = append(config.tlfSources, source)
	}
}

// Configure the default 'Files` data source to be included as a data source
func WithFilesDataSources(service space.Service) FSDataSourceConfig {
	basePath := fmt.Sprintf("%cFiles", os.PathSeparator)
	return WithTLFDataSource(&TLFDataSource{
		name:         "Files",
		basePath:     basePath,
		FSDataSource: &filesDataSource{service: service},
	})
}

// Configure the default 'Shared With Me` data source to be included as a data source
func WithSharedWithMeDataSources(service space.Service) FSDataSourceConfig {
	basePath := fmt.Sprintf("%cShared With Me", os.PathSeparator)
	return WithTLFDataSource(&TLFDataSource{
		name:     "Shared With Me",
		basePath: basePath,
		FSDataSource: &sharedWithMeDataSource{
			service:     service,
			maxDirLimit: 1000,
			cache:       make(map[string]*sharedFileEntry),
		},
	})
}

var blackListedDirEntryNames = map[string]bool{
	// OSX specific special directories
	".Trashes":              true,
	".localized":            true,
	".fseventsd":            true,
	".ql_disablethumbnails": true,
	".ql_disablecache":      true,
	// special space empty directory file
	".keep": true,
}
