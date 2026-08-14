package dao

import (
	"context"
	"fmt"

	"github.com/team-dandelion/ai-dandelion/system/internal/model"
	"gorm.io/gorm"
)

type MenuListFilter struct {
	Module    string
	Placement string
	Status    int
}

type Menu struct {
	db *gorm.DB
}

func NewMenu(db *gorm.DB) *Menu {
	return &Menu{db: db}
}

func (m *Menu) Transaction(ctx context.Context, fn func(*Menu) error) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(NewMenu(tx))
	})
}

func (m *Menu) DB() *gorm.DB {
	return m.db
}

func (m *Menu) List(ctx context.Context, filter MenuListFilter) ([]model.Menu, error) {
	var menus []model.Menu
	query := m.db.WithContext(ctx)
	if filter.Module != "" {
		query = query.Where("module = ?", filter.Module)
	}
	if filter.Placement != "" {
		query = query.Where("placement = ?", filter.Placement)
	}
	if filter.Status > 0 {
		query = query.Where("status = ?", filter.Status)
	}
	err := query.Order("sort ASC, created_at ASC").Find(&menus).Error
	return menus, err
}

func (m *Menu) Count(ctx context.Context) (int64, error) {
	var count int64
	err := m.db.WithContext(ctx).Model(&model.Menu{}).Count(&count).Error
	return count, err
}

func (m *Menu) Create(ctx context.Context, menu *model.Menu) error {
	return m.db.WithContext(ctx).Create(menu).Error
}

// ReassignID migrates a legacy seed row to its stable ID while preserving role grants.
func (m *Menu) ReassignID(ctx context.Context, oldID, newID string) error {
	if oldID == newID {
		return nil
	}

	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var targetCount int64
		if err := tx.Model(&model.Menu{}).Where("id = ?", newID).Count(&targetCount).Error; err != nil {
			return err
		}
		if targetCount > 0 {
			return fmt.Errorf("cannot reassign menu ID %q to %q: target already exists", oldID, newID)
		}

		var grants []model.RoleMenu
		if err := tx.Where("menu_id = ?", oldID).Find(&grants).Error; err != nil {
			return err
		}
		for _, grant := range grants {
			var existingCount int64
			if err := tx.Model(&model.RoleMenu{}).
				Where("role_id = ? AND menu_id = ?", grant.RoleID, newID).
				Count(&existingCount).Error; err != nil {
				return err
			}
			if existingCount == 0 {
				grant.MenuID = newID
				if err := tx.Create(&grant).Error; err != nil {
					return err
				}
			}
			if err := tx.Where("role_id = ? AND menu_id = ?", grant.RoleID, oldID).
				Delete(&model.RoleMenu{}).Error; err != nil {
				return err
			}
		}

		result := tx.Model(&model.Menu{}).Where("id = ?", oldID).Update("id", newID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (m *Menu) GetByCode(ctx context.Context, code string) (*model.Menu, error) {
	var menu model.Menu
	err := m.db.WithContext(ctx).Where("code = ?", code).First(&menu).Error
	if err != nil {
		return nil, err
	}
	return &menu, nil
}

func (m *Menu) Get(ctx context.Context, id string) (*model.Menu, error) {
	var menu model.Menu
	err := m.db.WithContext(ctx).Where("id = ?", id).First(&menu).Error
	if err != nil {
		return nil, err
	}
	return &menu, nil
}

func (m *Menu) GetBySourceID(ctx context.Context, sourceType, sourceID string) (*model.Menu, error) {
	var menu model.Menu
	err := m.db.WithContext(ctx).
		Where("source_type = ? AND source_id = ?", sourceType, sourceID).
		First(&menu).Error
	if err != nil {
		return nil, err
	}
	return &menu, nil
}

func (m *Menu) GetBySourceIDAndMenuType(ctx context.Context, sourceType, sourceID string, menuType int) (*model.Menu, error) {
	var menu model.Menu
	err := m.db.WithContext(ctx).
		Where("source_type = ? AND source_id = ? AND menu_type = ?", sourceType, sourceID, menuType).
		First(&menu).Error
	if err != nil {
		return nil, err
	}
	return &menu, nil
}

func (m *Menu) ListBySourceType(ctx context.Context, sourceType string) ([]model.Menu, error) {
	var menus []model.Menu
	err := m.db.WithContext(ctx).
		Where("source_type = ?", sourceType).
		Order("sort ASC, created_at ASC").
		Find(&menus).Error
	return menus, err
}

func (m *Menu) Delete(ctx context.Context, id string) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("menu_id = ?", id).Delete(&model.RoleMenu{}).Error; err != nil {
			return err
		}
		result := tx.Where("id = ?", id).Delete(&model.Menu{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (m *Menu) CountChildren(ctx context.Context, parentID string) (int64, error) {
	var count int64
	err := m.db.WithContext(ctx).Model(&model.Menu{}).Where("parent_id = ?", parentID).Count(&count).Error
	return count, err
}

func (m *Menu) MaxSort(ctx context.Context, parentID, module, placement string) (int, error) {
	var maxSort *int
	err := m.db.WithContext(ctx).
		Model(&model.Menu{}).
		Where("parent_id = ? AND module = ? AND placement = ?", parentID, module, placement).
		Select("MAX(sort)").
		Scan(&maxSort).Error
	if err != nil {
		return 0, err
	}
	if maxSort == nil {
		return 0, nil
	}
	return *maxSort, nil
}

func (m *Menu) Update(ctx context.Context, menu *model.Menu) error {
	return m.db.WithContext(ctx).
		Model(&model.Menu{}).
		Where("id = ?", menu.ID).
		Updates(map[string]any{
			"parent_id":   menu.ParentID,
			"module":      menu.Module,
			"placement":   menu.Placement,
			"name":        menu.Name,
			"code":        menu.Code,
			"view_key":    menu.ViewKey,
			"icon":        menu.Icon,
			"menu_type":   menu.MenuType,
			"sort":        menu.Sort,
			"status":      menu.Status,
			"visible":     menu.Visible,
			"is_default":  menu.IsDefault,
			"remark":      menu.Remark,
			"source_type": menu.SourceType,
			"source_id":   menu.SourceID,
			"updated_at":  menu.UpdatedAt,
		}).Error
}

func (m *Menu) UpdateStatus(ctx context.Context, id string, status int, updatedAt int64) error {
	result := m.db.WithContext(ctx).
		Model(&model.Menu{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     status,
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
