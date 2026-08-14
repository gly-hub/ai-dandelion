package dao

import (
	"context"
	"errors"
	"strings"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FunctionRelease struct{ db *gorm.DB }

func NewFunctionRelease(db *gorm.DB) *FunctionRelease { return &FunctionRelease{db: db} }

func (d *FunctionRelease) CreateStaged(ctx context.Context, release *model.FunctionRelease) error {
	if release.UUID == "" {
		release.UUID = uuid.NewString()
	}
	return d.db.WithContext(ctx).Create(release).Error
}

func (d *FunctionRelease) Latest(ctx context.Context, functionID string) (*model.FunctionRelease, error) {
	var release model.FunctionRelease
	err := d.db.WithContext(ctx).Where("function_id = ?", functionID).Order("version DESC, id DESC").First(&release).Error
	if err != nil {
		return nil, err
	}
	return &release, nil
}

func (d *FunctionRelease) ActiveForApp(ctx context.Context, appID string) (*model.FunctionRelease, error) {
	var release model.FunctionRelease
	err := d.db.WithContext(ctx).Where("app_id = ? AND status = ?", appID, model.FunctionReleaseStatusPublished).Order("version DESC, id DESC").First(&release).Error
	if err != nil {
		return nil, err
	}
	return &release, nil
}

func (d *FunctionRelease) ListPublished(ctx context.Context) ([]model.FunctionRelease, error) {
	var releases []model.FunctionRelease
	err := d.db.WithContext(ctx).Where("status = ?", model.FunctionReleaseStatusPublished).Find(&releases).Error
	return releases, err
}

func (d *FunctionRelease) RevokeByFunctionID(ctx context.Context, functionID string, now int64) error {
	return d.db.WithContext(ctx).Model(&model.FunctionRelease{}).
		Where("function_id = ? AND status = ?", functionID, model.FunctionReleaseStatusPublished).
		Updates(map[string]any{"status": model.FunctionReleaseStatusRevoked, "updated_at": now}).Error
}

// ListAll is used by artifact maintenance to retain every release referenced
// by the database, including staged and revoked versions.
func (d *FunctionRelease) ListAll(ctx context.Context) ([]model.FunctionRelease, error) {
	var releases []model.FunctionRelease
	err := d.db.WithContext(ctx).Order("id ASC").Find(&releases).Error
	return releases, err
}

func (d *FunctionRelease) Publish(ctx context.Context, releaseID string, publishedAt int64) error {
	return d.publish(ctx, releaseID, publishedAt, nil)
}

// PublishWithOutbox makes the active release and its cross-service event
// visible together, so a committed release always has a retryable delivery.
func (d *FunctionRelease) PublishWithOutbox(ctx context.Context, releaseID string, publishedAt int64, event *model.FunctionOutboxEvent) error {
	if event == nil {
		return errors.New("outbox event is required")
	}
	return d.publish(ctx, releaseID, publishedAt, event)
}

func (d *FunctionRelease) publish(ctx context.Context, releaseID string, publishedAt int64, event *model.FunctionOutboxEvent) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var release model.FunctionRelease
		if err := tx.Where("uuid = ?", releaseID).First(&release).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.FunctionRelease{}).Where("function_id = ? AND status = ?", release.FunctionID, model.FunctionReleaseStatusPublished).Updates(map[string]any{"status": model.FunctionReleaseStatusRevoked, "updated_at": publishedAt}).Error; err != nil {
			return err
		}
		result := tx.Model(&model.FunctionRelease{}).Where("uuid = ?", releaseID).Updates(map[string]any{"status": model.FunctionReleaseStatusPublished, "published_at": publishedAt, "updated_at": publishedAt})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("release not found")
		}
		if err := tx.Model(&model.Function{}).Where("uuid = ?", release.FunctionID).Updates(map[string]any{"active_release_id": releaseID, "updated_at": publishedAt}).Error; err != nil {
			return err
		}
		if event == nil {
			return nil
		}
		if event.UUID == "" {
			event.UUID = uuid.NewString()
		}
		return tx.Create(event).Error
	})
}

