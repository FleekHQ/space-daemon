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
	"sync"
	"time"

	"github.com/FleekHQ/space-daemon/core/textile"

	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/log"
)

func (s *Space) CreateBucket(ctx context.Context, slug string) (textile.Bucket, error) {
	b, err := s.tc.CreateBucket(ctx, slug)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (s *Space) listDirAtPath(
	ctx context.Context,
	b textile.Bucket,
	path string,
	listSubfolderContent bool,
) ([]domain.FileInfo, error) {
	dir, err := b.ListDirectory(ctx, path)
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
				Created: time.Now().Format(time.RFC3339),
				Updated: time.Now().Format(time.RFC3339),
			},
			IpfsHash: item.Cid,
		}
		entries = append(entries, entry)

		if item.IsDir && listSubfolderContent {
			newEntries, err := s.listDirAtPath(ctx, b, path+"/"+item.Name, true)
			if err != nil {
				return nil, err
			}
			entries = append(entries, newEntries...)
		}
	}

	return entries, nil
}

// ListDir returns children entries at path in a bucket
func (s *Space) ListDir(ctx context.Context, path string) ([]domain.FileInfo, error) {
	b, err := s.tc.GetDefaultBucket(ctx)
	if err != nil {
		return nil, err
	}

	if b == nil {
		return nil, errors.New("Could not find buckets")
	}

	return s.listDirAtPath(ctx, b, path, false)
}

// ListDirs lists all children entries at path in a bucket
// Unlike ListDir, it includes all subfolders children recursively
func (s *Space) ListDirs(ctx context.Context, path string) ([]domain.FileInfo, error) {
	// TODO: add support for multiple buckets
	b, err := s.tc.GetDefaultBucket(ctx)
	if err != nil {
		return nil, err
	}

	if b == nil {
		return nil, errors.New("Could not find buckets")
	}

	return s.listDirAtPath(ctx, b, path, true)
}

func (s *Space) OpenFile(ctx context.Context, path string) (domain.OpenFileInfo, error) {
	var filePath string
	var err error
	// check if file exists in sync
	// TODO : handle bucketslug for multiple buckets. For now default to personal bucket
	b, err := s.tc.GetDefaultBucket(ctx)
	if err != nil {
		return domain.OpenFileInfo{}, err
	}
	if filePath, exists := s.sync.GetOpenFilePath(b.Slug(), path); exists {
		// sanity check in case file was deleted or moved
		if PathExists(filePath) {
			// return file handle
			return domain.OpenFileInfo{
				Location: filePath,
			}, nil
		}
	}

	// else, open new file on FS
	filePath, err = s.openFileOnFs(ctx, path, b)
	if err != nil {
		return domain.OpenFileInfo{}, err
	}

	// return file handle
	return domain.OpenFileInfo{
		Location: filePath,
	}, nil
}

func (s *Space) openFileOnFs(ctx context.Context, path string, b textile.Bucket) (string, error) {
	// write file copy to temp folder
	cfg := s.GetConfig(ctx)
	_, fileName := filepath.Split(path)
	// NOTE: the pattern of the file ensures that it retains extension. e.g (rand num) + filename/path
	tmpFile, err := ioutil.TempFile(cfg.AppPath, "*-"+fileName)
	if err != nil {
		log.Error("cannot create temp file while executing OpenFile", err)
		return "", err
	}
	defer tmpFile.Close()

	// look for path in textile
	err = b.GetFile(ctx, path, tmpFile)
	if err != nil {
		log.Error(fmt.Sprintf("error retrieving file from bucket %s in path %s", b.Key(), path), err)
		return "", err
	}
	// register temp file in watcher
	addWatchFile := domain.AddWatchFile{
		LocalPath:  tmpFile.Name(),
		BucketPath: path,
		BucketKey:  b.Key(),
	}
	err = s.sync.AddFileWatch(addWatchFile)
	if err != nil {
		log.Error(fmt.Sprintf("error adding file to watch path %s from bucket %s in bucketpath %s", tmpFile.Name(), b.Key(), path), err)
		return "", err
	}
	return tmpFile.Name(), nil
}

func (s *Space) CreateFolder(ctx context.Context, path string) error {
	b, err := s.tc.GetDefaultBucket(ctx)
	if err != nil {
		return err
	}

	if _, err := s.createFolder(ctx, path, b); err != nil {
		return err
	}

	return nil
}

func (s *Space) createFolder(ctx context.Context, path string, b textile.Bucket) (string, error) {
	// NOTE: may need to change signature of createFolder if we need to return this info
	_, root, err := b.CreateDirectory(ctx, path)

	if err != nil {
		log.Error(fmt.Sprintf("error creating folder in bucket %s with path %s", b.Key(), path), err)
		return "", err
	}

	return root.String(), nil
}

