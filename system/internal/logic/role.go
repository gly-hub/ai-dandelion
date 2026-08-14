package logic

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/google/uuid"
	systemproto "github.com/team-dandelion/ai-dandelion/proto/system"
	"github.com/team-dandelion/ai-dandelion/system/internal/dao"
	"github.com/team-dandelion/ai-dandelion/system/internal/model"
	"github.com/team-dandelion/ai-dandelion/toolbox/authctx"
	"gorm.io/gorm"
)

var roleCodePattern = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,62}$`)

type RoleLogic struct {
	roleDao           *dao.Role
	menuDao           *dao.Menu
	operationLogLogic *OperationLogLogic
}

func NewRoleLogic(roleDao *dao.Role, menuDao *dao.Menu, operationLogLogic *OperationLogLogic) *RoleLogic {
	return &RoleLogic{roleDao: roleDao, menuDao: menuDao, operationLogLogic: operationLogLogic}
}

func (r *RoleLogic) ListRoles(ctx context.Context, _ *systemproto.ListRolesReq) ([]*systemproto.Role, error) {
	roles, err := r.roleDao.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*systemproto.Role, 0, len(roles))
	for i := range roles {
		item, err := r.modelRoleToProto(ctx, &roles[i], false)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (r *RoleLogic) CreateRole(ctx context.Context, req *systemproto.CreateRoleReq) (*systemproto.Role, error) {
	name, code, err := validateRoleInput(req.GetName(), req.GetCode())
	if err != nil {
		return nil, err
	}
	status, err := normalizeRoleStatus(req.GetStatus(), model.RoleStatusEnabled)
	if err != nil {
		return nil, err
	}
	now := nowUnixMicro()
	role := &model.Role{
		ID:        uuid.NewString(),
		Name:      name,
		Code:      code,
		Status:    status,
		Remark:    strings.TrimSpace(req.GetRemark()),
		Sort:      int(req.GetSort()),
		CreatedAt: now,
		UpdatedAt: now,
	}
	afterData, err := r.roleAuditJSON(ctx, r.roleDao, role)
	if err != nil {
		return nil, err
	}
	if err := r.roleDao.Transaction(ctx, func(roleDao *dao.Role) error {
		if err := roleDao.Create(ctx, role); err != nil {
			return err
		}
		return r.recordRoleChange(ctx, roleDao, "role.create", "创建角色", role, "", afterData)
	}); err != nil {
		return nil, wrapRoleDuplicateError(err)
	}
	return r.modelRoleToProto(ctx, role, false)
}

func (r *RoleLogic) UpdateRole(ctx context.Context, req *systemproto.UpdateRoleReq) (*systemproto.Role, error) {
	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return nil, errors.New("id is required")
	}
	role, err := r.roleDao.Get(ctx, id)
	if err != nil {
		return nil, wrapRoleNotFoundError(err)
	}
	beforeData, err := r.roleAuditJSON(ctx, r.roleDao, role)
	if err != nil {
		return nil, err
	}
	name, code, err := validateRoleInput(req.GetName(), req.GetCode())
	if err != nil {
		return nil, err
	}
	status, err := normalizeRoleStatus(req.GetStatus(), role.Status)
	if err != nil {
		return nil, err
	}
	role.Name = name
	role.Code = code
	role.Status = status
	role.Remark = strings.TrimSpace(req.GetRemark())
	if req.GetSort() > 0 {
		role.Sort = int(req.GetSort())
	}
	role.UpdatedAt = nowUnixMicro()
	afterData, err := r.roleAuditJSON(ctx, r.roleDao, role)
	if err != nil {
		return nil, err
	}
	if err := r.roleDao.Transaction(ctx, func(roleDao *dao.Role) error {
		if err := roleDao.Update(ctx, role); err != nil {
			return err
		}
		return r.recordRoleChange(ctx, roleDao, "role.update", "更新角色", role, beforeData, afterData)
	}); err != nil {
		return nil, wrapRoleDuplicateError(err)
	}
	return r.modelRoleToProto(ctx, role, false)
}

func (r *RoleLogic) DeleteRole(ctx context.Context, req *systemproto.DeleteRoleReq) error {
	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return errors.New("id is required")
	}
	role, err := r.roleDao.Get(ctx, id)
	if err != nil {
		return wrapRoleNotFoundError(err)
	}
	beforeData, err := r.roleAuditJSON(ctx, r.roleDao, role)
	if err != nil {
		return err
	}
	if err := r.roleDao.Transaction(ctx, func(roleDao *dao.Role) error {
		if err := roleDao.Delete(ctx, id); err != nil {
			return err
		}
		return r.recordRoleChange(ctx, roleDao, "role.delete", "删除角色", role, beforeData, "")
	}); err != nil {
		return wrapRoleNotFoundError(err)
	}
	return nil
}

func (r *RoleLogic) EnableRole(ctx context.Context, req *systemproto.EnableRoleReq) (*systemproto.Role, error) {
	return r.setRoleStatus(ctx, req.GetId(), model.RoleStatusEnabled)
}

func (r *RoleLogic) DisableRole(ctx context.Context, req *systemproto.DisableRoleReq) (*systemproto.Role, error) {
	return r.setRoleStatus(ctx, req.GetId(), model.RoleStatusDisabled)
}

func (r *RoleLogic) setRoleStatus(ctx context.Context, id string, status int) (*systemproto.Role, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("id is required")
	}
	role, err := r.roleDao.Get(ctx, id)
	if err != nil {
		return nil, wrapRoleNotFoundError(err)
	}
	beforeData, err := r.roleAuditJSON(ctx, r.roleDao, role)
	if err != nil {
		return nil, err
	}
	role.Status = status
	role.UpdatedAt = nowUnixMicro()
	afterData, err := r.roleAuditJSON(ctx, r.roleDao, role)
	if err != nil {
		return nil, err
	}
	action, actionLabel := "role.enable", "启用角色"
	if status == model.RoleStatusDisabled {
		action, actionLabel = "role.disable", "禁用角色"
	}
	if err := r.roleDao.Transaction(ctx, func(roleDao *dao.Role) error {
		if err := roleDao.UpdateStatus(ctx, id, status, role.UpdatedAt); err != nil {
			return err
		}
		return r.recordRoleChange(ctx, roleDao, action, actionLabel, role, beforeData, afterData)
	}); err != nil {
		return nil, wrapRoleNotFoundError(err)
	}
	return r.modelRoleToProto(ctx, role, false)
}

func (r *RoleLogic) GetRoleMenus(ctx context.Context, req *systemproto.GetRoleMenusReq) ([]string, error) {
	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return nil, errors.New("id is required")
	}
	if _, err := r.roleDao.Get(ctx, id); err != nil {
		return nil, wrapRoleNotFoundError(err)
	}
	return r.roleDao.ListMenuIDsByRole(ctx, id)
}

func (r *RoleLogic) SetRoleMenus(ctx context.Context, req *systemproto.SetRoleMenusReq) ([]string, error) {
	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return nil, errors.New("id is required")
	}
	role, err := r.roleDao.Get(ctx, id)
	if err != nil {
		return nil, wrapRoleNotFoundError(err)
	}
	beforeData, err := r.roleAuditJSON(ctx, r.roleDao, role)
	if err != nil {
		return nil, err
	}
	menuIDs, err := r.normalizeMenuIDs(ctx, req.GetMenuIds())
	if err != nil {
		return nil, err
	}
	if err := r.roleDao.Transaction(ctx, func(roleDao *dao.Role) error {
		if err := roleDao.SetRoleMenus(ctx, id, menuIDs, nowUnixMicro()); err != nil {
			return err
		}
		afterData, err := r.roleAuditJSON(ctx, roleDao, role)
		if err != nil {
			return err
		}
		return r.recordRoleChange(ctx, roleDao, "role.menu.update", "更新角色菜单权限", role, beforeData, afterData)
	}); err != nil {
		return nil, err
	}
	return menuIDs, nil
}

type roleAuditSnapshot struct {
	Name   string          `json:"name"`
	Code   string          `json:"code"`
	Status int             `json:"status"`
	Remark string          `json:"remark"`
	Sort   int             `json:"sort"`
	Menus  []roleAuditMenu `json:"menus"`
}

type roleAuditMenu struct {
	ID       string `json:"id"`
	ParentID string `json:"parentId"`
	MenuType int    `json:"menuType"`
	Name     string `json:"name"`
	Code     string `json:"code"`
}

func (r *RoleLogic) roleAuditJSON(ctx context.Context, roleDao *dao.Role, role *model.Role) (string, error) {
	menus, err := roleDao.ListMenusByRole(ctx, role.ID)
	if err != nil {
		return "", err
	}
	snapshot := roleAuditSnapshot{
		Name:   role.Name,
		Code:   role.Code,
		Status: role.Status,
		Remark: role.Remark,
		Sort:   role.Sort,
		Menus:  make([]roleAuditMenu, 0, len(menus)),
	}
	for _, menu := range menus {
		snapshot.Menus = append(snapshot.Menus, roleAuditMenu{
			ID:       menu.ID,
			ParentID: menu.ParentID,
			MenuType: menu.MenuType,
			Name:     menu.Name,
			Code:     menu.Code,
		})
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (r *RoleLogic) recordRoleChange(ctx context.Context, roleDao *dao.Role, action, actionLabel string, role *model.Role, beforeData, afterData string) error {
	if r.operationLogLogic == nil {
		return nil
	}
	operatorName := "系统"
	if user, ok := authctx.CurrentUser(ctx); ok && strings.TrimSpace(user.Username) != "" {
		operatorName = user.Username
	}
	summary := operatorName + actionLabel + "「" + role.Name + "」"
	return r.operationLogLogic.RecordWithDAO(ctx, dao.NewOperationLog(roleDao.DB()), OperationLogInput{
		Module:       OperationModuleSystem,
		Action:       action,
		ActionLabel:  actionLabel,
		ResourceType: OperationResourceRole,
		ResourceID:   role.ID,
		ResourceName: role.Name,
		Summary:      summary,
		BeforeData:   beforeData,
		AfterData:    afterData,
	})
}

func (r *RoleLogic) normalizeMenuIDs(ctx context.Context, menuIDs []string) ([]string, error) {
	unique := make([]string, 0, len(menuIDs))
	seen := make(map[string]struct{}, len(menuIDs))
	for _, menuID := range menuIDs {
		menuID = strings.TrimSpace(menuID)
		if menuID == "" {
			continue
		}
		if _, ok := seen[menuID]; ok {
			continue
		}
		if _, err := r.menuDao.Get(ctx, menuID); err != nil {
			return nil, errors.New("menu not found: " + menuID)
		}
		seen[menuID] = struct{}{}
		unique = append(unique, menuID)
	}
	return unique, nil
}

func (r *RoleLogic) modelRoleToProto(ctx context.Context, role *model.Role, withMenus bool) (*systemproto.Role, error) {
	if role == nil {
		return nil, nil
	}
	item := &systemproto.Role{
		Id:        role.ID,
		Name:      role.Name,
		Code:      role.Code,
		Status:    int32(role.Status),
		Remark:    role.Remark,
		Sort:      int32(role.Sort),
		CreatedAt: role.CreatedAt,
	}
	if !withMenus {
		return item, nil
	}
	menuIDs, err := r.roleDao.ListMenuIDsByRole(ctx, role.ID)
	if err != nil {
		return nil, err
	}
	item.MenuIds = menuIDs
	return item, nil
}

func validateRoleInput(name, code string) (string, string, error) {
	name = strings.TrimSpace(name)
	code = strings.TrimSpace(strings.ToLower(code))
	if name == "" {
		return "", "", errors.New("name is required")
	}
	if code == "" {
		return "", "", errors.New("code is required")
	}
	if !roleCodePattern.MatchString(code) {
		return "", "", errors.New("code must start with a letter and contain lowercase letters, numbers, dot, underscore or hyphen")
	}
	return name, code, nil
}

func normalizeRoleStatus(status int32, fallback int) (int, error) {
	if status == 0 {
		return fallback, nil
	}
	switch int(status) {
	case model.RoleStatusEnabled, model.RoleStatusDisabled:
		return int(status), nil
	default:
		return 0, errors.New("status must be 1 (enabled) or 2 (disabled)")
	}
}

func wrapRoleNotFoundError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("role not found")
	}
	return err
}

func wrapRoleDuplicateError(err error) error {
	if err == nil {
		return nil
	}
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "duplicate") || strings.Contains(lower, "unique") {
		return errors.New("role code already exists")
	}
	return err
}
