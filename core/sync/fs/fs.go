package fs

import (
	"fmt"
	"os"

	tc "github.com/FleekHQ/space-poc/core/textile/client"
	"github.com/FleekHQ/space-poc/log"
)

// Implementation to handle events from FS
type Handler struct {
	client *tc.TextileClient
}

// Creates a New File System Handler // TODO define what is needed as options like push notifications, etc
func NewHandler(textileClient *tc.TextileClient) *Handler {
	return &Handler{
		client: textileClient,
	}
}

func (h *Handler) OnCreate(path string, fileInfo os.FileInfo) {
	log.Info("Textile Handler: OnCreate", fmt.Sprintf("path:%s", path), fmt.Sprintf("fileInfo:%v", fileInfo))
}

func (h *Handler) OnRemove(path string, fileInfo os.FileInfo) {
	log.Info("Textile Handler: OnRemove", fmt.Sprintf("path:%s", path), fmt.Sprintf("fileInfo:%v", fileInfo))
}

func (h *Handler) OnWrite(path string, fileInfo os.FileInfo) {
	log.Info("Textile Handler: OnWrite", fmt.Sprintf("path:%s", path), fmt.Sprintf("fileInfo:%v", fileInfo))
}

func (h *Handler) OnRename(path string, fileInfo os.FileInfo, oldPath string) {
	log.Info(
		"Textile Handler: OnRename",
		fmt.Sprintf("path:%s", path),
		fmt.Sprintf("fileInfo:%v", fileInfo),
		fmt.Sprintf("path:%s", oldPath),
	)
}

func (h *Handler) OnMove(path string, fileInfo os.FileInfo, oldPath string) {
	log.Info(
		"Textile Handler: OnMove",
		fmt.Sprintf("path:%s", path),
		fmt.Sprintf("fileInfo:%v", fileInfo),
		fmt.Sprintf("path:%s", oldPath),
	)
}
