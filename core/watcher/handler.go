package watcher

import (
	"context"
	"fmt"
	"os"

	"github.com/FleekHQ/space-daemon/log"
)

// EventHandler
type EventHandler interface {
	OnCreate(ctx context.Context, path string, fileInfo os.FileInfo)
	OnRemove(ctx context.Context, path string, fileInfo os.FileInfo)
	OnWrite(ctx context.Context, path string, fileInfo os.FileInfo)
	OnRename(ctx context.Context, path string, fileInfo os.FileInfo, oldPath string)
	OnMove(ctx context.Context, path string, fileInfo os.FileInfo, oldPath string)
}

// Implements EventHandler and defaults to logging actions performed
type defaultWatcherHandler struct{}

func (h *defaultWatcherHandler) OnCreate(
	ctx context.Context,
	path string,
	fileInfo os.FileInfo,
) {
	log.Info("Default Watcher Handler: OnCreate", fmt.Sprintf("path:%s", path), fmt.Sprintf("fileInfo:%v", fileInfo))
}

func (h *defaultWatcherHandler) OnRemove(
	ctx context.Context,
	path string,
	fileInfo os.FileInfo,
) {
	log.Info("Default Watcher Handler: OnRemove", fmt.Sprintf("path:%s", path), fmt.Sprintf("fileInfo:%v", fileInfo))
}

func (h *defaultWatcherHandler) OnWrite(
	ctx context.Context,
	path string,
	fileInfo os.FileInfo,
) {
	log.Info("Default Watcher Handler: OnWrite", fmt.Sprintf("path:%s", path), fmt.Sprintf("fileInfo:%v", fileInfo))
}

func (h *defaultWatcherHandler) OnRename(
	ctx context.Context,
	path string,
	fileInfo os.FileInfo,
	oldPath string,
) {
	log.Info(
		"Default Watcher Handler: OnRename",
		fmt.Sprintf("path:%s", path),
		fmt.Sprintf("fileInfo:%v", fileInfo),
		fmt.Sprintf("path:%s", oldPath),
	)
}

func (h *defaultWatcherHandler) OnMove(
	ctx context.Context,
	path string,
	fileInfo os.FileInfo,
	oldPath string,
) {
	log.Info(
		"Default Watcher Handler: OnMove",
		fmt.Sprintf("path:%s", path),
		fmt.Sprintf("fileInfo:%v", fileInfo),
		fmt.Sprintf("path:%s", oldPath),
	)
}
