package dao

import (
	"context"

	"gorm.io/gorm"
)

type GeneratedFunctionMenuRecord struct {
	ID         string `gorm:"column:id"`
	ParentID   string `gorm:"column:parent_id"`
	MenuType   int    `gorm:"column:menu_type"`
	SourceType string `gorm:"column:source_type"`
	SourceID   string `gorm:"column:source_id"`
}

func (GeneratedFunctionMenuRecord) TableName() string {
	return "sys_menus"
}

type generatedFunctionRoleMenuRecord struct{}

func (generatedFunctionRoleMenuRecord) TableName() string {
	return "sys_role_menus"
}

type GeneratedFunctionMenu struct {
	db *gorm.DB
}

func NewGeneratedFunctionMenu(db *gorm.DB) *GeneratedFunctionMenu {
	return &GeneratedFunctionMenu{db: db}
}

func (d *GeneratedFunctionMenu) ListGeneratedFunctionMenus(ctx context.Context) ([]GeneratedFunctionMenuRecord, error) {
	var items []GeneratedFunctionMenuRecord
	err := d.db.WithContext(ctx).
		Table("sys_menus").
		Where("source_type = ?", "generated_function").
		Order("sort ASC, created_at ASC").
		Find(&items).Error
	return items, err
}

func (d *GeneratedFunctionMenu) DeleteByID(ctx context.Context, id string) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("menu_id = ?", id).Delete(&generatedFunctionRoleMenuRecord{}).Error; err != nil {
			return err
		}
		return tx.Table("sys_menus").Where("id = ?", id).Delete(&GeneratedFunctionMenuRecord{}).Error
	})
}
