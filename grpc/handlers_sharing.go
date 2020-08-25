package grpc

import (
	"context"

	"github.com/FleekHQ/space-daemon/grpc/pb"
	crypto "github.com/libp2p/go-libp2p-crypto"
)

func (srv *grpcServer) ShareFilesViaPublicKey(ctx context.Context, request *pb.ShareFilesViaPublicKeyRequest) (*pb.ShareFilesViaPublicKeyResponse, error) {

	var pks []crypto.PubKey

	for _, pk := range request.PublicKeys {
		p, err := crypto.UnmarshalEd25519PublicKey([]byte(pk))
		if err != nil {
			return nil, err
		}
		pks = append(pks, p)
	}

	err := srv.sv.ShareFilesViaPublicKey(ctx, request.Bucket, request.Paths, pks)
	if err != nil {
		return nil, err
	}

	return &pb.ShareFilesViaPublicKeyResponse{}, nil
}

func (srv *grpcServer) GetSharedWithMeFiles(ctx context.Context, request *pb.GetSharedWithMeFilesRequest) (*pb.GetSharedWithMeFilesResponse, error) {
	return nil, errNotImplemented
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
