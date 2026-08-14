package logic

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	systemproto "github.com/team-dandelion/ai-dandelion/proto/system"
	"github.com/team-dandelion/ai-dandelion/system/internal/dao"
	"github.com/team-dandelion/ai-dandelion/system/internal/model"
	"github.com/team-dandelion/ai-dandelion/toolbox/authctx"
	"gorm.io/gorm"
)

const (
	generatedFunctionMenuActionPublish   = "publish"
	generatedFunctionMenuActionUnpublish = "unpublish"
	generatedFunctionMenuActionSync      = "sync"
	generatedFunctionMenuActionDelete    = "delete"
)

func functionMenuCode(functionID string) string {
	return "func-operation.app." + strings.TrimSpace(functionID)
}

func functionMenuViewKey(functionID string) string {
	return "func:" + strings.TrimSpace(functionID)
}

func functionActionMenuCode(functionID, actionKey string) string {
	functionID = strings.TrimSpace(functionID)
	actionKey = strings.TrimSpace(actionKey)
	if functionID == "" || actionKey == "" {
		return ""
	}
	return functionMenuCode(functionID) + "." + actionKey
}

func (m *MenuLogic) SyncGeneratedFunctionMenu(ctx context.Context, req *systemproto.SyncGeneratedFunctionMenuReq) (*systemproto.Menu, error) {
	functionID := strings.TrimSpace(req.GetFunctionId())
	if functionID == "" {
		return nil, errors.New("function_id is required")
	}
	action := strings.TrimSpace(req.GetAction())
	if action == "" {
		action = generatedFunctionMenuActionSync
	}

	switch action {
	case generatedFunctionMenuActionDelete:
		return m.removeGeneratedFunctionMenu(ctx, functionID)
	case generatedFunctionMenuActionUnpublish:
		return m.disableGeneratedFunctionMenu(ctx, functionID)
	case generatedFunctionMenuActionPublish, generatedFunctionMenuActionSync:
		name := strings.TrimSpace(req.GetName())
		if name == "" {
			return nil, errors.New("name is required")
		}
		parentID := strings.TrimSpace(req.GetParentId())
		if parentID == "" {
			return nil, errors.New("parent_id is required")
		}
		if err := m.validateGeneratedFunctionParent(ctx, parentID); err != nil {
			return nil, err
		}
		status := model.MenuStatusEnabled
		if action == generatedFunctionMenuActionSync {
			existing, err := m.lookupGeneratedFunctionMenu(ctx, functionID)
			if err == nil && existing.Status == model.MenuStatusDisabled {
				status = model.MenuStatusDisabled
			}
		}
		actionKeys := normalizeGeneratedFunctionActionKeys(req.GetActionKeys())
		menu, err := m.upsertGeneratedFunctionMenu(ctx, functionID, name, parentID, status, actionKeys)
		if err != nil {
			return nil, err
		}
		if err := m.syncGeneratedFunctionActionMenus(ctx, menu, actionKeys, status); err != nil {
			return nil, err
		}
		if err := m.grantAdminGeneratedFunctionAccess(ctx, functionID); err != nil {
			return nil, err
		}
		return menu, nil
	default:
		return nil, fmt.Errorf("unsupported action: %s", action)
	}
}

// grantAdminGeneratedFunctionAccess preserves the admin role's documented
// "all menus and buttons" contract for menus created after service startup.
func (m *MenuLogic) grantAdminGeneratedFunctionAccess(ctx context.Context, functionID string) error {
	if m == nil || m.menuDao == nil || m.roleDao == nil {
		return errors.New("menu role authorization is not configured")
	}
	adminRole, err := m.roleDao.GetByCode(ctx, model.RoleCodeAdmin)
	if err != nil {
		return err
	}

	menus, err := m.menuDao.ListBySourceType(ctx, model.MenuSourceTypeGeneratedFunction)
	if err != nil {
		return err
	}
	menuIDs := make([]string, 0, len(menus))
	for i := range menus {
		if menus[i].SourceID == functionID {
			menuIDs = append(menuIDs, menus[i].ID)
		}
	}
	return m.roleDao.GrantRoleMenus(ctx, adminRole.ID, menuIDs, nowUnixMicro())
}

