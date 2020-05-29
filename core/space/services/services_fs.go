package services

import (
	"context"
	"github.com/FleekHQ/space-poc/core/space/domain"
)
// TODO: implement this
func (s *Space) ListDir(ctx context.Context) ([]domain.DirEntry, error) {
	panic("implement me")
}

// TODO: implement this
func (s *Space) GetPathInfo(ctx context.Context, path string) (domain.PathInfo, error) {
	res := domain.PathInfo{
		Path:     "test.txt",
		IpfsHash: "testhash",
		IsDir:    false,
	}

	return res, nil
}



