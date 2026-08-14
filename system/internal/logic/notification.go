package logic

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	systemproto "github.com/team-dandelion/ai-dandelion/proto/system"
	"github.com/team-dandelion/ai-dandelion/system/internal/dao"
	"github.com/team-dandelion/ai-dandelion/system/internal/model"
	"github.com/team-dandelion/ai-dandelion/toolbox/authctx"
	"github.com/team-dandelion/ai-dandelion/toolbox/eventbus"
)

const NotificationEventType = "system.notification"

type NotificationLogic struct {
	notificationDao *dao.Notification
	userDao         *dao.User
	bus             eventbus.Bus
}

func NewNotificationLogic(notificationDao *dao.Notification, userDao *dao.User, bus eventbus.Bus) *NotificationLogic {
	return &NotificationLogic{notificationDao: notificationDao, userDao: userDao, bus: bus}
}

func (l *NotificationLogic) List(ctx context.Context, req *systemproto.ListNotificationsReq) ([]*systemproto.Notification, int64, int64, error) {
	userID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return nil, 0, 0, err
	}
	page, size := int(req.GetPage()), int(req.GetPageSize())
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	items, total, err := l.notificationDao.List(ctx, dao.NotificationListFilter{UserID: userID, Page: page, PageSize: size, UnreadOnly: req.GetUnreadOnly()})
	if err != nil {
		return nil, 0, 0, err
	}
	unread, err := l.notificationDao.UnreadCount(ctx, userID)
	if err != nil {
		return nil, 0, 0, err
	}
	out := make([]*systemproto.Notification, 0, len(items))
	for i := range items {
		out = append(out, notificationToProto(&items[i]))
	}
	return out, total, unread, nil
}

func (l *NotificationLogic) Send(ctx context.Context, req *systemproto.SendNotificationReq) (*systemproto.Notification, error) {
	if l.bus == nil {
		return nil, errors.New("notification realtime event bus is unavailable")
	}
	title, content := strings.TrimSpace(req.GetTitle()), strings.TrimSpace(req.GetContent())
	if title == "" || content == "" {
		return nil, errors.New("title and content are required")
	}
	display := strings.TrimSpace(req.GetDisplayType())
	if display != model.NotificationDisplayModal && display != model.NotificationDisplayToast {
		return nil, errors.New("displayType must be modal or toast")
	}
	level := strings.TrimSpace(req.GetLevel())
	if level == "" {
		level = model.NotificationLevelInfo
	}
	validLevels := map[string]bool{model.NotificationLevelInfo: true, model.NotificationLevelSuccess: true, model.NotificationLevelWarning: true, model.NotificationLevelError: true}
	if !validLevels[level] {
		return nil, errors.New("invalid notification level")
	}
	users := req.GetUserIds()
	if len(users) == 0 {
		listed, err := l.userDao.List(ctx)
		if err != nil {
			return nil, err
		}
		for i := range listed {
			if listed[i].Status == model.UserStatusEnabled {
				users = append(users, listed[i].ID)
			}
		}
	}
	if len(users) == 0 {
		return nil, errors.New("no target users")
	}
	var first *model.Notification
	for _, userID := range users {
		userID = strings.TrimSpace(userID)
		if userID == "" {
			continue
		}
		user, err := l.userDao.Get(ctx, userID)
		if err != nil {
			continue
		}
		item := &model.Notification{ID: uuid.NewString(), Title: title, Content: content, DisplayType: display, Level: level, TargetUserID: user.ID, TargetUserName: user.Username, CreatedAt: nowUnixMicro()}
		if err := l.notificationDao.Create(ctx, item); err != nil {
			return nil, err
		}
		if first == nil {
			first = item
		}
		event, eventErr := eventbus.NewEvent(NotificationEventType, "realtime.events", user.ID, "system", notificationEventPayload(item))
		if eventErr != nil {
			return nil, eventErr
		}
		event.Headers = map[string]string{"userId": user.ID}
		if err := l.bus.Publish(ctx, event); err != nil {
			return nil, err
		}
	}
	if first == nil {
		return nil, errors.New("no target users")
	}
	return notificationToProto(first), nil
}

func (l *NotificationLogic) MarkRead(ctx context.Context, req *systemproto.ReadNotificationReq) error {
	userID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return err
	}
	return l.notificationDao.MarkRead(ctx, userID, strings.TrimSpace(req.GetId()))
}

type notificationEvent struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	DisplayType string `json:"displayType"`
	Level       string `json:"level"`
	CreatedAt   int64  `json:"createdAt"`
}

func notificationEventPayload(item *model.Notification) notificationEvent {
	return notificationEvent{ID: item.ID, Title: item.Title, Content: item.Content, DisplayType: item.DisplayType, Level: item.Level, CreatedAt: item.CreatedAt}
}
func notificationToProto(item *model.Notification) *systemproto.Notification {
	return &systemproto.Notification{Id: item.ID, Title: item.Title, Content: item.Content, DisplayType: item.DisplayType, Level: item.Level, TargetUserId: item.TargetUserID, TargetUserName: item.TargetUserName, Read: item.Read, CreatedAt: item.CreatedAt}
}
