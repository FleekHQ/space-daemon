package files

import (
	"context"
	"io"
	"os"

	"github.com/FleekHQ/space-daemon/log"

	"github.com/FleekHQ/space-daemon/core/space"
)

// Wrapper around space files read and write logic.
// On close, it pushes changes to space.Service
type SpaceFilesHandler struct {
	service    space.Service
	localFile  *os.File
	remotePath string
	bucketName string
	editted    bool
}

func OpenSpaceFilesHandler(
	service space.Service,
	localFilePath,
	remoteFilePath,
	bucketName string,
) (*SpaceFilesHandler, error) {
	localFile, err := os.Open(localFilePath)
	if err != nil {
		return nil, err
	}
	return &SpaceFilesHandler{
		service:    service,
		localFile:  localFile,
		remotePath: remoteFilePath,
		bucketName: bucketName,
		editted:    false,
	}, nil
}

func (s *SpaceFilesHandler) Read(ctx context.Context, b []byte, offset int64) (int, error) {
	_, err := s.localFile.Seek(offset, io.SeekStart)
	if err != nil {
		return 0, err
	}
	return s.localFile.Read(b)
}

func (s *SpaceFilesHandler) Write(ctx context.Context, b []byte, offset int64) (int, error) {
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
	log.Debug("Closing access to SpaceFileHandler", "remotePath:"+s.remotePath, "localPath:"+s.localFile.Name())
	defer s.localFile.Close()

	if s.editted {
		_, err := s.localFile.Seek(0, 0)
		if err != nil {
			log.Error("Error seeking local file to beginning for upload", err)
			return err
		}

		_, err = s.service.AddItemWithReader(
			ctx,
			s.localFile,
			s.remotePath,
			s.bucketName,
		)
		if err != nil {
			return err
		}
	}

	log.Debug("Adding files update complete", "remotePath:"+s.remotePath, "localFileName:"+s.localFile.Name())

	return nil
}
