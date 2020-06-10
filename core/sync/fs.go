package sync


import (
	"context"
	"fmt"
	"os"

	path2 "github.com/ipfs/interface-go-ipfs-core/path"

	"github.com/FleekHQ/space-poc/log"
)


func (h *watcherHandler) OnCreate(ctx context.Context, path string, fileInfo os.FileInfo) {
	log.Info(
		"FS Handler: OnCreate", fmt.Sprintf("path:%s", path),
		fmt.Sprintf("fileName:%s", fileInfo.Name()),
	)
	// TODO: Synchronizer lock check should ensure that no other operation is currently ongoing
	// with this path or its parent folder

	var result path2.Resolved
	var newRoot path2.Path
	var err error

	key, exists := h.bs.getOpenFileBucketKey(path)
	if !exists {
		msg := fmt.Sprintf("error: could not find path %s", path)
		log.Error(msg, fmt.Errorf(msg))
		return
	}

	if fileInfo.IsDir() {
		existsOnTextile, err := h.client.FolderExists(ctx, key, path)
		if err != nil {
			log.Error("Could not check if folder exists on textile", err)
			return
		}

		if existsOnTextile {
			log.Info("Folder alerady exists on textile")
			return
		}

		result, newRoot, err = h.client.CreateDirectory(ctx, key, path)
	} else {
		fileReader, err := os.Open(path)
		if err != nil {
			log.Error("Could not open file for upload", err)
			return
		}

		existsOnTextile, err := h.client.FileExists(ctx, key, path, fileReader)
		if err != nil {
			log.Error("Could not check if file exists on textile", err)
			return
		}

		if existsOnTextile {
			log.Info("File alerady exists on textile")
			return
		}

		result, newRoot, err = h.client.UploadFile(ctx, key, path, fileReader)
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

func (h *watcherHandler) OnRemove(ctx context.Context, path string, fileInfo os.FileInfo) {
	log.Info("FS Handler: OnRemove", fmt.Sprintf("path:%s", path), fmt.Sprintf("fileName:%s", fileInfo.Name()))
	// TODO: Also synchronizer lock check here

	key, exists := h.bs.getOpenFileBucketKey(path)
	if !exists {
		msg := fmt.Sprintf("error: could not find path %s", path)
		log.Error(msg, fmt.Errorf(msg))
		return
	}

	_, err := h.client.DeleteDirOrFile(ctx, key, path)

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

	/*key, exists := h.bs.getOpenFileBucketKey(path)
	if !exists {
		msg := fmt.Sprintf("error: could not find path %s", path)
		log.Error(msg, fmt.Errorf(msg))
		return
	}
	// LOCK PER BUCKET
	h.bs.textileClient.UploadFile()*/
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