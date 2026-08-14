package dao

import (
	"context"
	"github.com/team-dandelion/ai-dandelion/func-operation/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ExternalAPI struct{ db *gorm.DB }

func NewExternalAPI(db *gorm.DB) *ExternalAPI { return &ExternalAPI{db: db} }
func (d *ExternalAPI) ListClients(ctx context.Context) ([]model.ExternalAPIClient, error) {
	var rows []model.ExternalAPIClient
	err := d.db.WithContext(ctx).Where("deleted_at = 0").Order("client_key ASC").Find(&rows).Error
	return rows, err
}
func (d *ExternalAPI) GetClient(ctx context.Context, key string) (*model.ExternalAPIClient, error) {
	var row model.ExternalAPIClient
	if err := d.db.WithContext(ctx).Where("client_key = ? AND deleted_at = 0", key).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}
func (d *ExternalAPI) ListDeletedClients(ctx context.Context) ([]model.ExternalAPIClient, error) {
	var rows []model.ExternalAPIClient
	err := d.db.WithContext(ctx).Where("deleted_at > 0").Order("deleted_at DESC").Find(&rows).Error
	return rows, err
}
func (d *ExternalAPI) SoftDeleteClientAssets(ctx context.Context, clientKey, user string, deletedAt int64) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{"deleted_at": deletedAt, "updated_at": deletedAt, "updated_by": user}
		if err := tx.Model(&model.ExternalAPIClient{}).Where("client_key = ? AND deleted_at = 0", clientKey).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.ExternalAPI{}).Where("client_key = ? AND deleted_at = 0", clientKey).Updates(updates).Error; err != nil {
			return err
		}
		return tx.Model(&model.ExternalAPIGroup{}).Where("client_key = ? AND deleted_at = 0", clientKey).Updates(updates).Error
	})
}
func (d *ExternalAPI) PurgeClientAssets(ctx context.Context, clientKey string) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var client model.ExternalAPIClient
		if err := tx.Where("client_key = ? AND deleted_at > 0", clientKey).First(&client).Error; err != nil {
			return err
		}
		if err := tx.Where("client_key = ?", clientKey).Delete(&model.ExternalAPI{}).Error; err != nil {
			return err
		}
		if err := tx.Where("client_key = ?", clientKey).Delete(&model.ExternalAPIGroup{}).Error; err != nil {
			return err
		}
		return tx.Where("client_key = ? AND deleted_at > 0", clientKey).Delete(&model.ExternalAPIClient{}).Error
	})
}
func (d *ExternalAPI) CreateClient(ctx context.Context, row *model.ExternalAPIClient) error {
	return d.db.WithContext(ctx).Create(row).Error
}
func (d *ExternalAPI) UpdateClient(ctx context.Context, row *model.ExternalAPIClient) error {
	result := d.db.WithContext(ctx).Model(&model.ExternalAPIClient{}).
		Where("client_key = ? AND deleted_at = 0", row.ClientKey).
		Updates(map[string]any{
			"name":                    row.Name,
			"base_url":                row.BaseURL,
			"default_headers_json":    row.DefaultHeadersJSON,
			"description":             row.Description,
			"status":                  row.Status,
			"updated_by":              row.UpdatedBy,
			"updated_at":              row.UpdatedAt,
			"swagger_import_key_hash": row.SwaggerImportKeyHash,
			"pre_request_script":      row.PreRequestScript,
			"post_response_script":    row.PostResponseScript,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
func (d *ExternalAPI) ListAPIs(ctx context.Context, clientKey string) ([]model.ExternalAPI, error) {
	var rows []model.ExternalAPI
	err := d.db.WithContext(ctx).Where("client_key = ? AND deleted_at = 0", clientKey).Order("group_id ASC, method ASC, path ASC").Find(&rows).Error
	return rows, err
}
func (d *ExternalAPI) ListGroups(ctx context.Context, clientKey string) ([]model.ExternalAPIGroup, error) {
	var rows []model.ExternalAPIGroup
	err := d.db.WithContext(ctx).Where("client_key = ? AND deleted_at = 0", clientKey).Order("sort ASC, name ASC").Find(&rows).Error
	return rows, err
}
func (d *ExternalAPI) GetGroup(ctx context.Context, clientKey, groupID string) (*model.ExternalAPIGroup, error) {
	var row model.ExternalAPIGroup
	if err := d.db.WithContext(ctx).Where("client_key = ? AND uuid = ? AND deleted_at = 0", clientKey, groupID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}
func (d *ExternalAPI) GetGroupByName(ctx context.Context, clientKey, name string) (*model.ExternalAPIGroup, error) {
	var row model.ExternalAPIGroup
	if err := d.db.WithContext(ctx).Where("client_key = ? AND parent_id = '' AND name = ? AND deleted_at = 0", clientKey, name).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}
func (d *ExternalAPI) CreateGroup(ctx context.Context, row *model.ExternalAPIGroup) error {
	return d.db.WithContext(ctx).Create(row).Error
}
func (d *ExternalAPI) AssignUngroupedAPIs(ctx context.Context, clientKey, groupID string) error {
	return d.db.WithContext(ctx).Model(&model.ExternalAPI{}).Where("client_key = ? AND group_id = '' AND deleted_at = 0", clientKey).Update("group_id", groupID).Error
}
func (d *ExternalAPI) GetAPI(ctx context.Context, clientKey, apiKey string) (*model.ExternalAPI, error) {
	var row model.ExternalAPI
	if err := d.db.WithContext(ctx).Where("client_key = ? AND api_key = ? AND deleted_at = 0", clientKey, apiKey).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}
func (d *ExternalAPI) GetAPIByKey(ctx context.Context, apiKey string) (*model.ExternalAPI, error) {
	var row model.ExternalAPI
	if err := d.db.WithContext(ctx).Where("api_key = ? AND deleted_at = 0", apiKey).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}
func (d *ExternalAPI) CreateAPI(ctx context.Context, row *model.ExternalAPI) error {
	return d.db.WithContext(ctx).Create(row).Error
}
func (d *ExternalAPI) UpdateAPI(ctx context.Context, row *model.ExternalAPI) error {
	return d.db.WithContext(ctx).Save(row).Error
}
func (d *ExternalAPI) DeleteAPI(ctx context.Context, row *model.ExternalAPI) error {
	return d.db.WithContext(ctx).
		Where("client_key = ? AND api_key = ? AND deleted_at = 0", row.ClientKey, row.APIKey).
		Delete(&model.ExternalAPI{}).Error
}
func (d *ExternalAPI) GetAPIByMethodPath(ctx context.Context, clientKey, method, path string) (*model.ExternalAPI, error) {
	var row model.ExternalAPI
	if err := d.db.WithContext(ctx).Where("client_key = ? AND method = ? AND path = ? AND deleted_at = 0", clientKey, method, path).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// ImportDocument keeps a document import atomic. The transient endpoint GroupID
// contains a group name when it enters this method and is replaced by its UUID.
func (d *ExternalAPI) ImportDocument(ctx context.Context, groups []model.ExternalAPIGroup, apis []model.ExternalAPI) (created, updated int, err error) {
	err = d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		groupIDs := make(map[string]string, len(groups))
		for index := range groups {
			incoming := groups[index]
			var saved model.ExternalAPIGroup
			query := tx.Where("client_key = ? AND parent_id = '' AND name = ?", incoming.ClientKey, incoming.Name).First(&saved)
			if query.Error != nil && query.Error != gorm.ErrRecordNotFound {
				return query.Error
			}
			if query.Error == gorm.ErrRecordNotFound {
				if err := tx.Create(&incoming).Error; err != nil {
					// Concurrent imports can create the same tag at the same time.
					if reloadErr := tx.Where("client_key = ? AND parent_id = '' AND name = ?", incoming.ClientKey, incoming.Name).First(&saved).Error; reloadErr != nil {
						return err
					}
				} else {
					saved = incoming
				}
			}
			groupIDs[incoming.Name] = saved.UUID
		}
		for index := range apis {
			incoming := apis[index]
			groupID, ok := groupIDs[incoming.GroupID]
			if !ok {
				return gorm.ErrRecordNotFound
			}
			incoming.GroupID = groupID
			var saved model.ExternalAPI
			query := tx.Where("client_key = ? AND method = ? AND path = ?", incoming.ClientKey, incoming.Method, incoming.Path).First(&saved)
			if query.Error != nil && query.Error != gorm.ErrRecordNotFound {
				return query.Error
			}
			found := query.Error == nil
			if query.Error == gorm.ErrRecordNotFound {
				// Imported API keys are a stable derivative of client, method and path.
				// The fallback covers records created by an earlier import implementation.
				query = tx.Where("api_key = ?", incoming.APIKey).First(&saved)
				if query.Error != nil && query.Error != gorm.ErrRecordNotFound {
					return query.Error
				}
				found = query.Error == nil
			}
			if found {
				incoming.ID = saved.ID
				incoming.UUID = saved.UUID
				incoming.APIKey = saved.APIKey
				incoming.CreatedBy = saved.CreatedBy
				incoming.CreatedAt = saved.CreatedAt
				if !externalAPIImportChanged(&saved, &incoming) {
					continue
				}
				updated++
			} else {
				created++
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "api_key"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"group_id", "name", "method", "path", "headers_json", "request_schema_json",
					"response_schema_json", "description", "status", "updated_by", "updated_at", "deleted_at",
				}),
			}).Create(&incoming).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return created, updated, err
}

func externalAPIImportChanged(saved, incoming *model.ExternalAPI) bool {
	return saved.ClientKey != incoming.ClientKey ||
		saved.GroupID != incoming.GroupID ||
		saved.Name != incoming.Name ||
		saved.Method != incoming.Method ||
		saved.Path != incoming.Path ||
		saved.HeadersJSON != incoming.HeadersJSON ||
		saved.RequestSchemaJSON != incoming.RequestSchemaJSON ||
		saved.ResponseSchemaJSON != incoming.ResponseSchemaJSON ||
		saved.Description != incoming.Description ||
		saved.Status != incoming.Status ||
		saved.DeletedAt != incoming.DeletedAt
}
