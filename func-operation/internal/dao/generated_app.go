package dao

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/team-dandelion/ai-dandelion/func-operation/internal/model"
	"gorm.io/gorm"
)

type GeneratedApp struct {
	db *gorm.DB
}

func NewGeneratedApp(db *gorm.DB) *GeneratedApp {
	return &GeneratedApp{db: db}
}

func (g *GeneratedApp) List(ctx context.Context) ([]model.GeneratedApp, error) {
	var apps []model.GeneratedApp
	err := g.db.WithContext(ctx).
		Order("updated_at DESC").
		Order("id ASC").
		Find(&apps).Error
	return apps, err
}

func (g *GeneratedApp) Get(ctx context.Context, uuid string) (*model.GeneratedApp, error) {
	var app model.GeneratedApp
	err := g.db.WithContext(ctx).
		Where("uuid = ?", uuid).
		First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (g *GeneratedApp) Upsert(ctx context.Context, app *model.GeneratedApp) error {
	now := nowUnixMicro()
	if app.CreatedAt == 0 {
		app.CreatedAt = now
	}
	app.UpdatedAt = now
	var existing model.GeneratedApp
	err := g.db.WithContext(ctx).Where("uuid = ?", app.UUID).First(&existing).Error
	if err == nil {
		app.ID = existing.ID
		app.CreatedAt = existing.CreatedAt
		return g.db.WithContext(ctx).Save(app).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return g.db.WithContext(ctx).Create(app).Error
}

func (g *GeneratedApp) ListRecords(ctx context.Context, appUUID string) ([]model.AppRecord, error) {
	var records []model.AppRecord
	err := g.db.WithContext(ctx).
		Where("app_uuid = ?", appUUID).
		Order("created_at ASC").
		Find(&records).Error
	return records, err
}

func (g *GeneratedApp) UpsertRecord(ctx context.Context, record *model.AppRecord) error {
	now := nowUnixMicro()
	if strings.TrimSpace(record.UUID) == "" {
		record.UUID = uuid.NewString()
	}
	if record.CreatedAt == 0 {
		record.CreatedAt = now
	}
	record.UpdatedAt = now
	return g.db.WithContext(ctx).Save(record).Error
}

func (g *GeneratedApp) DeleteRecord(ctx context.Context, appUUID string, recordUUID string) error {
	return g.db.WithContext(ctx).
		Where("app_uuid = ? AND uuid = ?", appUUID, recordUUID).
		Delete(&model.AppRecord{}).Error
}

func sqlColumnType(columnTypes []*sql.ColumnType, index int) *sql.ColumnType {
	if index >= 0 && index < len(columnTypes) && columnTypes[index] != nil {
		return columnTypes[index]
	}
	return nil
}

func normalizeColumnTypeName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "unknown"
	}
	return value
}

func normalizeSQLValue(value any, columnType *sql.ColumnType) any {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []byte:
		text := string(typed)
		if isTextColumn(columnType) {
			return text
		}
		if isIntegerColumn(columnType) {
			if number, err := strconv.ParseInt(text, 10, 64); err == nil {
				return number
			}
			return text
		}
		if isUnsignedIntegerColumn(columnType) {
			if number, err := strconv.ParseUint(text, 10, 64); err == nil {
				return number
			}
			return text
		}
		if isDecimalColumn(columnType) {
			return text
		}
		if isFloatColumn(columnType) {
			if number, err := strconv.ParseFloat(text, 64); err == nil {
				return number
			}
		}
		return text
	default:
		return typed
	}
}

func isTextColumn(columnType *sql.ColumnType) bool {
	name := normalizeColumnTypeName(columnTypeName(columnType))
	return strings.Contains(name, "char") ||
		strings.Contains(name, "text") ||
		strings.Contains(name, "json") ||
		strings.Contains(name, "enum") ||
		strings.Contains(name, "set") ||
		strings.Contains(name, "date") ||
		strings.Contains(name, "time") ||
		strings.Contains(name, "year") ||
		strings.Contains(name, "uuid")
}

func isIntegerColumn(columnType *sql.ColumnType) bool {
	name := normalizeColumnTypeName(columnTypeName(columnType))
	return strings.Contains(name, "int") && !strings.Contains(name, "unsigned")
}

func isUnsignedIntegerColumn(columnType *sql.ColumnType) bool {
	name := normalizeColumnTypeName(columnTypeName(columnType))
	return strings.Contains(name, "int") && strings.Contains(name, "unsigned")
}

func isDecimalColumn(columnType *sql.ColumnType) bool {
	name := normalizeColumnTypeName(columnTypeName(columnType))
	return strings.Contains(name, "decimal") || strings.Contains(name, "numeric")
}

func isFloatColumn(columnType *sql.ColumnType) bool {
	name := normalizeColumnTypeName(columnTypeName(columnType))
	return strings.Contains(name, "float") || strings.Contains(name, "double") || strings.Contains(name, "real")
}

func columnTypeName(columnType *sql.ColumnType) string {
	if columnType == nil {
		return ""
	}
	return columnType.DatabaseTypeName()
}

func nowUnixMicro() int64 {
	return time.Now().UnixMicro()
}
