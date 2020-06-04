package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/core/space/domain"
	"github.com/FleekHQ/space-poc/log"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (s *Space) ListDir(ctx context.Context) ([]domain.DirEntry, error) {
	path := s.cfg.GetString(config.SpaceFolderPath, "")
	if path == "" {
		return nil, errors.New("config does not have a valid path specified")
	}

	var files = make([]domain.DirEntry, 0)

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		entry := domain.DirEntry{
			Path:          path,
			IsDir:         info.IsDir(),
			Name:          info.Name(),
			SizeInBytes:   strconv.FormatInt(info.Size(), 10),
			Created:       info.ModTime().String(),
			Updated:       info.ModTime().String(),
			FileExtension: strings.Replace(filepath.Ext(info.Name()),".", "", -1),
		}
		files = append(files, entry)
		return nil
	})

	if err != nil {
		return nil, err
	}
	return files, nil
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
	tmpFile, err := ioutil.TempFile(cfg.FolderPath, "*-" + path)
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



