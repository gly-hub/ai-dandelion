package dao

import (
	"context"

	"github.com/team-dandelion/ai-dandelion/ai-agent/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SessionReference struct {
	db *gorm.DB
}

func NewSessionReference(db *gorm.DB) *SessionReference {
	return &SessionReference{db: db}
}

func (s *SessionReference) UpsertMany(ctx context.Context, refs []model.SessionReference) error {
	if len(refs) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "session_id"},
				{Name: "ref_type"},
				{Name: "ref_id"},
			},
			DoUpdates: clause.AssignmentColumns([]string{"name", "updated_at"}),
		}).
		Create(&refs).
		Error
}

func (s *SessionReference) ListBySession(ctx context.Context, sessionID string) ([]model.SessionReference, error) {
	var refs []model.SessionReference
	err := s.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&refs).
		Error
	if err != nil {
		return nil, err
	}
	return refs, nil
}
