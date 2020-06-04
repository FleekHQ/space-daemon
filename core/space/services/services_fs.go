package services

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/core/space/domain"
	"github.com/FleekHQ/space-poc/log"
)

func (s *Space) listDirAtPath(
	ctx context.Context,
	bucketKey, path string,
	entriesPtr *[]domain.DirEntry,
	listSubfolderContent bool,
) ([]domain.DirEntry, error) {
	log.Debug("Processing dir" + path)
	dir, err := s.tc.ListDirectory(ctx, bucketKey, path)
	if err != nil {
		return nil, err
	}

	for _, item := range dir.Item.Items {
		log.Debug("Processing file " + item.Name)
		if item.Name == ".textileseed" || item.Name == ".textile" {
			continue
		}

		entry := domain.DirEntry{
			Path:          item.Path,
			IsDir:         item.IsDir,
			Name:          item.Name,
			SizeInBytes:   strconv.FormatInt(item.Size, 10),
			FileExtension: strings.Replace(filepath.Ext(item.Name), ".", "", -1),
			// TODO: Get these fields from Textile Buckets
			Created: "",
			Updated: "",
		}
		*entriesPtr = append(*entriesPtr, entry)

		if item.IsDir && listSubfolderContent {
			s.listDirAtPath(ctx, bucketKey, path+"/"+item.Name, entriesPtr, true)
		}
	}

	return *entriesPtr, nil
}

func (s *Space) ListDir(ctx context.Context) ([]domain.DirEntry, error) {
	path := s.cfg.GetString(config.SpaceFolderPath, "")
	if path == "" {
		return nil, errors.New("config does not have a valid path specified")
	}

	buckets, err := s.tc.ListBuckets()
	if err != nil {
		return nil, err
	}

	entries := make([]domain.DirEntry, 0)

	// List the root directory
	listPath := ""

	return s.listDirAtPath(ctx, buckets[0].Key, listPath, &entries, true)
}

// TODO: implement this
func (s *Space) GetPathInfo(ctx context.Context, path string) (domain.PathInfo, error) {
	res := domain.PathInfo{
		Path:     "test.txt",
		IpfsHash: "testhash",
		IsDir:    false,
	}

	return res, nil
}
