package grpc

import (
	"context"

	"github.com/FleekHQ/space-daemon/core/events"
	"github.com/FleekHQ/space-daemon/core/space/domain"
	"github.com/FleekHQ/space-daemon/grpc/pb"
	"github.com/FleekHQ/space-daemon/log"
	"github.com/golang/protobuf/ptypes/empty"
)

func mapToPbNotification(n domain.Notification) *pb.Notification {
	// maybe there is a cooler way to do this (e.g., with reflection)
	switch n.NotificationType {
	case domain.INVITATION:
		inv := n.InvitationValue
		pbpths := make([]*pb.FullPath, 0)

		for _, pth := range n.InvitationValue.ItemPaths {
			pbpth := &pb.FullPath{
				Bucket: pth.Bucket,
				DbId:   pth.DbId,
				Path:   pth.Path,
			}
			pbpths = append(pbpths, pbpth)
		}

		pbinv := &pb.Invitation{
			InvitationID:     n.ID,
			InviterPublicKey: inv.InviterPublicKey,
			// TODO: Status: come form shared with me thread,
			ItemPaths: pbpths,
		}
		ro := &pb.Notification_InvitationValue{pbinv}
		parsedNotif := &pb.Notification{
			ID:            n.ID,
			Body:          n.Body,
			ReadAt:        n.ReadAt,
			CreatedAt:     n.CreatedAt,
			RelatedObject: ro,
			Type:          pb.NotificationType(n.NotificationType),
		}
		return parsedNotif
	case domain.USAGEALERT:
		ua := n.UsageAlertValue
		pbua := &pb.UsageAlert{
			Used:    ua.Used,
			Limit:   ua.Limit,
			Message: ua.Message,
		}
		ro := &pb.Notification_UsageAlert{pbua}
		parsedNotif := &pb.Notification{
			ID:            n.ID,
			Body:          n.Body,
			ReadAt:        n.ReadAt,
			CreatedAt:     n.CreatedAt,
			RelatedObject: ro,
			Type:          pb.NotificationType(n.NotificationType),
		}
		return parsedNotif
	case domain.INVITATION_REPLY:
		ir := n.InvitationAcceptValue
		pbir := &pb.InvitationAccept{
			InvitationID: ir.InvitationID,
		}
		ro := &pb.Notification_InvitationAccept{pbir}
		parsedNotif := &pb.Notification{
			ID:            n.ID,
			Body:          n.Body,
			ReadAt:        n.ReadAt,
			CreatedAt:     n.CreatedAt,
			RelatedObject: ro,
			Type:          pb.NotificationType(n.NotificationType),
		}
		return parsedNotif
	default:
		parsedNotif := &pb.Notification{
			ID:        n.ID,
			Body:      n.Body,
			ReadAt:    n.ReadAt,
			CreatedAt: n.CreatedAt,
			Type:      pb.NotificationType(n.NotificationType),
		}
		return parsedNotif
	}
}

func (srv *grpcServer) SetNotificationsLastSeenAt(ctx context.Context, request *pb.SetNotificationsLastSeenAtRequest) (*pb.SetNotificationsLastSeenAtResponse, error) {
	err := srv.sv.SetNotificationsLastSeenAt(request.Timestamp)
	if err != nil {
		return nil, err
	}
	return &pb.SetNotificationsLastSeenAtResponse{}, nil
}

func (srv *grpcServer) GetNotifications(ctx context.Context, request *pb.GetNotificationsRequest) (*pb.GetNotificationsResponse, error) {
	// textile expects int instead of int64 for limit field
	n, err := srv.sv.GetNotifications(ctx, request.Seek, int(request.Limit))
	if err != nil {
		return nil, err
	}

	parsedNotifs := []*pb.Notification{}

	for _, notif := range n {
		parsedNotif := mapToPbNotification(*notif)
		parsedNotifs = append(parsedNotifs, parsedNotif)
	}

	var no string
	if len(parsedNotifs) > 0 {
		no = parsedNotifs[len(parsedNotifs)-1].ID
	}

	ls, err := srv.sv.GetNotificationsLastSeenAt()
	if err != nil {
		// error getting last seen at but we dont want to fail the
		// whole request for that
		ls = 0
	}

	return &pb.GetNotificationsResponse{
		Notifications: parsedNotifs,
		NextOffset:    no,
		LastSeenAt:    ls,
	}, nil
}

func (srv *grpcServer) ReadNotification(ctx context.Context, request *pb.ReadNotificationRequest) (*pb.ReadNotificationResponse, error) {
	return nil, errNotImplemented
}

func (srv *grpcServer) HandleFilesInvitation(
	ctx context.Context,
	request *pb.HandleFilesInvitationRequest,
) (*pb.HandleFilesInvitationResponse, error) {
	err := srv.sv.HandleSharedFilesInvitation(ctx, request.InvitationID, request.Accept)
	if err != nil {
		return nil, err
	}

	return &pb.HandleFilesInvitationResponse{}, nil
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

func (srv *grpcServer) SendNotificationEvent(event events.NotificationEvent) {
	pe := &pb.NotificationEventResponse{}

	srv.sendNotificationEvent(pe)
}
