package watcher

import (
	"fmt"
	"os"

	"github.com/FleekHQ/space-poc/log"
)

// EventHandler
type EventHandler interface {
	OnCreate(path string, fileInfo os.FileInfo)
	OnRemove(path string, fileInfo os.FileInfo)
	OnWrite(path string, fileInfo os.FileInfo)
	OnRename(path string, fileInfo os.FileInfo, oldPath string)
	OnMove(path string, fileInfo os.FileInfo, oldPath string)
}

// Implements EventHandler and defaults to logging actions performed
type defaultWatcherHandler struct{}

func (h *defaultWatcherHandler) OnCreate(path string, fileInfo os.FileInfo) {
	log.Info("Default Watcher Handler: OnCreate", fmt.Sprintf("path:%s", path), fmt.Sprintf("fileInfo:%v", fileInfo))
}

func (h *defaultWatcherHandler) OnRemove(path string, fileInfo os.FileInfo) {
	log.Info("Default Watcher Handler: OnRemove", fmt.Sprintf("path:%s", path), fmt.Sprintf("fileInfo:%v", fileInfo))
}

func (h *defaultWatcherHandler) OnWrite(path string, fileInfo os.FileInfo) {
	log.Info("Default Watcher Handler: OnWrite", fmt.Sprintf("path:%s", path), fmt.Sprintf("fileInfo:%v", fileInfo))
}

func (h *defaultWatcherHandler) OnRename(path string, fileInfo os.FileInfo, oldPath string) {
	log.Info(
		"Default Watcher Handler: OnRename",
		fmt.Sprintf("path:%s", path),
		fmt.Sprintf("fileInfo:%v", fileInfo),
		fmt.Sprintf("path:%s", oldPath),
	)
}

func (h *defaultWatcherHandler) OnMove(path string, fileInfo os.FileInfo, oldPath string) {
	log.Info(
		"Default Watcher Handler: OnMove",
		fmt.Sprintf("path:%s", path),
		fmt.Sprintf("fileInfo:%v", fileInfo),
		fmt.Sprintf("path:%s", oldPath),
	)
}