// BackfillPublished establishes an immutable trust record for a legacy
// published function. It is idempotent and only used during startup upgrade.
func (d *FunctionRelease) BackfillPublished(ctx context.Context, functionID, appID, artifactSHA256, manifestJSON string, publishedAt int64) (*model.FunctionRelease, error) {
	var out model.FunctionRelease
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var function model.Function
		if err := tx.Where("uuid = ?", functionID).First(&function).Error; err != nil {
			return err
		}
		if function.ActiveReleaseID != "" {
			return nil
		}
		var existing model.FunctionRelease
		err := tx.Where("function_id = ? AND status = ?", functionID, model.FunctionReleaseStatusPublished).
			Order("version DESC, id DESC").
			First(&existing).Error
		if err == nil {
			out = existing
			return tx.Model(&model.Function{}).Where("id = ?", function.ID).Updates(map[string]any{
				"active_release_id": existing.UUID,
				"updated_at":        publishedAt,
			}).Error
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		out = model.FunctionRelease{
			UUID:           uuid.NewString(),
			FunctionID:     functionID,
			AppID:          appID,
			Version:        1,
			ArtifactSHA256: artifactSHA256,
			ManifestJSON:   manifestJSON,
			Status:         model.FunctionReleaseStatusPublished,
			CreatedAt:      publishedAt,
			PublishedAt:    publishedAt,
			UpdatedAt:      publishedAt,
		}
		if err := tx.Create(&out).Error; err != nil {
			return err
		}
		return tx.Model(&model.Function{}).Where("id = ?", function.ID).Updates(map[string]any{
			"active_release_id": out.UUID,
			"updated_at":        publishedAt,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type FunctionOutbox struct{ db *gorm.DB }

func NewFunctionOutbox(db *gorm.DB) *FunctionOutbox { return &FunctionOutbox{db: db} }
func (d *FunctionOutbox) Create(ctx context.Context, event *model.FunctionOutboxEvent) error {
	if event.UUID == "" {
		event.UUID = uuid.NewString()
	}
	return d.db.WithContext(ctx).Create(event).Error
}

func (d *FunctionOutbox) GetByReleaseID(ctx context.Context, eventType, releaseID string) (*model.FunctionOutboxEvent, error) {
	var event model.FunctionOutboxEvent
	err := d.db.WithContext(ctx).Where("event_type = ? AND release_id = ?", eventType, releaseID).Order("id DESC").First(&event).Error
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (d *FunctionOutbox) List(ctx context.Context, limit int) ([]model.FunctionOutboxEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var events []model.FunctionOutboxEvent
	err := d.db.WithContext(ctx).Order("id DESC").Limit(limit).Find(&events).Error
	return events, err
}

func (d *FunctionOutbox) Requeue(ctx context.Context, now int64) error {
	return d.db.WithContext(ctx).Model(&model.FunctionOutboxEvent{}).Where("status = ?", "failed").Updates(map[string]any{"status": "pending", "next_attempt_at": now, "updated_at": now}).Error
}

func (d *FunctionOutbox) ClaimReady(ctx context.Context, now int64, limit int) ([]model.FunctionOutboxEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var events []model.FunctionOutboxEvent
	err := d.db.WithContext(ctx).
		Where("(status IN ? AND next_attempt_at <= ?) OR (status = ? AND updated_at <= ?)", []string{"pending", "failed"}, now, "processing", now-60*1000000).
		Order("id ASC").
		Limit(limit).
		Find(&events).Error
	if err != nil {
		return nil, err
	}
	claimed := make([]model.FunctionOutboxEvent, 0, len(events))
	for i := range events {
		event := events[i]
		result := d.db.WithContext(ctx).Model(&model.FunctionOutboxEvent{}).
			Where("id = ? AND ((status IN ?) OR (status = ? AND updated_at <= ?))", event.ID, []string{"pending", "failed"}, "processing", now-60*1000000).
			Updates(map[string]any{"status": "processing", "attempts": event.Attempts + 1, "updated_at": now})
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected == 1 {
			event.Attempts++
			event.Status = "processing"
			claimed = append(claimed, event)
		}
	}
	return claimed, nil
}

func (d *FunctionOutbox) MarkDone(ctx context.Context, id uint64, now int64) error {
	return d.db.WithContext(ctx).Model(&model.FunctionOutboxEvent{}).Where("id = ?", id).Updates(map[string]any{"status": "done", "last_error": "", "next_attempt_at": 0, "updated_at": now}).Error
}

func (d *FunctionOutbox) MarkDoneByUUID(ctx context.Context, uuid string, now int64) error {
	return d.db.WithContext(ctx).Model(&model.FunctionOutboxEvent{}).Where("uuid = ?", uuid).Updates(map[string]any{"status": "done", "last_error": "", "next_attempt_at": 0, "updated_at": now}).Error
}

func (d *FunctionOutbox) MarkRetry(ctx context.Context, id uint64, nextAttemptAt int64, cause error, now int64) error {
	message := "outbox delivery failed"
	if cause != nil {
		message = strings.TrimSpace(cause.Error())
	}
	if len(message) > 1000 {
		message = message[:1000]
	}
	return d.db.WithContext(ctx).Model(&model.FunctionOutboxEvent{}).Where("id = ?", id).Updates(map[string]any{"status": "failed", "last_error": message, "next_attempt_at": nextAttemptAt, "updated_at": now}).Error
}
