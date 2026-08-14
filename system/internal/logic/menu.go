package logic

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/ai-dandelion/system/internal/dao"
	"github.com/gly-hub/ai-dandelion/system/internal/model"
	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MenuLogic struct {
	menuDao           *dao.Menu
	roleDao           *dao.Role
	operationLogLogic *OperationLogLogic
}

func NewMenuLogic(menuDao *dao.Menu, roleDao *dao.Role, operationLogs ...*OperationLogLogic) *MenuLogic {
	logic := &MenuLogic{menuDao: menuDao, roleDao: roleDao}
	if len(operationLogs) > 0 {
		logic.operationLogLogic = operationLogs[0]
	}
	return logic
}

func (m *MenuLogic) EnsureSeedMenus(ctx context.Context) error {
	return seedMenus(ctx, m.menuDao)
}

func (m *MenuLogic) ListMenus(ctx context.Context, req *systemproto.ListMenusReq) ([]*systemproto.Menu, error) {
	menus, err := m.menuDao.List(ctx, dao.MenuListFilter{
		Module:    strings.TrimSpace(req.GetModule()),
		Placement: strings.TrimSpace(req.GetPlacement()),
		Status:    int(req.GetStatus()),
	})
	if err != nil {
		return nil, err
	}
	items := make([]*systemproto.Menu, 0, len(menus))
	for i := range menus {
		items = append(items, modelMenuToProto(&menus[i], nil))
	}
	if req.GetTree() {
		return buildMenuTree(items), nil
	}
	return items, nil
}

func (m *MenuLogic) CreateMenu(ctx context.Context, req *systemproto.CreateMenuReq) (*systemproto.Menu, error) {
	menu, err := m.buildMenuFromInput(ctx, "", req.GetParentId(), req.GetModule(), req.GetPlacement(), req.GetName(), req.GetCode(), req.GetViewKey(), req.GetIcon(), int(req.GetMenuType()), int(req.GetSort()), int(req.GetStatus()), int(req.GetVisible()), req.GetIsDefault(), req.GetRemark())
	if err != nil {
		return nil, err
	}
	afterData, err := menuAuditJSON(menu)
	if err != nil {
		return nil, err
	}
	if err := m.menuDao.Transaction(ctx, func(menuDao *dao.Menu) error {
		if err := menuDao.Create(ctx, menu); err != nil {
			return err
		}
		return m.recordMenuChange(ctx, menuDao, "menu.create", "创建菜单", menu, "", afterData)
	}); err != nil {
		return nil, wrapMenuDuplicateError(err)
	}
	return modelMenuToProto(menu, nil), nil
}

func (m *MenuLogic) UpdateMenu(ctx context.Context, req *systemproto.UpdateMenuReq) (*systemproto.Menu, error) {
	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return nil, errors.New("id is required")
	}
	existing, err := m.menuDao.Get(ctx, id)
	if err != nil {
		return nil, wrapMenuNotFoundError(err)
	}
	if isGeneratedFunctionMenu(existing) {
		return nil, errors.New("generated function menu is read-only")
	}
	beforeData, err := menuAuditJSON(existing)
	if err != nil {
		return nil, err
	}
	menu, err := m.buildMenuFromInput(ctx, id, req.GetParentId(), req.GetModule(), req.GetPlacement(), req.GetName(), req.GetCode(), req.GetViewKey(), req.GetIcon(), int(req.GetMenuType()), int(req.GetSort()), int(req.GetStatus()), int(req.GetVisible()), req.GetIsDefault(), req.GetRemark())
	if err != nil {
		return nil, err
	}
	menu.CreatedAt = existing.CreatedAt
	if menu.Sort == 0 {
		menu.Sort = existing.Sort
	}
	afterData, err := menuAuditJSON(menu)
	if err != nil {
		return nil, err
	}
	if err := m.menuDao.Transaction(ctx, func(menuDao *dao.Menu) error {
		if err := menuDao.Update(ctx, menu); err != nil {
			return err
		}
		return m.recordMenuChange(ctx, menuDao, "menu.update", "更新菜单", menu, beforeData, afterData)
	}); err != nil {
		return nil, wrapMenuDuplicateError(err)
	}
	return modelMenuToProto(menu, nil), nil
}

func (m *MenuLogic) DeleteMenu(ctx context.Context, req *systemproto.DeleteMenuReq) error {
	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return errors.New("id is required")
	}
	existing, err := m.menuDao.Get(ctx, id)
	if err != nil {
		return wrapMenuNotFoundError(err)
	}
	if isGeneratedFunctionMenu(existing) {
		return errors.New("generated function menu cannot be deleted manually")
	}
	beforeData, err := menuAuditJSON(existing)
	if err != nil {
		return err
	}
	childCount, err := m.menuDao.CountChildren(ctx, id)
	if err != nil {
		return err
	}
	if childCount > 0 {
		return errors.New("menu has children")
	}
	if err := m.menuDao.Transaction(ctx, func(menuDao *dao.Menu) error {
		if err := menuDao.Delete(ctx, id); err != nil {
			return err
		}
		return m.recordMenuChange(ctx, menuDao, "menu.delete", "删除菜单", existing, beforeData, "")
	}); err != nil {
		return wrapMenuNotFoundError(err)
	}
	return nil
}

