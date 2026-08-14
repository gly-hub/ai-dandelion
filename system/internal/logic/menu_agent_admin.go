package logic

import (
	"context"

	"github.com/gly-hub/ai-dandelion/system/internal/dao"
)

func (m *MenuLogic) EnsureAgentAdminMenus(ctx context.Context) error {
	return ensureAgentAdminMenus(ctx, m.menuDao)
}

func ensureAgentAdminMenus(ctx context.Context, menuDao *dao.Menu) error {
	// SeedMenus owns this hierarchy. Keeping this compatibility hook as a no-op
	// prevents role seeding from moving Agent menus back to a separate module.
	return nil
}
