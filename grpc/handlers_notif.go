package grpc

import (
	"context"
	"errors"

	"github.com/FleekHQ/space-daemon/core/events"
	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/grpc/pb"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/golang/protobuf/ptypes/empty"
)

func mapToPbNotification(n domain.Notification) (*pb.Notification, error) {
	// maybe there is a cooler way to do this (e.g., with reflection)
	switch n.NotificationType {
	case domain.INVITATION:
		inv := n.InvitationValue
		pbinv := &pb.Invitation{
			InvitationID:     n.ID,
			InviterPublicKey: inv.InviterPublicKey,
			// TODO: Status: come form shared with me thread,
			ItemPaths: inv.ItemPaths,
		}
		ro := &pb.Notification_InvitationValue{
			InvitationValue: pbinv,
		}
		parsedNotif := &pb.Notification{
			ID:            n.ID,
			Body:          n.Body,
			ReadAt:        n.ReadAt,
			CreatedAt:     n.CreatedAt,
			RelatedObject: ro,
			Type:          pb.NotificationType(n.NotificationType),
		}
		return parsedNotif, nil
	case domain.USAGEALERT:
		ua := n.UsageAlertValue
		pbua := &pb.UsageAlert{
			Used:    ua.Used,
			Limit:   ua.Limit,
			Message: ua.Message,
		}
		ro := &pb.Notification_UsageAlert{
			UsageAlert: pbua,
		}
		parsedNotif := &pb.Notification{
			ID:            n.ID,
			Body:          n.Body,
			ReadAt:        n.ReadAt,
			CreatedAt:     n.CreatedAt,
			RelatedObject: ro,
			Type:          pb.NotificationType(n.NotificationType),
		}
		return parsedNotif, nil
	default:
		return nil, errors.New("Unsupported message type")
	}
}

func (srv *grpcServer) ClearNewNotifications(ctx context.Context, request *pb.ClearNewNotificationsRequest) (*pb.ClearNewNotificationsResponse, error) {
	return nil, errNotImplemented
}

func (srv *grpcServer) GetNotifications(ctx context.Context, request *pb.GetNotificationsRequest) (*pb.GetNotificationsResponse, error) {
	// textile expects int instead of int64 for limit field
	n, err := srv.sv.GetNotifications(ctx, request.Seek, int(request.Limit))
	if err != nil {
		return nil, err
	}

	parsedNotifs := []*pb.Notification{}

	for _, notif := range n {
		parsedNotif, err := mapToPbNotification(*notif)
		if err != nil {
			return nil, err
		}
		parsedNotifs = append(parsedNotifs, parsedNotif)
	}

	no := parsedNotifs[len(parsedNotifs)].ID

	return &pb.GetNotificationsResponse{
		Notifications: parsedNotifs,
		NextOffset:    no,
	}, nil
}

func (srv *grpcServer) ReadNotification(ctx context.Context, request *pb.ReadNotificationRequest) (*pb.ReadNotificationResponse, error) {
	return nil, errNotImplemented
}

func (srv *grpcServer) AcceptFilesInvitation(ctx context.Context, request *pb.AcceptFilesInvitationRequest) (*pb.AcceptFilesInvitationResponse, error) {
	return nil, errNotImplemented
}

func (srv *grpcServer) RejectFilesInvitation(ctx context.Context, request *pb.RejectFilesInvitationRequest) (*pb.RejectFilesInvitationResponse, error) {
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
