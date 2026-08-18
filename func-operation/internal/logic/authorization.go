package logic

import (
	"context"
	"errors"
	"strings"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
)

const (
	functionPermissionCreate         = "func-operation.functions.create"
	functionPermissionEdit           = "func-operation.functions.edit"
	functionPermissionPublish        = "func-operation.functions.publish"
	functionPermissionUnpublish      = "func-operation.functions.unpublish"
	functionPermissionDelete         = "func-operation.functions.delete"
	functionPermissionSkillConfigure = "func-operation.functions.skill.configure"
	publicConfigPermissionView       = "func-operation.configs.view"
	publicConfigPermissionCreate     = "func-operation.configs.create"
	publicConfigPermissionUpdate     = "func-operation.configs.update"
	publicConfigPermissionRollback   = "func-operation.configs.rollback"
	externalAPIPermissionView        = "func-operation.external-apis.view"
	externalAPIPermissionCreate      = "func-operation.external-apis.create"
	externalAPIPermissionUpdate      = "func-operation.external-apis.update"
)

// FunctionAuthorizer keeps authorization in the service boundary. The browser
// may hide controls, but every mutating RPC is checked again here.
type FunctionAuthorizer struct {
	system systemproto.SystemServiceClient
}

func NewFunctionAuthorizer(system systemproto.SystemServiceClient) *FunctionAuthorizer {
	return &FunctionAuthorizer{system: system}
}

func (a *FunctionAuthorizer) Require(ctx context.Context, menuCode string) error {
	allowed, err := a.Allowed(ctx, menuCode)
	if err != nil {
		return err
	}
	if !allowed {
		return errors.New("permission denied")
	}
	return nil
}

func (a *FunctionAuthorizer) Allowed(ctx context.Context, menuCode string) (bool, error) {
	if a == nil || a.system == nil {
		return false, errors.New("function authorization is not configured")
	}
	userID, err := authctx.RequireUserID(ctx)
	if err != nil || strings.TrimSpace(userID) == "" {
		return false, nil
	}
	resp, err := a.system.CheckMenuAccess(authctx.ForwardUserContext(ctx), &systemproto.CheckMenuAccessReq{
		UserId:   userID,
		MenuCode: strings.TrimSpace(menuCode),
	})
	if err != nil {
		return false, err
	}
	return resp.GetAllowed(), nil
}