func (m *MenuLogic) EnableMenu(ctx context.Context, req *systemproto.EnableMenuReq) (*systemproto.Menu, error) {
	return m.setMenuStatus(ctx, req.GetId(), model.MenuStatusEnabled)
}

func (m *MenuLogic) DisableMenu(ctx context.Context, req *systemproto.DisableMenuReq) (*systemproto.Menu, error) {
	return m.setMenuStatus(ctx, req.GetId(), model.MenuStatusDisabled)
}

func (m *MenuLogic) setMenuStatus(ctx context.Context, id string, status int) (*systemproto.Menu, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("id is required")
	}
	menu, err := m.menuDao.Get(ctx, id)
	if err != nil {
		return nil, wrapMenuNotFoundError(err)
	}
	beforeData, err := menuAuditJSON(menu)
	if err != nil {
		return nil, err
	}
	menu.Status = status
	menu.UpdatedAt = nowUnixMicro()
	afterData, err := menuAuditJSON(menu)
	if err != nil {
		return nil, err
	}
	action, actionLabel := "menu.enable", "启用菜单"
	if status == model.MenuStatusDisabled {
		action, actionLabel = "menu.disable", "禁用菜单"
	}
	if err := m.menuDao.Transaction(ctx, func(menuDao *dao.Menu) error {
		if err := menuDao.UpdateStatus(ctx, id, status, menu.UpdatedAt); err != nil {
			return err
		}
		return m.recordMenuChange(ctx, menuDao, action, actionLabel, menu, beforeData, afterData)
	}); err != nil {
		return nil, wrapMenuNotFoundError(err)
	}
	return modelMenuToProto(menu, nil), nil
}