func (m *MenuLogic) CheckFunctionMenuAccess(ctx context.Context, req *systemproto.CheckFunctionMenuAccessReq) (bool, error) {
	userID := strings.TrimSpace(req.GetUserId())
	if ctxUserID, err := authctx.RequireUserID(ctx); err == nil {
		userID = ctxUserID
	}
	functionID := strings.TrimSpace(req.GetFunctionId())
	actionKey := strings.TrimSpace(req.GetActionKey())
	if functionID == "" {
		return false, errors.New("function_id is required")
	}
	if userID == "" {
		return false, nil
	}

	menus, err := m.menuDao.List(ctx, dao.MenuListFilter{})
	if err != nil {
		return false, err
	}
	menuIDs, err := m.roleDao.ListMenuIDsByUser(ctx, userID)
	if err != nil {
		return false, err
	}
	if len(menuIDs) == 0 {
		return false, nil
	}
	allowed := expandAllowedMenuIDs(menus, menuIDsToSet(menuIDs))

	target, err := m.lookupGeneratedFunctionMenu(ctx, functionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	if target.Status != model.MenuStatusEnabled || target.Visible != model.MenuVisibleYes {
		return false, nil
	}
	if actionKey == "" {
		_, ok := allowed[target.ID]
		return ok, nil
	}
	actionMenu, err := m.lookupGeneratedFunctionActionMenu(ctx, functionID, actionKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	if actionMenu.Status != model.MenuStatusEnabled || actionMenu.Visible != model.MenuVisibleYes {
		return false, nil
	}
	_, ok := allowed[actionMenu.ID]
	return ok, nil
}

func (m *MenuLogic) CheckMenuAccess(ctx context.Context, req *systemproto.CheckMenuAccessReq) (bool, error) {
	userID := strings.TrimSpace(req.GetUserId())
	if ctxUserID, err := authctx.RequireUserID(ctx); err == nil {
		userID = ctxUserID
	}
	menuCode := strings.TrimSpace(req.GetMenuCode())
	if userID == "" || menuCode == "" {
		return false, nil
	}
	menus, err := m.menuDao.List(ctx, dao.MenuListFilter{})
	if err != nil {
		return false, err
	}
	menuIDs, err := m.roleDao.ListMenuIDsByUser(ctx, userID)
	if err != nil {
		return false, err
	}
	allowed := expandAllowedMenuIDs(menus, menuIDsToSet(menuIDs))
	target, err := m.menuDao.GetByCode(ctx, menuCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	if target.Status != model.MenuStatusEnabled || target.Visible != model.MenuVisibleYes {
		return false, nil
	}
	_, ok := allowed[target.ID]
	return ok, nil
}

func (m *MenuLogic) upsertGeneratedFunctionMenu(ctx context.Context, functionID, name, parentID string, status int, actionKeys []string) (*systemproto.Menu, error) {
	existing, err := m.lookupGeneratedFunctionMenu(ctx, functionID)
	now := nowUnixMicro()
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		menu := &model.Menu{
			ID:         uuid.NewString(),
			ParentID:   parentID,
			Module:     model.MenuModuleFuncOperation,
			Placement:  model.MenuPlacementModuleNav,
			Name:       name,
			Code:       functionMenuCode(functionID),
			ViewKey:    functionMenuViewKey(functionID),
			Icon:       "AppstoreOutlined",
			MenuType:   model.MenuTypeMenu,
			Status:     status,
			Visible:    model.MenuVisibleYes,
			SourceType: model.MenuSourceTypeGeneratedFunction,
			SourceID:   functionID,
			Remark:     generatedFunctionMenuRemark(actionKeys),
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		maxSort, sortErr := m.menuDao.MaxSort(ctx, parentID, menu.Module, menu.Placement)
		if sortErr != nil {
			return nil, sortErr
		}
		menu.Sort = maxSort + 10
		if err := m.menuDao.Create(ctx, menu); err != nil {
			return nil, wrapMenuDuplicateError(err)
		}
		return modelMenuToProto(menu, nil), nil
	}

	existing.ParentID = parentID
	existing.Name = name
	existing.Status = status
	existing.Visible = model.MenuVisibleYes
	existing.Remark = generatedFunctionMenuRemark(actionKeys)
	existing.SourceType = model.MenuSourceTypeGeneratedFunction
	existing.SourceID = functionID
	existing.UpdatedAt = now
	if err := m.menuDao.Update(ctx, existing); err != nil {
		return nil, wrapMenuDuplicateError(err)
	}
	return modelMenuToProto(existing, nil), nil
}

func (m *MenuLogic) disableGeneratedFunctionMenu(ctx context.Context, functionID string) (*systemproto.Menu, error) {
	existing, err := m.lookupGeneratedFunctionMenu(ctx, functionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	now := nowUnixMicro()
	if err := m.menuDao.UpdateStatus(ctx, existing.ID, model.MenuStatusDisabled, now); err != nil {
		return nil, wrapMenuNotFoundError(err)
	}
	if err := m.updateGeneratedFunctionActionMenuStatus(ctx, functionID, model.MenuStatusDisabled, now); err != nil {
		return nil, err
	}
	existing.Status = model.MenuStatusDisabled
	existing.UpdatedAt = now
	return modelMenuToProto(existing, nil), nil
}

func (m *MenuLogic) removeGeneratedFunctionMenu(ctx context.Context, functionID string) (*systemproto.Menu, error) {
	existing, err := m.lookupGeneratedFunctionMenu(ctx, functionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if err := m.deleteGeneratedFunctionActionMenus(ctx, functionID, existing.ID); err != nil {
		return nil, err
	}
	if err := m.menuDao.Delete(ctx, existing.ID); err != nil {
		return nil, wrapMenuNotFoundError(err)
	}
	return modelMenuToProto(existing, nil), nil
}

func (m *MenuLogic) syncGeneratedFunctionActionMenus(ctx context.Context, parent *systemproto.Menu, actionKeys []string, status int) error {
	if parent == nil {
		return nil
	}

	existing, err := m.menuDao.List(ctx, dao.MenuListFilter{
		Module:    model.MenuModuleFuncOperation,
		Placement: model.MenuPlacementModuleNav,
	})
	if err != nil {
		return err
	}

	parentID := strings.TrimSpace(parent.GetId())
	functionID := strings.TrimSpace(parent.GetSourceId())
	if functionID == "" {
		functionID = strings.TrimSpace(parent.GetSourceId())
	}
	allowed := make(map[string]struct{}, len(actionKeys))
	for index, actionKey := range actionKeys {
		actionCode := functionActionMenuCode(functionID, actionKey)
		if actionCode == "" {
			continue
		}
		allowed[actionCode] = struct{}{}

		menuItem, lookupErr := m.lookupGeneratedFunctionActionMenu(ctx, functionID, actionKey)
		now := nowUnixMicro()
		if lookupErr != nil {
			if !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
				return lookupErr
			}
			menuItem = &model.Menu{
				ID:         uuid.NewString(),
				ParentID:   parentID,
				Module:     model.MenuModuleFuncOperation,
				Placement:  model.MenuPlacementModuleNav,
				Name:       generatedFunctionActionMenuName(actionKey),
				Code:       actionCode,
				ViewKey:    actionKey,
				Icon:       "",
				MenuType:   model.MenuTypeButton,
				Sort:       (index + 1) * 10,
				Status:     status,
				Visible:    model.MenuVisibleYes,
				IsDefault:  false,
				Remark:     "功能动作权限",
				SourceType: model.MenuSourceTypeGeneratedFunction,
				SourceID:   functionID,
				CreatedAt:  now,
				UpdatedAt:  now,
			}
			if err := m.menuDao.Create(ctx, menuItem); err != nil {
				return wrapMenuDuplicateError(err)
			}
			continue
		}

		menuItem.ParentID = parentID
		menuItem.Name = generatedFunctionActionMenuName(actionKey)
		menuItem.ViewKey = actionKey
		menuItem.MenuType = model.MenuTypeButton
		menuItem.Sort = (index + 1) * 10
		menuItem.Status = status
		menuItem.Visible = model.MenuVisibleYes
		menuItem.SourceType = model.MenuSourceTypeGeneratedFunction
		menuItem.SourceID = functionID
		menuItem.UpdatedAt = now
		if err := m.menuDao.Update(ctx, menuItem); err != nil {
			return wrapMenuDuplicateError(err)
		}
	}

	for i := range existing {
		item := existing[i]
		if item.ParentID != parentID || item.MenuType != model.MenuTypeButton || item.SourceType != model.MenuSourceTypeGeneratedFunction {
			continue
		}
		if item.SourceID != functionID {
			continue
		}
		if _, ok := allowed[item.Code]; ok {
			continue
		}
		if err := m.menuDao.Delete(ctx, item.ID); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	return nil
}

func (m *MenuLogic) lookupGeneratedFunctionMenu(ctx context.Context, functionID string) (*model.Menu, error) {
	menu, err := m.menuDao.GetBySourceIDAndMenuType(ctx, model.MenuSourceTypeGeneratedFunction, functionID, model.MenuTypeMenu)
	if err == nil {
		return menu, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return m.menuDao.GetByCode(ctx, functionMenuCode(functionID))
}

func (m *MenuLogic) lookupGeneratedFunctionActionMenu(ctx context.Context, functionID, actionKey string) (*model.Menu, error) {
	return m.menuDao.GetByCode(ctx, functionActionMenuCode(functionID, actionKey))
}

func (m *MenuLogic) updateGeneratedFunctionActionMenuStatus(ctx context.Context, functionID string, status int, updatedAt int64) error {
	items, err := m.menuDao.List(ctx, dao.MenuListFilter{
		Module:    model.MenuModuleFuncOperation,
		Placement: model.MenuPlacementModuleNav,
	})
	if err != nil {
		return err
	}
	for i := range items {
		item := items[i]
		if item.SourceType != model.MenuSourceTypeGeneratedFunction || item.SourceID != functionID || item.MenuType != model.MenuTypeButton {
			continue
		}
		if err := m.menuDao.UpdateStatus(ctx, item.ID, status, updatedAt); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	return nil
}

func (m *MenuLogic) deleteGeneratedFunctionActionMenus(ctx context.Context, functionID, parentID string) error {
	items, err := m.menuDao.List(ctx, dao.MenuListFilter{
		Module:    model.MenuModuleFuncOperation,
		Placement: model.MenuPlacementModuleNav,
	})
	if err != nil {
		return err
	}
	for i := range items {
		item := items[i]
		if item.MenuType != model.MenuTypeButton || item.ParentID != parentID {
			continue
		}
		if item.SourceType != model.MenuSourceTypeGeneratedFunction || item.SourceID != functionID {
			continue
		}
		if err := m.menuDao.Delete(ctx, item.ID); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	return nil
}

func (m *MenuLogic) validateGeneratedFunctionParent(ctx context.Context, parentID string) error {
	parent, err := m.menuDao.Get(ctx, parentID)
	if err != nil {
		return errors.New("parent menu not found")
	}
	if parent.Module != model.MenuModuleFuncOperation || parent.Placement != model.MenuPlacementModuleNav {
		return errors.New("parent menu must belong to func-operation module nav")
	}
	if parent.MenuType != model.MenuTypeDirectory {
		return errors.New("parent menu must be directory")
	}
	if parent.Status != model.MenuStatusEnabled {
		return errors.New("parent directory is disabled")
	}
	return m.ensureMenuUnderFuncUse(ctx, parentID)
}

func (m *MenuLogic) ensureMenuUnderFuncUse(ctx context.Context, menuID string) error {
	useMenu, err := m.menuDao.GetByCode(ctx, model.FuncUseMenuCode)
	if err != nil {
		return errors.New("func use menu is not configured")
	}
	allMenus, err := m.menuDao.List(ctx, dao.MenuListFilter{
		Module:    model.MenuModuleFuncOperation,
		Placement: model.MenuPlacementModuleNav,
	})
	if err != nil {
		return err
	}
	byID := make(map[string]model.Menu, len(allMenus))
	for i := range allMenus {
		byID[allMenus[i].ID] = allMenus[i]
	}
	currentID := menuID
	for currentID != "" {
		if currentID == useMenu.ID {
			return nil
		}
		menu, ok := byID[currentID]
		if !ok {
			break
		}
		currentID = menu.ParentID
	}
	return errors.New("parent directory must be under func use menu")
}

func isGeneratedFunctionMenu(menu *model.Menu) bool {
	if menu == nil {
		return false
	}
	if menu.SourceType == model.MenuSourceTypeGeneratedFunction {
		return true
	}
	return strings.HasPrefix(menu.Code, "func-operation.app.")
}

func normalizeGeneratedFunctionActionKeys(items []string) []string {
	normalized := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, key)
	}
	return normalized
}

func generatedFunctionMenuRemark(actionKeys []string) string {
	normalized := normalizeGeneratedFunctionActionKeys(actionKeys)
	if len(normalized) == 0 {
		return "功能发布自动创建"
	}
	sort.Strings(normalized)
	return "功能发布自动创建; actions:" + strings.Join(normalized, ",")
}

func generatedFunctionActionMenuName(actionKey string) string {
	actionKey = strings.TrimSpace(actionKey)
	if actionKey == "" {
		return "未命名动作"
	}
	return actionKey
}
