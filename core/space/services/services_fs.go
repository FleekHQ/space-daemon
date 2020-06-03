package services

import (
	"context"
	"errors"
	"github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/core/space/domain"
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
func (s *Space) GetPathInfo(ctx context.Context, path string) (domain.PathInfo, error) {
	res := domain.PathInfo{
		Path:     "test.txt",
		IpfsHash: "testhash",
		IsDir:    false,
	}

	return res, nil
}



