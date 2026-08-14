package logic

import (
	"context"
	"sort"
	"strings"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/ai-dandelion/system/internal/dao"
	"github.com/gly-hub/ai-dandelion/system/internal/model"
	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
)

func (m *MenuLogic) GetNavMenus(ctx context.Context, req *systemproto.GetNavMenusReq) ([]*systemproto.Menu, error) {
	menus, err := m.menuDao.List(ctx, dao.MenuListFilter{
		Status: model.MenuStatusEnabled,
	})
	if err != nil {
		return nil, err
	}

	items := make([]*systemproto.Menu, 0, len(menus))
	for i := range menus {
		if menus[i].Visible != model.MenuVisibleYes {
			continue
		}
		items = append(items, modelMenuToProto(&menus[i], nil))
	}

	userID := strings.TrimSpace(req.GetUserId())
	if ctxUserID, err := authctx.RequireUserID(ctx); err == nil {
		userID = ctxUserID
	}
	if userID == "" {
		return []*systemproto.Menu{}, nil
	}
	menuIDs, err := m.roleDao.ListMenuIDsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(menuIDs) == 0 {
		return []*systemproto.Menu{}, nil
	}
	allowed := expandAllowedMenuIDs(menus, menuIDsToSet(menuIDs))
	items = filterProtoMenusByAllowed(items, allowed)

	tree := buildUnifiedNavTree(items)
	module := strings.TrimSpace(req.GetModule())
	if module == "" {
		return tree, nil
	}
	return filterNavTreeByModule(tree, module), nil
}

func buildUnifiedNavTree(items []*systemproto.Menu) []*systemproto.Menu {
	platformMenus := make([]*systemproto.Menu, 0)
	navByModule := make(map[string][]*systemproto.Menu)
	for _, item := range items {
		switch item.Placement {
		case model.MenuPlacementPlatform:
			platformMenus = append(platformMenus, item)
		case model.MenuPlacementModuleNav:
			navByModule[item.Module] = append(navByModule[item.Module], item)
		}
	}

	sortMenus(platformMenus)
	roots := make([]*systemproto.Menu, 0, len(platformMenus))
	for _, platform := range platformMenus {
		copy := *platform
		children := buildMenuTree(navByModule[platform.Module])
		sortMenus(children)
		delete(navByModule, platform.Module)
		copy.Children = children
		roots = append(roots, &copy)
	}

	if len(navByModule) == 0 {
		return roots
	}

	modules := make([]string, 0, len(navByModule))
	for module := range navByModule {
		modules = append(modules, module)
	}
	sort.Strings(modules)
	for _, module := range modules {
		children := buildMenuTree(navByModule[module])
		sortMenus(children)
		roots = append(roots, &systemproto.Menu{
			Id:       "orphan:" + module,
			Module:   module,
			Name:     module,
			Children: children,
		})
	}
	return roots
}

func filterNavTreeByModule(tree []*systemproto.Menu, module string) []*systemproto.Menu {
	for _, item := range tree {
		if item.Module == module || item.ViewKey == module {
			copy := *item
			return []*systemproto.Menu{&copy}
		}
	}
	if children := buildMenuTree(collectNavMenusByModule(tree, module)); len(children) > 0 {
		sortMenus(children)
		return []*systemproto.Menu{{
			Id:       "module:" + module,
			Module:   module,
			ViewKey:  module,
			Name:     module,
			Children: children,
		}}
	}
	return nil
}

func collectNavMenusByModule(tree []*systemproto.Menu, module string) []*systemproto.Menu {
	items := make([]*systemproto.Menu, 0)
	var walk func(nodes []*systemproto.Menu)
	walk = func(nodes []*systemproto.Menu) {
		for _, node := range nodes {
			if node.Placement == model.MenuPlacementModuleNav && node.Module == module {
				copy := *node
				copy.Children = nil
				items = append(items, &copy)
			}
			if len(node.Children) > 0 {
				walk(node.Children)
			}
		}
	}
	walk(tree)
	return items
}

func sortMenus(items []*systemproto.Menu) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Sort == items[j].Sort {
			return items[i].Name < items[j].Name
		}
		return items[i].Sort < items[j].Sort
	})
}
