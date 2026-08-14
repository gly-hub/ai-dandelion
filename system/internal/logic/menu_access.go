package logic

import (
	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/ai-dandelion/system/internal/model"
)

func expandAllowedMenuIDs(all []model.Menu, seed map[string]struct{}) map[string]struct{} {
	byID := make(map[string]model.Menu, len(all))
	platformByModule := make(map[string][]model.Menu, len(all))
	for i := range all {
		byID[all[i].ID] = all[i]
		if all[i].Placement == model.MenuPlacementPlatform && all[i].Module != "" {
			platformByModule[all[i].Module] = append(platformByModule[all[i].Module], all[i])
		}
	}
	allowed := make(map[string]struct{}, len(seed))
	var visitUp func(id string)
	visitUp = func(id string) {
		if id == "" {
			return
		}
		if _, ok := allowed[id]; ok {
			return
		}
		menu, ok := byID[id]
		if !ok {
			return
		}
		allowed[id] = struct{}{}
		visitUp(menu.ParentID)
	}
	for id := range seed {
		visitUp(id)
	}
	for id := range allowed {
		menu, ok := byID[id]
		if !ok {
			continue
		}
		if menu.Placement != model.MenuPlacementModuleNav || menu.Module == "" {
			continue
		}
		for _, platformMenu := range platformByModule[menu.Module] {
			allowed[platformMenu.ID] = struct{}{}
		}
	}
	return allowed
}

func filterProtoMenusByAllowed(items []*systemproto.Menu, allowed map[string]struct{}) []*systemproto.Menu {
	if len(allowed) == 0 {
		return nil
	}
	filtered := make([]*systemproto.Menu, 0, len(items))
	for _, item := range items {
		if _, ok := allowed[item.Id]; ok {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func menuIDsToSet(menuIDs []string) map[string]struct{} {
	set := make(map[string]struct{}, len(menuIDs))
	for _, id := range menuIDs {
		if id == "" {
			continue
		}
		set[id] = struct{}{}
	}
	return set
}
