package sync

import (
	"context"
	"fmt"
	"os"

	ipfspath "github.com/ipfs/interface-go-ipfs-core/path"

	"github.com/FleekHQ/space-daemon/core/textile"
	"github.com/FleekHQ/space-daemon/log"
)

func (h *watcherHandler) OnCreate(ctx context.Context, path string, fileInfo os.FileInfo) {
	log.Info(
		"FS Handler: OnCreate", fmt.Sprintf("path:%s", path),
		fmt.Sprintf("fileName:%s", fileInfo.Name()),
	)
	// TODO: Synchronizer lock check should ensure that no other operation is currently ongoing
	// with this path or its parent folder

	var result ipfspath.Resolved
	var newRoot ipfspath.Path
	var err error

	watchInfo, exists := h.bs.getOpenFileBucketSlugAndPath(path)
	if !exists {
		msg := fmt.Sprintf("error: could not find path %s", path)
		log.Error(msg, fmt.Errorf(msg))
		return
	}

	bucketSlug := watchInfo.BucketSlug
	bucketPath := watchInfo.BucketPath

	b, err := h.bs.textileClient.GetBucket(ctx, bucketSlug, nil)
	if err != nil {
		msg := fmt.Sprintf("error: could not find bucket with slug %s", bucketSlug)
		log.Error(msg, fmt.Errorf(msg))
		return
	}

	if fileInfo.IsDir() {
		existsOnTextile, err := b.DirExists(ctx, path)
		if err != nil {
			log.Error("Could not check if folder exists on textile", err)
			return
		}

		if existsOnTextile {
			log.Info("Folder alerady exists on textile")
			return
		}

		result, newRoot, err = b.CreateDirectory(ctx, path)
	} else {
		existsOnTextile, err := b.FileExists(ctx, path)
		if err != nil {
			log.Error("Could not check if file exists on textile", err)
			return
		}

		if existsOnTextile {
			log.Info("File alerady exists on textile")
			return
		}

		fileReader, err := os.Open(path)
		if err != nil {
			log.Error("Could not open file for upload", err)
			return
		}

		result, newRoot, err = b.UploadFile(ctx, bucketPath, fileReader)
	}

	if err != nil {
		log.Error("Uploading to textile failed", err, fmt.Sprintf("path:%s", path))
		return
	}
	if err = result.IsValid(); err != nil {
		log.Error("Uploading to textile not valid", err, fmt.Sprintf("path:%s", path))
		return
	}

	log.Info(
		"Successfully uploaded item to textile",
		fmt.Sprintf("bucketPath:%s", bucketPath),
		fmt.Sprintf("itemCid:%s", result.Cid()),
		fmt.Sprintf("rootCid:%s", newRoot.String()),
	)

	// TODO: Update synchronizer/store (maybe in a defer function)
}

func (h *watcherHandler) OnRemove(ctx context.Context, path string, fileInfo os.FileInfo) {
	log.Info("FS Handler: OnRemove", fmt.Sprintf("path:%s", path), fmt.Sprintf("fileName:%s", fileInfo.Name()))
	// TODO: Also synchronizer lock check here

	watchInfo, exists := h.bs.getOpenFileBucketSlugAndPath(path)
	if !exists {
		msg := fmt.Sprintf("error: could not find path %s", path)
		log.Error(msg, fmt.Errorf(msg))
		return
	}
	bucketSlug := watchInfo.BucketSlug
	bucketPath := watchInfo.BucketPath

	b, err := h.bs.textileClient.GetBucket(ctx, bucketSlug, nil)
	if err != nil {
		msg := fmt.Sprintf("error: could not find bucket with slug %s", bucketSlug)
		log.Error(msg, fmt.Errorf(msg))
		return
	}

	_, err = b.DeleteDirOrFile(ctx, bucketPath)

	if err != nil {
		log.Error("Deleting from textile failed", err, fmt.Sprintf("path:%s", path))
		return
	}

	log.Info(
		"Successfully deleted item from textile",
		fmt.Sprintf("path:%s", path),
	)
	// TODO: Update synchronizer/store (maybe in a defer function)
}

// OnWrite is invoked when a new file is created or files content is updated
func (h *watcherHandler) OnWrite(ctx context.Context, path string, fileInfo os.FileInfo) {
	log.Info("FS Handler: OnWrite", fmt.Sprintf("path:%s", path), fmt.Sprintf("fileName:%s", fileInfo.Name()))

	watchInfo, exists := h.bs.getOpenFileBucketSlugAndPath(path)
	if !exists {
		msg := fmt.Sprintf("error: could not find path %s", path)
		log.Error(msg, fmt.Errorf(msg))
		return
	}

	var b textile.Bucket
	var err error
	bucketSlug := watchInfo.BucketSlug
	bucketPath := watchInfo.BucketPath

	if watchInfo.IsRemote {
		b, err = h.bs.textileClient.GetBucket(ctx, bucketSlug, &textile.GetBucketForRemoteFileInput{
			Bucket: bucketSlug,
			DbID:   watchInfo.DbId,
			Path:   watchInfo.BucketPath,
		})
	} else {
		b, err = h.bs.textileClient.GetBucket(ctx, bucketSlug, nil)
	}

	if err != nil {
		msg := fmt.Sprintf("error: could not find bucket with slug %s", bucketSlug)
		log.Error(msg, fmt.Errorf(msg))
		return
	}
	fileReader, err := os.Open(path)
	if err != nil {
		log.Error("Could not open file for upload", err)
		return
	}

	_, _, err = b.UploadFile(ctx, bucketPath, fileReader)
	if err != nil {
		msg := fmt.Sprintf("error: could not sync file at path %s to bucket %s as %s", path, bucketSlug, bucketPath)
		log.Error(msg, fmt.Errorf(msg))
		return
	}
	msg := fmt.Sprintf("success syncing file at path %s to bucket %s as %s", path, bucketSlug, bucketPath)
	log.Printf(msg)

}

func (h *watcherHandler) OnRename(ctx context.Context, path string, fileInfo os.FileInfo, oldPath string) {
	log.Info(
		"Watcher Handler: OnRename",
		fmt.Sprintf("path:%s", path),
		fmt.Sprintf("fileName:%s", fileInfo.Name()),
		fmt.Sprintf("path:%s", oldPath),
	)
	h.OnRemove(ctx, oldPath, fileInfo)
	h.OnCreate(ctx, path, fileInfo)
}

func (h *watcherHandler) OnMove(ctx context.Context, path string, fileInfo os.FileInfo, oldPath string) {
	log.Info(
		"Watcher Handler: OnMove",
		fmt.Sprintf("path:%s", path),
		fmt.Sprintf("fileName:%s", fileInfo.Name()),
		fmt.Sprintf("path:%s", oldPath),
	)
	h.OnRemove(ctx, oldPath, fileInfo)
	h.OnCreate(ctx, path, fileInfo)
}
