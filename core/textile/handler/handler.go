package handler

import (
	"fmt"
	"os"

	tc "github.com/FleekHQ/space-poc/core/textile/client"
	"github.com/FleekHQ/space-poc/log"
)

type TextileHandler struct {
	client *tc.TextileClient
}

func New(textileClient *tc.TextileClient) *TextileHandler {
	return &TextileHandler{
		client: textileClient,
	}
}

func (h *TextileHandler) OnCreate(path string, fileInfo os.FileInfo) {
	log.Info("Textile Handler: OnCreate", fmt.Sprintf("path:%s", path), fmt.Sprintf("fileInfo:%v", fileInfo))
}

func (h *TextileHandler) OnRemove(path string, fileInfo os.FileInfo) {
	log.Info("Textile Handler: OnRemove", fmt.Sprintf("path:%s", path), fmt.Sprintf("fileInfo:%v", fileInfo))
}

func (h *TextileHandler) OnWrite(path string, fileInfo os.FileInfo) {
	log.Info("Textile Handler: OnWrite", fmt.Sprintf("path:%s", path), fmt.Sprintf("fileInfo:%v", fileInfo))
}

func (h *TextileHandler) OnRename(path string, fileInfo os.FileInfo, oldPath string) {
	log.Info(
		"Textile Handler: OnRename",
		fmt.Sprintf("path:%s", path),
		fmt.Sprintf("fileInfo:%v", fileInfo),
		fmt.Sprintf("path:%s", oldPath),
	)
}

func (h *TextileHandler) OnMove(path string, fileInfo os.FileInfo, oldPath string) {
	log.Info(
		"Textile Handler: OnMove",
		fmt.Sprintf("path:%s", path),
		fmt.Sprintf("fileInfo:%v", fileInfo),
		fmt.Sprintf("path:%s", oldPath),
	)
}
