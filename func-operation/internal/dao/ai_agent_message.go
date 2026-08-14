package dao

import (
	"context"

	"gorm.io/gorm"
)

type AiAgentMessage struct {
	ID        string `gorm:"column:id"`
	SessionID string `gorm:"column:session_id"`
	Role      string `gorm:"column:role"`
	Content   string `gorm:"column:content"`
	CreatedAt int64  `gorm:"column:created_at"`
}

func (AiAgentMessage) TableName() string {
	return "messages"
}

type AiAgentMessageStore struct {
	db *gorm.DB
}

func NewAiAgentMessageStore(db *gorm.DB) *AiAgentMessageStore {
	return &AiAgentMessageStore{db: db}
}

func (s *AiAgentMessageStore) ListRecentBySession(ctx context.Context, sessionID string, limit int) ([]AiAgentMessage, error) {
	if limit <= 0 {
		limit = 6
	}
	var messages []AiAgentMessage
	err := s.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Order("id DESC").
		Limit(limit).
		Find(&messages).Error
	return messages, err
}