type menuAuditSnapshot struct {
	ParentID  string `json:"parentId"`
	Module    string `json:"module"`
	Placement string `json:"placement"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	ViewKey   string `json:"viewKey"`
	MenuType  int    `json:"menuType"`
	Sort      int    `json:"sort"`
	Status    int    `json:"status"`
	Visible   int    `json:"visible"`
	IsDefault bool   `json:"isDefault"`
	Remark    string `json:"remark"`
}

func menuAuditJSON(menu *model.Menu) (string, error) {
	data, err := json.Marshal(menuAuditSnapshot{
		ParentID: menu.ParentID, Module: menu.Module, Placement: menu.Placement, Name: menu.Name,
		Code: menu.Code, ViewKey: menu.ViewKey, MenuType: menu.MenuType, Sort: menu.Sort,
		Status: menu.Status, Visible: menu.Visible, IsDefault: menu.IsDefault, Remark: menu.Remark,
	})
	return string(data), err
}

func (m *MenuLogic) recordMenuChange(ctx context.Context, menuDao *dao.Menu, action, actionLabel string, menu *model.Menu, beforeData, afterData string) error {
	if m.operationLogLogic == nil {
		return nil
	}
	operatorName := "系统"
	if operator, ok := authctx.CurrentUser(ctx); ok && strings.TrimSpace(operator.Username) != "" {
		operatorName = operator.Username
	}
	return m.operationLogLogic.RecordWithDAO(ctx, dao.NewOperationLog(menuDao.DB()), OperationLogInput{
		Module: OperationModuleSystem, Action: action, ActionLabel: actionLabel,
		ResourceType: OperationResourceMenu, ResourceID: menu.ID, ResourceName: menu.Name,
		Summary: operatorName + actionLabel + "「" + menu.Name + "」", BeforeData: beforeData, AfterData: afterData,
	})
}

func (m *MenuLogic) buildMenuFromInput(
	ctx context.Context,
	id string,
	parentID string,
	module string,
	placement string,
	name string,
	code string,
	viewKey string,
	icon string,
	menuType int,
	sort int,
	status int,
	visible int,
	isDefault bool,
	remark string,
) (*model.Menu, error) {
	parentID = strings.TrimSpace(parentID)
	module = strings.TrimSpace(module)
	placement = strings.TrimSpace(placement)
	name = strings.TrimSpace(name)
	code = strings.TrimSpace(code)
	viewKey = strings.TrimSpace(viewKey)
	if viewKey == "" {
		viewKey = code
	}
	icon = strings.TrimSpace(icon)
	remark = strings.TrimSpace(remark)

	if module == "" {
		return nil, errors.New("module is required")
	}
	if placement == "" {
		return nil, errors.New("placement is required")
	}
	if name == "" {
		return nil, errors.New("name is required")
	}
	if code == "" {
		return nil, errors.New("code is required")
	}
	if menuType == 0 {
		menuType = model.MenuTypeMenu
	}
	if err := validateMenuType(menuType); err != nil {
		return nil, err
	}
	if err := validatePlacement(placement); err != nil {
		return nil, err
	}
	if status == 0 {
		status = model.MenuStatusEnabled
	}
	status, err := normalizeMenuStatus(int32(status), model.MenuStatusEnabled)
	if err != nil {
		return nil, err
	}
	if visible == 0 {
		visible = model.MenuVisibleYes
	}
	if err := normalizeMenuVisible(int32(visible)); err != nil {
		return nil, err
	}
	if parentID != "" {
		if id != "" && parentID == id {
			return nil, errors.New("parent_id cannot be self")
		}
		parent, err := m.menuDao.Get(ctx, parentID)
		if err != nil {
			return nil, errors.New("parent menu not found")
		}
		if parent.Module != module || parent.Placement != placement {
			return nil, errors.New("parent menu module or placement mismatch")
		}
		if parent.MenuType != model.MenuTypeDirectory {
			if !(parent.MenuType == model.MenuTypeMenu && parent.Code == model.FuncUseMenuCode) {
				return nil, errors.New("parent menu must be directory")
			}
		}
	}
	if sort <= 0 {
		maxSort, err := m.menuDao.MaxSort(ctx, parentID, module, placement)
		if err != nil {
			return nil, err
		}
		sort = maxSort + 10
	}

	now := nowUnixMicro()
	if id == "" {
		id = uuid.NewString()
	}
	menu := &model.Menu{
		ID:         id,
		ParentID:   parentID,
		Module:     module,
		Placement:  placement,
		Name:       name,
		Code:       code,
		ViewKey:    viewKey,
		Icon:       icon,
		MenuType:   menuType,
		Sort:       sort,
		Status:     status,
		Visible:    visible,
		IsDefault:  isDefault,
		Remark:     remark,
		SourceType: model.MenuSourceTypeStatic,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	return menu, nil
}

func buildMenuTree(items []*systemproto.Menu) []*systemproto.Menu {
	byID := make(map[string]*systemproto.Menu, len(items))
	roots := make([]*systemproto.Menu, 0)
	for _, item := range items {
		copy := *item
		copy.Children = nil
		byID[item.Id] = &copy
	}
	for _, item := range byID {
		if item.ParentId == "" {
			roots = append(roots, item)
			continue
		}
		parent, ok := byID[item.ParentId]
		if !ok {
			roots = append(roots, item)
			continue
		}
		parent.Children = append(parent.Children, item)
	}
	sortMenus(roots)
	return roots
}

func modelMenuToProto(menu *model.Menu, children []*systemproto.Menu) *systemproto.Menu {
	if menu == nil {
		return nil
	}
	return &systemproto.Menu{
		Id:         menu.ID,
		ParentId:   menu.ParentID,
		Module:     menu.Module,
		Placement:  menu.Placement,
		Name:       menu.Name,
		Code:       menu.Code,
		ViewKey:    menu.ViewKey,
		Icon:       menu.Icon,
		MenuType:   int32(menu.MenuType),
		Sort:       int32(menu.Sort),
		Status:     int32(menu.Status),
		Visible:    int32(menu.Visible),
		IsDefault:  menu.IsDefault,
		Remark:     menu.Remark,
		SourceType: menu.SourceType,
		SourceId:   menu.SourceID,
		CreatedAt:  menu.CreatedAt,
		Children:   children,
	}
}

func validateMenuType(menuType int) error {
	switch menuType {
	case model.MenuTypeDirectory, model.MenuTypeMenu, model.MenuTypeButton:
		return nil
	default:
		return errors.New("menu_type must be 1 (directory), 2 (menu) or 3 (button)")
	}
}

func validatePlacement(placement string) error {
	switch placement {
	case model.MenuPlacementModuleNav:
		return nil
	default:
		return errors.New("placement must be module_nav")
	}
}

func normalizeMenuStatus(status int32, fallback int) (int, error) {
	if status == 0 {
		return fallback, nil
	}
	switch int(status) {
	case model.MenuStatusEnabled, model.MenuStatusDisabled:
		return int(status), nil
	default:
		return 0, errors.New("status must be 1 (enabled) or 2 (disabled)")
	}
}

func normalizeMenuVisible(visible int32) error {
	switch int(visible) {
	case model.MenuVisibleYes, model.MenuVisibleNo:
		return nil
	default:
		return errors.New("visible must be 1 or 0")
	}
}

func wrapMenuNotFoundError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("menu not found")
	}
	return err
}

func wrapMenuDuplicateError(err error) error {
	if err == nil {
		return nil
	}
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "duplicate") || strings.Contains(lower, "unique") {
		return errors.New("menu code already exists")
	}
	return err
}
