package service

import (
	"context"
	systemproto "github.com/team-dandelion/ai-dandelion/proto/system"
	"github.com/team-dandelion/quickgo/grpcep"
)

func (s *SystemService) ListNotifications(ctx context.Context, req *systemproto.ListNotificationsReq) (out *systemproto.ListNotificationsResp, err error) {
	grpcep.InitResponse(&out)
	out.Notifications, out.Total, out.UnreadCount, err = s.notificationLogic.List(ctx, req)
	return
}
func (s *SystemService) SendNotification(ctx context.Context, req *systemproto.SendNotificationReq) (out *systemproto.SendNotificationResp, err error) {
	grpcep.InitResponse(&out)
	out.Notification, err = s.notificationLogic.Send(ctx, req)
	return
}
func (s *SystemService) ReadNotification(ctx context.Context, req *systemproto.ReadNotificationReq) (out *systemproto.ReadNotificationResp, err error) {
	grpcep.InitResponse(&out)
	err = s.notificationLogic.MarkRead(ctx, req)
	return
}
