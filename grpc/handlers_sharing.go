package grpc

import (
	"context"

	"github.com/FleekHQ/space-daemon/core/events"
	"github.com/FleekHQ/space-daemon/grpc/pb"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/golang/protobuf/ptypes/empty"
)

func (srv *grpcServer) ShareBucketViaPublicKey(ctx context.Context, request *pb.ShareBucketViaPublicKeyRequest) (*pb.ShareBucketViaPublicKeyResponse, error) {
	return nil, errNotImplemented
}

func (srv *grpcServer) GeneratePublicFileLink(ctx context.Context, request *pb.GeneratePublicFileLinkRequest) (*pb.GeneratePublicFileLinkResponse, error) {
	// TODO: Generalize for multiple file upload
	res, err := srv.sv.GenerateFileSharingLink(ctx, request.ItemPaths[0], request.Bucket)
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

func (srv *grpcServer) GetPendingBucketInvitations(ctx context.Context, request *pb.GetPendingBucketInvitationsRequest) (*pb.GetPendingBucketInvitationsResponse, error) {
	return nil, errNotImplemented
}

func (srv *grpcServer) AcceptBucketInvitation(ctx context.Context, request *pb.AcceptBucketInvitationRequest) (*pb.AcceptBucketInvitationResponse, error) {
	return nil, errNotImplemented
}

func (srv *grpcServer) RejectBucketInvitation(ctx context.Context, request *pb.RejectBucketInvitationRequest) (*pb.RejectBucketInvitationResponse, error) {
	return nil, errNotImplemented
}

func (srv *grpcServer) InvitationSubscribe(empty *empty.Empty, stream pb.SpaceApi_InvitationSubscribeServer) error {
	srv.registerInvitationStream(stream)
	// waits until request is done
	select {
	case <-stream.Context().Done():
		break
	}
	// clean up stream
	srv.registerInvitationStream(nil)
	log.Info("closing stream")
	return nil
}

func (srv *grpcServer) registerInvitationStream(stream pb.SpaceApi_InvitationSubscribeServer) {
	srv.invitationEventStream = stream
}

func (srv *grpcServer) sendInvitationEvent(event *pb.InvitationEventResponse) {
	if srv.invitationEventStream != nil {
		log.Info("sending events to client")
		srv.invitationEventStream.Send(event)
	}
}

func (srv *grpcServer) SendInvitationEvent(event events.InvitationEvent) {
	pe := &pb.InvitationEventResponse{}

	srv.sendInvitationEvent(pe)
}
