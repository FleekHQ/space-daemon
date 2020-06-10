package services

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/FleekHQ/space-poc/core/space/domain"
	"github.com/FleekHQ/space-poc/log"
)

func (s *Space) listDirAtPath(
	ctx context.Context,
	bucketKey, path string,
	listSubfolderContent bool,
) ([]domain.FileInfo, error) {
	dir, err := s.tc.ListDirectory(ctx, bucketKey, path)
	if err != nil {
		log.Error("Error in ListDir", err)
		return nil, err
	}

	relPathRegex := regexp.MustCompile(`\/ip(f|n)s\/[^\/]*(?P<relPath>\/.*)`)

	entries := make([]domain.FileInfo, 0)
	for _, item := range dir.Item.Items {
		if item.Name == ".textileseed" || item.Name == ".textile" {
			continue
		}

		paths := relPathRegex.FindStringSubmatch(item.Path)
		var relPath string
		if len(paths) > 2 {
			relPath = relPathRegex.FindStringSubmatch(item.Path)[2]
		} else {
			relPath = item.Path
		}

		entry := domain.FileInfo{
			DirEntry: domain.DirEntry{
				Path:          relPath,
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
		entries = append(entries, entry)

		if item.IsDir && listSubfolderContent {
			newEntries, err := s.listDirAtPath(ctx, bucketKey, path+"/"+item.Name, true)
			if err != nil {
				return nil, err
			}
			entries = append(entries, newEntries...)
		}
	}

	return entries, nil
}

func (s *Space) ListDir(ctx context.Context) ([]domain.FileInfo, error) {
	buckets, err := s.tc.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}

	if len(buckets) == 0 {
		return nil, errors.New("Could not find buckets")
	}

	// List the root directory
	listPath := ""

	return s.listDirAtPath(ctx, buckets[0].Key, listPath, true)
}

func (s *Space) OpenFile(ctx context.Context, path string, bucketSlug string) (domain.OpenFileInfo, error) {
	// TODO : handle bucketslug for multiple buckets. For now default to personal bucket
	key, err := s.getDefaultBucketKey(ctx)
	if err != nil {
		return domain.OpenFileInfo{}, err
	}

	// write file copy to temp folder
	cfg := s.GetConfig(ctx)
	_, fileName := filepath.Split(path)
	// NOTE: the pattern of the file ensures that it retains extension. e.g (rand num) + filename/path
	tmpFile, err := ioutil.TempFile(cfg.FolderPath, "*-"+fileName)
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
	// register temp file in watcher
	addWatchFile := domain.AddWatchFile{
		LocalPath:  tmpFile.Name(),
		BucketPath: path,
		BucketKey:  key,
	}
	err = s.watchFile(addWatchFile)
	if err != nil {
		log.Error(fmt.Sprintf("error adding file to watch path %s from bucket %s in bucketpath %s", tmpFile.Name(), key, path), err)
		return domain.OpenFileInfo{}, err
	}

	// return file handle
	return domain.OpenFileInfo{
		Location: tmpFile.Name(),
	}, nil
}

func (s *Space) getDefaultBucketKey(ctx context.Context) (string, error) {
	buckets, err := s.tc.ListBuckets(ctx)
	if err != nil {
		log.Error("error while fetching buckets in OpenFile", err)
		return "", err
	}
	if len(buckets) == 0 {
		log.Error("no buckets found in OpenFile", err)
		return "", err
	}
	key := buckets[0].Key
	return key, nil
}

func (s *Space) CreateFolder(ctx context.Context, path string) error {
	key, err := s.getDefaultBucketKey(ctx)
	if err != nil {
		return err
	}

	if _, err := s.createFolder(ctx, path, key); err != nil {
		return err
	}

	return nil
}

func (s *Space) createFolder(ctx context.Context, path string, key string) (string, error) {
	// NOTE: may need to change signature of createFolder if we need to return this info
	_, root, err := s.tc.CreateDirectory(ctx, key, path)

	if err != nil {
		log.Error(fmt.Sprintf("error creating folder in bucket %s with path %s", key, path), err)
		return "", err
	}

	return root.String(), nil
}

func (s *Space) AddItems(ctx context.Context, sourcePaths []string, targetPath string) (<-chan domain.AddItemResult, error) {
	// TODO: add support for bucket slug
	key, err := s.getDefaultBucketKey(ctx)
	if err != nil {
		return nil, err
	}
	results := make(chan domain.AddItemResult)
	go func() {
		s.addItems(ctx, RemoveDuplicates(sourcePaths), targetPath, key, results)
		close(results)
	}()

	return results, nil
}

func (s *Space) addItems(ctx context.Context, sourcePaths []string, targetPath string, bucketKey string, results chan<- domain.AddItemResult) error {
	// check if all sourcePaths exist, else return err
	for _, sourcePath := range sourcePaths {
		if !PathExists(sourcePath) {
			return errors.New(fmt.Sprintf("path not found at %s", sourcePath))
		}
	}

	// NOTE: sequential upload of files and folders
	for _, sourcePath := range sourcePaths {
		if IsPathDir(sourcePath) {
			s.handleAddItemFolder(ctx, sourcePath, targetPath, bucketKey, results)
		} else {
			// add files
			r, err := s.addFile(ctx, sourcePath, targetPath, bucketKey)
			if err != nil {
				results <- domain.AddItemResult{
					SourcePath: sourcePath,
					Error:      err,
				}
				// next iteration
				continue
			}
			results <- domain.AddItemResult{
				SourcePath: sourcePath,
				BucketPath: r.BucketPath,
			}
		}
	}

	return nil
}

func (s *Space) handleAddItemFolder(ctx context.Context, sourcePath string, targetPath string, bucketKey string, results chan<- domain.AddItemResult) {
	// create folder
	_, folderName := filepath.Split(sourcePath)
	targetBucketFolder := targetPath + "/" + folderName
	folderBucketPath, err := s.createFolder(ctx, targetBucketFolder, bucketKey)
	if err != nil {
		results <- domain.AddItemResult{
			SourcePath: sourcePath,
			Error:      err,
		}
		return
	}

	results <- domain.AddItemResult{
		SourcePath: sourcePath,
		BucketPath: folderBucketPath,
	}
	err = s.addFolderRec(sourcePath, targetBucketFolder, ctx, bucketKey, results)
	if err != nil {
		results <- domain.AddItemResult{
			SourcePath: sourcePath,
			Error:      err,
		}
		return
	}
}

func (s *Space) addFolderRec(sourcePath string, targetPath string, ctx context.Context, bucketKey string, results chan<- domain.AddItemResult) error {
	var folderSubPaths []string

	// NOTE: only reading each folder one level deep since this function is recursive
	// if we use Walk we would need to track source paths across recursive calls to avoid duplicates
	files, err := ioutil.ReadDir(sourcePath)

	if err != nil {
		log.Error(fmt.Sprintf("error reading folder path %s ", sourcePath), err)
		return err
	}

	for _, file := range files {
		if file.Name() != sourcePath {
			folderSubPaths = append(folderSubPaths, sourcePath+"/"+file.Name())
		}
	}

	// recursive call to addItems
	return s.addItems(ctx, folderSubPaths, targetPath, bucketKey, results)
}

// Working with a file
func (s *Space) addFile(ctx context.Context, sourcePath string, targetPath string, bucketKey string) (domain.AddItemResult, error) {
	// get sourcePath to io.Reader
	f, err := os.Open(sourcePath)
	if err != nil {
		log.Error(fmt.Sprintf("error opening path %s", sourcePath), err)
		return domain.AddItemResult{}, err
	}

	defer f.Close()

	_, fileName := filepath.Split(sourcePath)

	targetPathBucket := targetPath + "/" + fileName

	// NOTE: could modify addFile to return back more info for processing
	_, root, err := s.tc.UploadFile(ctx, bucketKey, targetPathBucket, f)
	if err != nil {
		log.Error(fmt.Sprintf("error creating targetPath %s in bucket %s", targetPathBucket, bucketKey), err)
		return domain.AddItemResult{}, err
	}

	return domain.AddItemResult{
		SourcePath: sourcePath,
		BucketPath: root.String(),
	}, err
}
