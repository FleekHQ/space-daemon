package grpc

import (
	"context"
	"encoding/hex"

	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/core/util/address"
	"github.com/FleekHQ/space-daemon/grpc/pb"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/opentracing/opentracing-go"
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

func (srv *grpcServer) UnshareFilesViaPublicKey(
	ctx context.Context,
	request *pb.UnshareFilesViaPublicKeyRequest,
) (*pb.UnshareFilesViaPublicKeyResponse, error) {
	var pks []crypto.PubKey

	for _, pk := range request.PublicKeys {
		b, err := hex.DecodeString(pk)
		if err != nil {
			return nil, err
		}
		p, err := crypto.UnmarshalEd25519PublicKey(b)
		if err != nil {
			return nil, err
		}
		pks = append(pks, p)
	}

	var domainPaths []domain.FullPath
	for _, path := range request.Paths {
		cleanedPath := domain.FullPath{
			Bucket: path.Bucket,
			Path:   path.Path,
			DbId:   path.DbId,
		}

		domainPaths = append(domainPaths, cleanedPath)
	}

	err := srv.sv.UnshareFilesViaPublicKey(ctx, domainPaths, pks)

	return &pb.UnshareFilesViaPublicKeyResponse{}, err
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
				Address:   m.Address,
			})
		}

		var backupCount = 0
		if e.BackedUp {
			backupCount = 1
		}

		dirEntry := &pb.SharedListDirectoryEntry{
			DbId:     e.DbID,
			Bucket:   e.Bucket,
			SharedBy: e.SharedBy,
			Entry: &pb.ListDirectoryEntry{
				Path:               e.Path,
				IsDir:              e.IsDir,
				Name:               e.Name,
				SizeInBytes:        e.SizeInBytes,
				Created:            e.Created,
				Updated:            e.Updated,
				FileExtension:      e.FileExtension,
				IpfsHash:           e.IpfsHash,
				Members:            members,
				IsLocallyAvailable: e.LocallyAvailable,
				BackupCount:        int64(backupCount),
			},
			IsPublicLink: e.IsPublicLink,
		}
		dirEntries = append(dirEntries, dirEntry)
	}

	res := &pb.GetSharedWithMeFilesResponse{
		Items:      dirEntries,
		NextOffset: offset,
	}

	return res, nil
}

func (srv *grpcServer) GetSharedByMeFiles(ctx context.Context, request *pb.GetSharedByMeFilesRequest) (*pb.GetSharedByMeFilesResponse, error) {
	entries, offset, err := srv.sv.GetSharedByMeFiles(ctx, request.Seek, int(request.Limit))
	if err != nil {
		return nil, err
	}

	dirEntries := make([]*pb.SharedListDirectoryEntry, 0)

	for _, e := range entries {
		members := make([]*pb.FileMember, 0)

		for _, m := range e.Members {
			members = append(members, &pb.FileMember{
				PublicKey: m.PublicKey,
				Address:   m.Address,
			})
		}

		var backupCount = 0
		if e.BackedUp {
			backupCount = 1
		}

		dirEntry := &pb.SharedListDirectoryEntry{
			DbId:   e.DbID,
			Bucket: e.Bucket,
			Entry: &pb.ListDirectoryEntry{
				Path:               e.Path,
				IsDir:              e.IsDir,
				Name:               e.Name,
				SizeInBytes:        e.SizeInBytes,
				Created:            e.Created,
				Updated:            e.Updated,
				FileExtension:      e.FileExtension,
				IpfsHash:           e.IpfsHash,
				Members:            members,
				IsLocallyAvailable: e.LocallyAvailable,
				BackupCount:        int64(backupCount),
			},
			IsPublicLink: e.IsPublicLink,
		}
		dirEntries = append(dirEntries, dirEntry)
	}

	res := &pb.GetSharedByMeFilesResponse{
		Items:      dirEntries,
		NextOffset: offset,
	}

	return res, nil
}

func (srv *grpcServer) GeneratePublicFileLink(ctx context.Context, request *pb.GeneratePublicFileLinkRequest) (*pb.GeneratePublicFileLinkResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GeneratePublicFileLink")
	defer span.Finish()

	res, err := srv.sv.GenerateFilesSharingLink(ctx, request.Password, request.ItemPaths, request.Bucket, request.DbId)
	if err != nil {
		return nil, err
	}

	return &pb.GeneratePublicFileLinkResponse{
		Link:    res.SpaceDownloadLink,
		FileCid: res.SharedFileCid,
	}, nil
}

func (srv *grpcServer) OpenPublicFile(ctx context.Context, request *pb.OpenPublicFileRequest) (*pb.OpenPublicFileResponse, error) {
	res, err := srv.sv.OpenSharedFile(ctx, request.FileCid, request.Password, request.Filename)
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
