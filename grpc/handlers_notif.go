package grpc

import (
	"context"

	"github.com/FleekHQ/space-daemon/core/events"
	"github.com/FleekHQ/space-daemon/grpc/pb"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/golang/protobuf/ptypes/empty"
)

func (srv *grpcServer) GetNotifications(ctx context.Context, request *pb.GetNotificationsRequest) (*pb.GetNotificationsResponse, error) {
	return nil, errNotImplemented
}

func (srv *grpcServer) ReadNotification(ctx context.Context, request *pb.ReadNotificationRequest) (*pb.ReadNotificationResponse, error) {
	return nil, errNotImplemented
}

func (srv *grpcServer) AcceptBucketInvitation(ctx context.Context, request *pb.AcceptBucketInvitationRequest) (*pb.AcceptBucketInvitationResponse, error) {
	return nil, errNotImplemented
}

func (srv *grpcServer) RejectBucketInvitation(ctx context.Context, request *pb.RejectBucketInvitationRequest) (*pb.RejectBucketInvitationResponse, error) {
	return nil, errNotImplemented
}

func (srv *grpcServer) NotificationSubscribe(empty *empty.Empty, stream pb.SpaceApi_NotificationSubscribeServer) error {
	srv.registerNotificationStream(stream)
	// waits until request is done
	select {
	case <-stream.Context().Done():
		break
	}
	// clean up stream
	srv.registerNotificationStream(nil)
	log.Info("closing stream")
	return nil
}

func (srv *grpcServer) registerNotificationStream(stream pb.SpaceApi_NotificationSubscribeServer) {
	srv.notificationEventStream = stream
}

func (srv *grpcServer) sendNotificationEvent(event *pb.NotificationEventResponse) {
	if srv.notificationEventStream != nil {
		log.Info("sending events to client")
		srv.notificationEventStream.Send(event)
	}
}

func (srv *grpcServer) SendInvitationEvent(event events.NotificationEvent) {
	pe := &pb.NotificationEventResponse{}

	srv.sendNotificationEvent(pe)
}
