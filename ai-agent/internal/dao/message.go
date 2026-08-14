package dao

import (
	"context"

	"github.com/gly-hub/ai-dandelion/ai-agent/internal/model"
	"gorm.io/gorm"
)

type Message struct {
	db *gorm.DB
}

type MessagePageOptions struct {
	Limit  int
	Before string
}

type MessagePage struct {
	Items      []model.Message
	HasMore    bool
	NextBefore string
}

func NewMessage(db *gorm.DB) *Message {
	return &Message{db: db}
}

func (m *Message) HasMessages(ctx context.Context, sessionID string) (bool, error) {
	var count int64
	err := m.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("session_id = ?", sessionID).
		Limit(1).
		Count(&count).
		Error
	return count > 0, err
}

func (m *Message) List(ctx context.Context, sessionID string, options MessagePageOptions) (MessagePage, error) {
	limit := normalizeMessagePageLimit(options.Limit)
	query := m.db.WithContext(ctx).
		Where("session_id = ?", sessionID)

	if options.Before != "" {
		cursor, err := m.loadCursor(ctx, sessionID, options.Before)
		if err != nil {
			return MessagePage{}, err
		}
		query = query.Where("(created_at < ? OR (created_at = ? AND id < ?))", cursor.CreatedAt, cursor.CreatedAt, cursor.ID)
	}

	var messages []model.Message
	err := query.
		Order("created_at DESC").
		Order("id DESC").
		Limit(limit + 1).
		Find(&messages).
		Error
	if err != nil {
		return MessagePage{}, err
	}

	page := MessagePage{Items: messages}
	if len(page.Items) > limit {
		page.HasMore = true
		page.Items = page.Items[:limit]
	}
	if len(page.Items) > 0 {
		page.NextBefore = page.Items[len(page.Items)-1].ID
	}
	return page, nil
}

func (m *Message) Add(ctx context.Context, message *model.Message, sessionTitle string) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(message).Error; err != nil {
			return err
		}

		updates := map[string]interface{}{"updated_at": message.CreatedAt}
		if message.Role == model.RoleUser && sessionTitle != "" {
			updates["title"] = gorm.Expr("CASE WHEN title = ? THEN ? ELSE title END", model.DefaultSessionTitle, sessionTitle)
		}
		return tx.Model(&model.Session{}).
			Where("id = ?", message.SessionID).
			Updates(updates).
			Error
	})
}

func (m *Message) loadCursor(ctx context.Context, sessionID string, messageID string) (model.Message, error) {
	var message model.Message
	err := m.db.WithContext(ctx).
		Where("session_id = ? AND id = ?", sessionID, messageID).
		First(&message).
		Error
	return message, err
}

func normalizeMessagePageLimit(limit int) int {
	switch {
	case limit <= 0:
		return 40
	case limit > 200:
		return 200
	default:
		return limit
	}
}
