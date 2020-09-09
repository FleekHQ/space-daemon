package grpc

import (
	"context"
	"encoding/hex"

	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/core/util/address"
	"github.com/FleekHQ/space-daemon/grpc/pb"
	crypto "github.com/libp2p/go-libp2p-crypto"
)

func (srv *grpcServer) ShareFilesViaPublicKey(ctx context.Context, request *pb.ShareFilesViaPublicKeyRequest) (*pb.ShareFilesViaPublicKeyResponse, error) {

	var pks []crypto.PubKey

	for _, pk := range request.PublicKeys {
		b, err := hex.DecodeString(pk)
		if err != nil {
			return nil, err
		}
		p, err := crypto.UnmarshalEd25519PublicKey([]byte(b))
		if err != nil {
			return nil, err
		}
		pks = append(pks, p)
	}

	var cleanedPaths []domain.FullPath
	for _, path := range request.Paths {
		cleanedPath := &domain.FullPath{
			Bucket: path.Bucket,
			Path:   path.Path,
			DbId:   path.DbId,
		}

		cleanedPaths = append(cleanedPaths, *cleanedPath)
	}

	// fail before since actual sharing is irreversible
	err := srv.sv.AddRecentlySharedPublicKeys(ctx, pks)
	if err != nil {
		return nil, err
	}

	err = srv.sv.ShareFilesViaPublicKey(ctx, cleanedPaths, pks)
	if err != nil {
		return nil, err
	}

	return &pb.ShareFilesViaPublicKeyResponse{}, nil
}

func (srv *grpcServer) GetSharedWithMeFiles(ctx context.Context, request *pb.GetSharedWithMeFilesRequest) (*pb.GetSharedWithMeFilesResponse, error) {
	entries, offset, err := srv.sv.GetSharedWithMeFiles(ctx, request.Seek, int(request.Limit))
	if err != nil {
		return nil, err
	}

	dirEntries := make([]*pb.SharedListDirectoryEntry, 0)

	for _, e := range entries {
		members := make([]*pb.FileMember, 0)

		for _, m := range e.Members {
			members = append(members, &pb.FileMember{
				PublicKey: m.PublicKey,
			})
		}

		dirEntry := &pb.SharedListDirectoryEntry{
			DbId:   e.DbID,
			Bucket: e.Bucket,
			Entry: &pb.ListDirectoryEntry{
				Path:          e.Path,
				IsDir:         e.IsDir,
				Name:          e.Name,
				SizeInBytes:   e.SizeInBytes,
				Created:       e.Created,
				Updated:       e.Updated,
				FileExtension: e.FileExtension,
				IpfsHash:      e.IpfsHash,
				Members:       members,
			},
		}
		dirEntries = append(dirEntries, dirEntry)
	}

	res := &pb.GetSharedWithMeFilesResponse{
		Items:      dirEntries,
		NextOffset: offset,
	}

	return res, nil
}

func (srv *grpcServer) GeneratePublicFileLink(ctx context.Context, request *pb.GeneratePublicFileLinkRequest) (*pb.GeneratePublicFileLinkResponse, error) {
	res, err := srv.sv.GenerateFilesSharingLink(ctx, request.Password, request.ItemPaths, request.Bucket)
	if err != nil {
		return nil, err
	}

	return &pb.GeneratePublicFileLinkResponse{
		Link:    res.SpaceDownloadLink,
		FileCid: res.SharedFileCid,
	}, nil
}

func (srv *grpcServer) OpenPublicFile(ctx context.Context, request *pb.OpenPublicFileRequest) (*pb.OpenPublicFileResponse, error) {
	res, err := srv.sv.OpenSharedFile(ctx, request.FileCid, request.FileKey, request.Filename)
	if err != nil {
		return nil, err
	}

	return &pb.OpenPublicFileResponse{
		Location: res.Location,
	}, nil
}

func (srv *grpcServer) GetRecentlySharedWith(ctx context.Context, request *pb.GetRecentlySharedWithRequest) (*pb.GetRecentlySharedWithResponse, error) {
	fileMembers := make([]*pb.FileMember, 0)

	pks, err := srv.sv.RecentlySharedPublicKeys(ctx)
	if err != nil {
		return nil, err
	}

	for _, pk := range pks {
		pubBytes, err := pk.Raw()
		if err != nil {
			return nil, err
		}

		fileMember := &pb.FileMember{
			PublicKey: hex.EncodeToString(pubBytes),
			Address:   address.DeriveAddress(pk),
		}

		fileMembers = append(fileMembers, fileMember)
	}

	res := &pb.GetRecentlySharedWithResponse{
		Members: fileMembers,
	}

	return res, nil
}
