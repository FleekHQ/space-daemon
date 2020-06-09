package services

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/FleekHQ/space-poc/core/space/domain"
	"github.com/FleekHQ/space-poc/log"
)

type addItemWorkerRes struct {
	sourcePath string
	bucketPath string
	err        error
}

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

	entries := make([]domain.FileInfo, 0)
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
	s.watchFile(tmpFile.Name())

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

	return s.createFolder(ctx, path, key)
}

func (s *Space) createFolder(ctx context.Context, path string, key string) error {
	// NOTE: may need to change signature of createFolder if we need to return this info
	_, _, err := s.tc.CreateDirectory(ctx, key, path)

	if err != nil {
		log.Error(fmt.Sprintf("error creating folder in bucket %s with path %s", key, path), err)
		return err
	}

	return nil
}

func (s *Space) AddItems(ctx context.Context, sourcePaths []string, targetPath string) (domain.AddItemsResponse, error) {
	// TODO: add support for bucket slug
	key, err := s.getDefaultBucketKey(ctx)
	if err != nil {
		return domain.AddItemsResponse{}, err
	}
	return s.addItems(ctx, RemoveDuplicates(sourcePaths), targetPath, key)
}

func (s *Space) addItems(ctx context.Context, sourcePaths []string, targetPath string, bucketKey string) (domain.AddItemsResponse, error) {
	// check if all sourcePaths exist, else return err
	for _, sourcePath := range sourcePaths {
		if !PathExists(sourcePath) {
			return domain.AddItemsResponse{}, errors.New(fmt.Sprintf("path not found at %s", sourcePath))
		}
	}

	// create wait group with amount of sourcePaths
	var wg sync.WaitGroup
	wg.Add(len(sourcePaths))
	workerRes := make(chan addItemWorkerRes)

	results := make([]domain.AddItemResult, 0)
	errors := make([]domain.AddItemError, 0)

	// start parallel creation of paths
	for _, sourcePath := range sourcePaths {
		go func(pathInFs string) {
			if IsPathDir(pathInFs) {
				// create folder
				_, folderName := filepath.Split(pathInFs)
				targetBucketFolder := targetPath + "/" + folderName
				err := s.createFolder(ctx, targetBucketFolder, bucketKey)
				if err != nil {
					workerRes <- addItemWorkerRes{
						sourcePath: pathInFs,
						err:        err,
					}
				} else {
					res, err := s.addFolderRec(pathInFs, targetPath, ctx, bucketKey)
					if err != nil {
						workerRes <- addItemWorkerRes{
							sourcePath: pathInFs,
							err:        err,
						}
					} else {
						// send results to collect channel
						for _, r := range res.Results {
							workerRes <- addItemWorkerRes{
								sourcePath: r.SourcePath,
								bucketPath: r.BucketPath,
							}
						}

						for _, e := range res.Errors {
							workerRes <- addItemWorkerRes{
								sourcePath: e.SourcePath,
								err:        e.Error,
							}
						}
					}
				}
			} else {
				r, err := s.addFile(ctx, pathInFs, targetPath, bucketKey)
				if err != nil {
					// NOTE: we could also create a chan struct and pass path + err
					workerRes <- addItemWorkerRes{
						sourcePath: pathInFs,
						err:        err,
					}
				} else {
					workerRes <- addItemWorkerRes{
						sourcePath: r.SourcePath,
						bucketPath: r.BucketPath,
					}
				}
			}
			wg.Done()
		}(sourcePath)
	}
	go func() {
		// NOTE: this go routine is the only one writing to results arrays
		for chRes := range workerRes {
			if chRes.err != nil {
				errors = append(errors, domain.AddItemError{
					SourcePath: chRes.sourcePath,
					Error:      chRes.err,
				})
			} else {
				results = append(results, domain.AddItemResult{
					SourcePath: chRes.sourcePath,
					BucketPath: chRes.bucketPath,
				})
			}
		}
	}()

	wg.Wait()
	// closing channel to close err handling goroutine
	close(workerRes)

	return domain.AddItemsResponse{
		Results: results,
		Errors:  errors,
	}, nil
}

func (s *Space) addFolderRec(sourcePath string, targetPath string, ctx context.Context, bucketKey string) (domain.AddItemsResponse, error) {
	var folderSubPaths []string
	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		// avoid infinite recursion by excluding folder path
		if path != sourcePath {
			folderSubPaths = append(folderSubPaths, path)
		}
		return nil
	})

	if err != nil {
		log.Error(fmt.Sprintf("error reading folder path %s ", sourcePath), err)
		return domain.AddItemsResponse{}, err
	}
	// recursive call to addItems
	return s.addItems(ctx, folderSubPaths, targetPath, bucketKey)
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
