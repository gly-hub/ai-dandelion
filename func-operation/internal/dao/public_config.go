package dao

import (
	"context"
	"sort"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PublicConfig struct {
	db *gorm.DB
}

func NewPublicConfig(db *gorm.DB) *PublicConfig {
	return &PublicConfig{db: db}
}

func (d *PublicConfig) List(ctx context.Context) ([]model.PublicConfig, error) {
	var items []model.PublicConfig
	err := d.db.WithContext(ctx).Order("config_key ASC").Find(&items).Error
	return items, err
}

func (d *PublicConfig) Get(ctx context.Context, configKey string) (*model.PublicConfig, error) {
	var item model.PublicConfig
	if err := d.db.WithContext(ctx).Where("config_key = ?", configKey).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *PublicConfig) Create(ctx context.Context, item *model.PublicConfig, version *model.PublicConfigVersion) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(item).Error; err != nil {
			return err
		}
		return tx.Create(version).Error
	})
}

// UpdateValue serializes version allocation for one config key and records a
// full immutable snapshot along with the current value.
func (d *PublicConfig) UpdateValue(ctx context.Context, configKey, name, description, valueJSON, operatorID, source string, now int64) (*model.PublicConfig, error) {
	var updated model.PublicConfig
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current model.PublicConfig
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("config_key = ?", configKey).First(&current).Error; err != nil {
			return err
		}
		current.Name = name
		current.Description = description
		current.ValueJSON = valueJSON
		current.Version++
		current.UpdatedBy = operatorID
		current.UpdatedAt = now
		if err := tx.Model(&model.PublicConfig{}).Where("config_key = ?", configKey).Updates(map[string]any{
			"name":        current.Name,
			"description": current.Description,
			"value_json":  current.ValueJSON,
			"version":     current.Version,
			"updated_by":  current.UpdatedBy,
			"updated_at":  current.UpdatedAt,
		}).Error; err != nil {
			return err
		}
		version := &model.PublicConfigVersion{
			UUID:       uuid.NewString(),
			ConfigID:   current.UUID,
			ConfigKey:  current.ConfigKey,
			Version:    current.Version,
			ValueJSON:  current.ValueJSON,
			OperatorID: operatorID,
			Source:     source,
			CreatedAt:  now,
		}
		if err := tx.Create(version).Error; err != nil {
			return err
		}
		updated = current
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func (d *PublicConfig) GetImportKey(ctx context.Context) (*model.PublicConfigImportKey, error) {
	var item model.PublicConfigImportKey
	if err := d.db.WithContext(ctx).Where("key_name = ?", "public-config-import").First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *PublicConfig) UpsertImportKey(ctx context.Context, keyHash, operatorID string, now int64) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current model.PublicConfigImportKey
		err := tx.Where("key_name = ?", "public-config-import").First(&current).Error
		if err == gorm.ErrRecordNotFound {
			return tx.Create(&model.PublicConfigImportKey{KeyName: "public-config-import", KeyHash: keyHash, UpdatedBy: operatorID, CreatedAt: now, UpdatedAt: now}).Error
		}
		if err != nil {
			return err
		}
		return tx.Model(&model.PublicConfigImportKey{}).Where("key_name = ?", current.KeyName).Updates(map[string]any{"key_hash": keyHash, "updated_by": operatorID, "updated_at": now}).Error
	})
}

func (d *PublicConfig) ImportValues(ctx context.Context, values map[string]string, operatorID, source string, now int64) ([]model.PublicConfig, error) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	updated := make([]model.PublicConfig, 0, len(keys))
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, key := range keys {
			var current model.PublicConfig
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("config_key = ?", key).First(&current).Error
			if err == gorm.ErrRecordNotFound {
				current = model.PublicConfig{UUID: uuid.NewString(), ConfigKey: key, Name: key, ValueJSON: values[key], Version: 1, CreatedBy: operatorID, UpdatedBy: operatorID, CreatedAt: now, UpdatedAt: now}
				if err := tx.Create(&current).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			} else {
				current.ValueJSON = values[key]
				current.Version++
				current.UpdatedBy = operatorID
				current.UpdatedAt = now
				if err := tx.Model(&model.PublicConfig{}).Where("config_key = ?", current.ConfigKey).Updates(map[string]any{"value_json": current.ValueJSON, "version": current.Version, "updated_by": current.UpdatedBy, "updated_at": current.UpdatedAt}).Error; err != nil {
					return err
				}
			}
			version := &model.PublicConfigVersion{UUID: uuid.NewString(), ConfigID: current.UUID, ConfigKey: current.ConfigKey, Version: current.Version, ValueJSON: current.ValueJSON, OperatorID: operatorID, Source: source, CreatedAt: now}
			if err := tx.Create(version).Error; err != nil {
				return err
			}
			updated = append(updated, current)
		}
		return nil
	})
	return updated, err
}

func (d *PublicConfig) ListVersions(ctx context.Context, configKey string) ([]model.PublicConfigVersion, error) {
	var versions []model.PublicConfigVersion
	err := d.db.WithContext(ctx).Where("config_key = ?", configKey).Order("version DESC").Find(&versions).Error
	return versions, err
}

func (d *PublicConfig) GetVersion(ctx context.Context, configKey string, version int64) (*model.PublicConfigVersion, error) {
	var item model.PublicConfigVersion
	if err := d.db.WithContext(ctx).Where("config_key = ? AND version = ?", configKey, version).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}
