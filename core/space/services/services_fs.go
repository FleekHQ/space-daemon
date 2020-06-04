package services

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/FleekHQ/space-poc/core/space/domain"
	"github.com/FleekHQ/space-poc/log"
)

func (s *Space) listDirAtPath(
	ctx context.Context,
	bucketKey, path string,
	entriesPtr *[]domain.FileInfo,
	listSubfolderContent bool,
) ([]domain.FileInfo, error) {
	dir, err := s.tc.ListDirectory(ctx, bucketKey, path)
	if err != nil {
		log.Error("Error in ListDir", err)
		return nil, err
	}

	for _, item := range dir.Item.Items {
		if item.Name == ".textileseed" || item.Name == ".textile" {
			continue
		}

		entry := domain.FileInfo{
			DirEntry: domain.DirEntry{
				Path:          item.Path,
				IsDir:         item.IsDir,
				Name:          item.Name,
				SizeInBytes:   strconv.FormatInt(item.Size, 10),
				FileExtension: strings.Replace(filepath.Ext(item.Name), ".", "", -1),
				// TODO: Get these fields from Textile Buckets
				Created: "",
				Updated: "",
			},
			IpfsHash: item.Cid,
		}
		*entriesPtr = append(*entriesPtr, entry)

		if item.IsDir && listSubfolderContent {
			s.listDirAtPath(ctx, bucketKey, path+"/"+item.Name, entriesPtr, true)
		}
	}

	return *entriesPtr, nil
}

func (s *Space) ListDir(ctx context.Context) ([]domain.FileInfo, error) {
	buckets, err := s.tc.ListBuckets()
	if err != nil {
		return nil, err
	}

	entries := make([]domain.FileInfo, 0)

	// List the root directory
	listPath := ""

	return s.listDirAtPath(ctx, buckets[0].Key, listPath, &entries, true)
}

// TODO: implement this
func (s *Space) GetPathInfo(ctx context.Context, path string) (domain.FileInfo, error) {
	res := domain.FileInfo{}

	return res, nil
}

func (s *Space) OpenFile(ctx context.Context, path string, bucketSlug string) (domain.OpenFileInfo, error) {
	// TODO : handle bucketslug for multiple buckets. For now default to personal bucket
	buckets, err := s.tc.ListBuckets()
	if err != nil {
		log.Error("error while fetching buckets in OpenFile", err)
		return domain.OpenFileInfo{}, err
	}
	if len(buckets) == 0 {
		log.Error("no buckets found in OpenFile", err)
		return domain.OpenFileInfo{}, err
	}
	key := buckets[0].Key

	// write file copy to temp folder
	cfg := s.GetConfig(ctx)
	// NOTE: the pattern of the file ensures that it retains extension. e.g (rand num) + filename/path
	tmpFile, err := ioutil.TempFile(cfg.FolderPath, "*-"+path)
	if err != nil {
		log.Error("cannot create temp file while executing OpenFile", err)
		return domain.OpenFileInfo{}, err
	}
	defer tmpFile.Close()

	// look for path in textile
	err = s.tc.GetFile(ctx, key, path, tmpFile)
	if err != nil {
		log.Error(fmt.Sprintf("error retrieving file from bucket %s in path %s", key, path), err)
		return domain.OpenFileInfo{}, err
	}
	// TODO: register temp file in watcher

	// return file handle
	return domain.OpenFileInfo{
		Location: tmpFile.Name(),
	}, nil
}
