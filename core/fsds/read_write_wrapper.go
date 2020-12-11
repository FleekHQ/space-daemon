package fsds

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/FleekHQ/space-daemon/core/space/domain"

	"github.com/FleekHQ/space-daemon/log"
)

// Wrapper around space files read and write logic.
// On close, it pushes changes to space.Service
type SpaceFilesHandler struct {
	service       SyncService
	localFile     *os.File
	localFilePath string
	remotePath    string
	bucketName    string
	editted       bool
}

type SyncService interface {
	AddItemWithReader(ctx context.Context, reader io.Reader, targetPath, bucketName string) (domain.AddItemResult, error)
}

func OpenSpaceFilesHandler(
	service SyncService,
	localFilePath,
	remoteFilePath,
	bucketName string,
) *SpaceFilesHandler {
	return &SpaceFilesHandler{
		service:       service,
		localFilePath: localFilePath,
		localFile:     nil,
		remotePath:    remoteFilePath,
		bucketName:    bucketName,
		editted:       false,
	}
}

func (s *SpaceFilesHandler) Read(ctx context.Context, b []byte, offset int64) (int, error) {
	log.Debug(
		"Reading bytes from file handler",
		"path:"+s.remotePath,
		"bucket:"+s.bucketName,
		fmt.Sprintf("offset:%d", offset),
	)
	s.openLocalFile()

	_, err := s.localFile.Seek(offset, io.SeekStart)
	if err != nil {
		return 0, err
	}
	return s.localFile.Read(b)
}

func (s *SpaceFilesHandler) Write(ctx context.Context, b []byte, offset int64) (int, error) {
	log.Debug(
		"Writing bytes to file handler",
		"path:"+s.remotePath,
		"bucket:"+s.bucketName,
		fmt.Sprintf("offset:%d", offset),
	)
	s.openLocalFile()

	_, err := s.localFile.Seek(offset, io.SeekStart)
	if err != nil {
		return 0, err
	}

	n, err := s.localFile.Write(b)
	if err == nil {
		s.editted = true
	}
	return n, err
}

func (s *SpaceFilesHandler) Close(ctx context.Context) error {
	log.Debug("Closing access to SpaceFileHandler", "remotePath:"+s.remotePath, "localPath:"+s.localFilePath)
	defer func() {
		if s.localFile != nil {
			// background synchronizer should handle sync on close
			s.localFile.Close()
			s.localFile = nil
		}
	}()

	//if s.editted && s.localFile != nil {
	//	_, err := s.localFile.Seek(0, 0)
	//	if err != nil {
	//		log.Error("Error seeking local file to beginning for upload", err)
	//		return err
	//	}
	//
	//	_, err = s.service.AddItemWithReader(
	//		ctx,
	//		s.localFile,
	//		s.remotePath,
	//		s.bucketName,
	//	)
	//	if err != nil {
	//		return err
	//	}
	//}

	return nil
}

// Stats for now always reads stats from local file
func (s *SpaceFilesHandler) Stats(ctx context.Context) (*DirEntry, error) {
	s.openLocalFile()
	info, err := os.Stat(s.localFilePath)
	if err != nil {
		return nil, err
	}

	return NewDirEntryFromFileInfo(info, s.remotePath), nil
}

func (s *SpaceFilesHandler) Truncate(ctx context.Context, size uint64) error {
	s.openLocalFile()
	return s.localFile.Truncate(int64(size))
}

func (s *SpaceFilesHandler) openLocalFile() {
	if s.localFile != nil {
		return
	}

	s.localFile, _ = os.OpenFile(s.localFilePath, os.O_APPEND|os.O_RDWR, os.ModeAppend)
}
