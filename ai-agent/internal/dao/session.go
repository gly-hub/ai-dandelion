package dao

import (
	"context"

	"github.com/gly-hub/ai-dandelion/ai-agent/internal/model"
	"gorm.io/gorm"
)

type Session struct {
	db *gorm.DB
}

func NewSession(db *gorm.DB) *Session {
	return &Session{db: db}
}

func (s *Session) List(ctx context.Context, sessionType int32) ([]model.Session, error) {
	var sessions []model.Session
	query := s.db.WithContext(ctx)
	if sessionType > 0 {
		query = query.Where("session_type = ?", sessionType)
	}
	err := query.Order("updated_at DESC").
		Find(&sessions).Error
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

func (s *Session) Create(ctx context.Context, session *model.Session) error {
	return s.db.WithContext(ctx).Create(session).Error
}

func (s *Session) UpdateTitle(ctx context.Context, sessionID string, title string, updatedAt int64) error {
	result := s.db.WithContext(ctx).
		Model(&model.Session{}).
		Where("id = ?", sessionID).
		Updates(map[string]interface{}{
			"title":      title,
			"updated_at": updatedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *Session) Get(ctx context.Context, sessionID string) (*model.Session, error) {
	var session model.Session
	err := s.db.WithContext(ctx).
		Where("id = ?", sessionID).
		First(&session).
		Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *Session) Delete(ctx context.Context, sessionID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.
			Where("id = ?", sessionID).
			Delete(&model.Session{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		if err := tx.
			Where("session_id = ?", sessionID).
			Delete(&model.SessionReference{}).
			Error; err != nil {
			return err
		}
		return tx.
			Where("session_id = ?", sessionID).
			Delete(&model.Message{}).
			Error
	})
}

func (s *Session) Exists(ctx context.Context, sessionID string) error {
	var session model.Session
	return s.db.WithContext(ctx).
		Select("id").
		Where("id = ?", sessionID).
		First(&session).
		Error
}

func (s *Session) UpdateAgentSession(ctx context.Context, sessionID string, agentSessionID string, updatedAt int64) error {
	return s.db.WithContext(ctx).
		Model(&model.Session{}).
		Where("id = ?", sessionID).
		Updates(map[string]interface{}{
			"agent_session_id": agentSessionID,
			"updated_at":       updatedAt,
		}).
		Error
}
