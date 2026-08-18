package dao

import (
	"context"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FunctionSkill struct{ db *gorm.DB }

func NewFunctionSkill(db *gorm.DB) *FunctionSkill { return &FunctionSkill{db: db} }

func (d *FunctionSkill) GetByFunctionID(ctx context.Context, functionID string) (*model.FunctionSkill, error) {
	var item model.FunctionSkill
	if err := d.db.WithContext(ctx).Where("function_id = ?", functionID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *FunctionSkill) Get(ctx context.Context, id string) (*model.FunctionSkill, error) {
	var item model.FunctionSkill
	if err := d.db.WithContext(ctx).Where("uuid = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *FunctionSkill) Upsert(ctx context.Context, item *model.FunctionSkill) error {
	return d.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "function_id"}}, DoUpdates: clause.AssignmentColumns([]string{"name", "description", "tool_prefix", "status", "updated_at"})}).Create(item).Error
}

func (d *FunctionSkill) SetStatus(ctx context.Context, functionID, status string, now int64) error {
	return d.db.WithContext(ctx).Model(&model.FunctionSkill{}).Where("function_id = ?", functionID).Updates(map[string]any{"status": status, "updated_at": now}).Error
}

func (d *FunctionSkill) ListEnabledByIDs(ctx context.Context, ids []string) ([]model.FunctionSkill, error) {
	var items []model.FunctionSkill
	if len(ids) == 0 {
		return items, nil
	}
	err := d.db.WithContext(ctx).Where("uuid IN ? AND status = ?", ids, model.FunctionSkillStatusEnabled).Find(&items).Error
	return items, err
}

func (d *FunctionSkill) ListEnabled(ctx context.Context) ([]model.FunctionSkill, error) {
	var items []model.FunctionSkill
	err := d.db.WithContext(ctx).Where("status = ?", model.FunctionSkillStatusEnabled).Order("updated_at DESC").Find(&items).Error
	return items, err
}

type FunctionSkillRelease struct{ db *gorm.DB }

func NewFunctionSkillRelease(db *gorm.DB) *FunctionSkillRelease { return &FunctionSkillRelease{db: db} }
func (d *FunctionSkillRelease) Create(ctx context.Context, item *model.FunctionSkillRelease) error {
	return d.db.WithContext(ctx).Create(item).Error
}
func (d *FunctionSkillRelease) ActiveBySkill(ctx context.Context, skillID string) (*model.FunctionSkillRelease, error) {
	var item model.FunctionSkillRelease
	if err := d.db.WithContext(ctx).Where("skill_id = ? AND status = ?", skillID, model.FunctionSkillReleaseStatusActive).Order("id DESC").First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}
func (d *FunctionSkillRelease) ActiveByFunctionRelease(ctx context.Context, releaseID string) (*model.FunctionSkillRelease, error) {
	var item model.FunctionSkillRelease
	if err := d.db.WithContext(ctx).Where("function_release_id = ? AND status = ?", releaseID, model.FunctionSkillReleaseStatusActive).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}
func (d *FunctionSkillRelease) RevokeByFunctionID(ctx context.Context, functionID string, now int64) error {
	return d.db.WithContext(ctx).Model(&model.FunctionSkillRelease{}).Where("function_id = ? AND status = ?", functionID, model.FunctionSkillReleaseStatusActive).Updates(map[string]any{"status": model.FunctionSkillReleaseStatusRevoked, "updated_at": now}).Error
}

type FunctionSkillGrant struct{ db *gorm.DB }

func NewFunctionSkillGrant(db *gorm.DB) *FunctionSkillGrant { return &FunctionSkillGrant{db: db} }
func (d *FunctionSkillGrant) Create(ctx context.Context, item *model.FunctionSkillGrant) error {
	return d.db.WithContext(ctx).Create(item).Error
}
func (d *FunctionSkillGrant) GetByTokenHash(ctx context.Context, hash string) (*model.FunctionSkillGrant, error) {
	var item model.FunctionSkillGrant
	if err := d.db.WithContext(ctx).Where("token_hash = ?", hash).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}
func (d *FunctionSkillGrant) RevokeBySkillID(ctx context.Context, skillID string, now int64) error {
	return d.db.WithContext(ctx).Model(&model.FunctionSkillGrant{}).Where("revoked_at = 0 AND skill_ids LIKE ?", "%\""+skillID+"\"%").Updates(map[string]any{"revoked_at": now}).Error
}

type FunctionSkillApproval struct{ db *gorm.DB }

func NewFunctionSkillApproval(db *gorm.DB) *FunctionSkillApproval {
	return &FunctionSkillApproval{db: db}
}
func (d *FunctionSkillApproval) Create(ctx context.Context, item *model.FunctionSkillApproval) error {
	return d.db.WithContext(ctx).Create(item).Error
}
func (d *FunctionSkillApproval) Consume(ctx context.Context, hash string, now int64) (*model.FunctionSkillApproval, error) {
	var item model.FunctionSkillApproval
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("token_hash = ?", hash).First(&item).Error; err != nil {
			return err
		}
		if item.UsedAt != 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Model(&model.FunctionSkillApproval{}).Where("id = ? AND used_at = 0", item.ID).Update("used_at", now).Error
	})
	if err != nil {
		return nil, err
	}
	return &item, nil
}

type FunctionSkillExecution struct{ db *gorm.DB }

func NewFunctionSkillExecution(db *gorm.DB) *FunctionSkillExecution {
	return &FunctionSkillExecution{db: db}
}
func (d *FunctionSkillExecution) GetByIdempotency(ctx context.Context, releaseID, toolUseID string) (*model.FunctionSkillExecution, error) {
	var item model.FunctionSkillExecution
	if err := d.db.WithContext(ctx).Where("skill_release_id = ? AND tool_use_id = ?", releaseID, toolUseID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}
func (d *FunctionSkillExecution) Create(ctx context.Context, item *model.FunctionSkillExecution) error {
	return d.db.WithContext(ctx).Create(item).Error
}
func (d *FunctionSkillExecution) ListByFunctionID(ctx context.Context, functionID string, limit int) ([]model.FunctionSkillExecution, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var items []model.FunctionSkillExecution
	err := d.db.WithContext(ctx).Where("function_id = ?", functionID).Order("created_at DESC").Limit(limit).Find(&items).Error
	return items, err
}