func (s *Space) AddItems(ctx context.Context, sourcePaths []string, targetPath string) (<-chan domain.AddItemResult, domain.AddItemsResponse, error) {
	// check if all sourcePaths exist, else return err
	for _, sourcePath := range sourcePaths {
		if !PathExists(sourcePath) {
			return nil, domain.AddItemsResponse{}, errors.New(fmt.Sprintf("path not found at %s", sourcePath))
		}
	}
	// TODO: add support for bucket slug
	b, err := s.tc.GetDefaultBucket(ctx)
	if err != nil {
		return nil, domain.AddItemsResponse{}, err
	}
	results := make(chan domain.AddItemResult)

	totalsRes, err := getTotals(RemoveDuplicates(sourcePaths))
	if err != nil {
		return nil, domain.AddItemsResponse{}, err
	}
	go func() {
		s.addItems(ctx, RemoveDuplicates(sourcePaths), targetPath, b, results)
		close(results)
	}()

	return results, totalsRes, nil
}

// get totals for addItems operation
func getTotals(sourcePaths []string) (domain.AddItemsResponse, error) {
	var wg sync.WaitGroup
	wg.Add(len(sourcePaths))
	filesRes := make(chan domain.AddItemsResponse)
	results := make([]domain.AddItemsResponse, 0)
	for _, sourcePath := range sourcePaths {
		go func(pathInFs string) {
			defer wg.Done()
			if IsPathDir(pathInFs) {
				// counting folder as a file in total with 0 bytes
				filesRes <- domain.AddItemsResponse{
					TotalFiles: 1,
					TotalBytes: 0,
				}
				// get recursive
				var folderSubPaths []string
				files, err := ioutil.ReadDir(pathInFs)
				if err != nil {
					log.Error(fmt.Sprintf("error reading folder path %s ", pathInFs), err)
					filesRes <- domain.AddItemsResponse{
						Error: err,
					}
					return
				}
				for _, file := range files {
					subPath := pathInFs + "/" + file.Name()
					if subPath != pathInFs {
						folderSubPaths = append(folderSubPaths, subPath)
					}
				}
				folderSubPathsRes, err := getTotals(folderSubPaths)
				if err != nil {
					filesRes <- domain.AddItemsResponse{
						Error: err,
					}
					return
				}
				filesRes <- folderSubPathsRes
			} else {
				// get totals bytes
				fi, err := os.Stat(pathInFs)
				if err != nil {
					log.Error(fmt.Sprintf("error getting file size %s ", pathInFs), err)
					filesRes <- domain.AddItemsResponse{
						Error: err,
					}
					return
				}
				// get the size
				filesRes <- domain.AddItemsResponse{
					TotalFiles: 1,
					TotalBytes: fi.Size(),
				}
			}
		}(sourcePath)
	}

	resultsDone := make(chan struct{})
	var collectErr error
	totalResult := domain.AddItemsResponse{}

	go func() {
		// collect results
		for chRes := range filesRes {
			if chRes.Error != nil {
				collectErr = chRes.Error
				continue
			}
			results = append(results, chRes)
		}

		for _, res := range results {
			totalResult.TotalBytes += res.TotalBytes
			totalResult.TotalFiles += res.TotalFiles
		}
		resultsDone <- struct{}{}
	}()

	wg.Wait()
	// closing channel to close results handling goroutine
	close(filesRes)
	// wait for all results to finish
	<-resultsDone

	if collectErr != nil {
		return totalResult, collectErr
	}

	return totalResult, nil
}

func (s *Space) addItems(ctx context.Context, sourcePaths []string, targetPath string, b textile.Bucket, results chan<- domain.AddItemResult) error {
	// NOTE: sequential upload of files and folders
	for _, sourcePath := range sourcePaths {
		if IsPathDir(sourcePath) {
			s.handleAddItemFolder(ctx, sourcePath, targetPath, b, results)
		} else {
			// add files
			r, err := s.addFile(ctx, sourcePath, targetPath, b)
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
				Bytes:      r.Bytes,
			}
		}
	}

	return nil
}

func (s *Space) handleAddItemFolder(ctx context.Context, sourcePath string, targetPath string, b textile.Bucket, results chan<- domain.AddItemResult) {
	// create folder
	_, folderName := filepath.Split(sourcePath)
	targetBucketFolder := targetPath + "/" + folderName
	folderBucketPath, err := s.createFolder(ctx, targetBucketFolder, b)
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
	err = s.addFolderRec(sourcePath, targetBucketFolder, ctx, b, results)
	if err != nil {
		results <- domain.AddItemResult{
			SourcePath: sourcePath,
			Error:      err,
		}
		return
	}
}

func (s *Space) addFolderRec(sourcePath string, targetPath string, ctx context.Context, b textile.Bucket, results chan<- domain.AddItemResult) error {
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
	return s.addItems(ctx, folderSubPaths, targetPath, b, results)
}

// Working with a file
func (s *Space) addFile(ctx context.Context, sourcePath string, targetPath string, b textile.Bucket) (domain.AddItemResult, error) {
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
	_, root, err := b.UploadFile(ctx, targetPathBucket, f)
	if err != nil {
		log.Error(fmt.Sprintf("error creating targetPath %s in bucket %s", targetPathBucket, b.Key()), err)
		return domain.AddItemResult{}, err
	}

	fi, err := f.Stat()
	var fileSize int64 = 0
	if err == nil {
		fileSize = fi.Size()
	}
	return domain.AddItemResult{
		SourcePath: sourcePath,
		BucketPath: root.String(),
		Bytes:      fileSize,
	}, err
}
