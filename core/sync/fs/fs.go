package fs

import (
	"context"
	"fmt"
	"os"

	path2 "github.com/ipfs/interface-go-ipfs-core/path"

	tc "github.com/FleekHQ/space-poc/core/textile/client"
	"github.com/FleekHQ/space-poc/log"
)

// Implementation to handle events from FS
type Handler struct {
	client *tc.TextileClient
	bucket *tc.TextileBucketRoot
}

// Creates a New File System Handler // TODO define what is needed as options like push notifications, etc
func NewHandler(textileClient *tc.TextileClient, bucketRoot *tc.TextileBucketRoot) *Handler {
	return &Handler{
		client: textileClient,
		bucket: bucketRoot,
	}
}

func (h *Handler) OnCreate(ctx context.Context, path string, fileInfo os.FileInfo) {
	log.Info(
		"FS Handler: OnCreate", fmt.Sprintf("path:%s", path),
		fmt.Sprintf("fileName:%s", fileInfo.Name()),
	)
	// TODO: Synchronizer lock check should ensure that no other operation is currently ongoing
	// with this path or its parent folder

	var result path2.Resolved
	var newRoot path2.Path
	var err error

	if fileInfo.IsDir() {
		result, newRoot, err = h.client.CreateDirectory(ctx, h.bucket.Key, path)
	} else {
		fileReader, err := os.Open(path)
		if err != nil {
			log.Error("Could not open file for upload", err)
			return
		}

		result, newRoot, err = h.client.UploadFile(ctx, h.bucket.Key, path, fileReader)
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
		fmt.Sprintf("path:%s", path),
		fmt.Sprintf("itemCid:%s", result.Cid()),
		fmt.Sprintf("rootCid:%s", newRoot.String()),
	)

	// TODO: Update synchronizer/store (maybe in a defer function)
}

func (h *Handler) OnRemove(ctx context.Context, path string, fileInfo os.FileInfo) {
	log.Info("FS Handler: OnRemove", fmt.Sprintf("path:%s", path), fmt.Sprintf("fileName:%s", fileInfo.Name()))
	// TODO: Also synchronizer lock check here

	err := h.client.DeleteDirOrFile(ctx, h.bucket.Key, path)

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
func (h *Handler) OnWrite(ctx context.Context, path string, fileInfo os.FileInfo) {
	log.Info("FS Handler: OnWrite", fmt.Sprintf("path:%s", path), fmt.Sprintf("fileName:%s", fileInfo.Name()))
	h.OnCreate(ctx, path, fileInfo)
}

func (h *Handler) OnRename(ctx context.Context, path string, fileInfo os.FileInfo, oldPath string) {
	log.Info(
		"Watcher Handler: OnRename",
		fmt.Sprintf("path:%s", path),
		fmt.Sprintf("fileName:%s", fileInfo.Name()),
		fmt.Sprintf("path:%s", oldPath),
	)
	h.OnRemove(ctx, oldPath, fileInfo)
	h.OnCreate(ctx, path, fileInfo)
}

func (h *Handler) OnMove(ctx context.Context, path string, fileInfo os.FileInfo, oldPath string) {
	log.Info(
		"Watcher Handler: OnMove",
		fmt.Sprintf("path:%s", path),
		fmt.Sprintf("fileName:%s", fileInfo.Name()),
		fmt.Sprintf("path:%s", oldPath),
	)
	h.OnRemove(ctx, oldPath, fileInfo)
	h.OnCreate(ctx, path, fileInfo)
}
