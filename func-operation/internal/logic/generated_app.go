package logic

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/dao"
	"github.com/gly-hub/ai-dandelion/func-operation/internal/model"
	"github.com/gly-hub/ai-dandelion/func-operation/internal/runtime/generatedapp"
	funcoperation "github.com/gly-hub/ai-dandelion/proto/func-operation"
	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
	"github.com/gly-hub/quickgo/logger"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GeneratedAppLogic struct {
	runtime           *generatedapp.Service
	previewRuntime    *generatedapp.Service
	functionDao       *dao.Function
	menuSync          *FunctionMenuSync
	generatedMenuDao  *dao.GeneratedFunctionMenu
	releaseLogic      *ReleaseLogic
	authorizer        *FunctionAuthorizer
	publicConfigLogic *PublicConfigLogic
	executionLogs     *dao.FunctionExecutionLog
}

func NewGeneratedAppLogic(runtime, previewRuntime *generatedapp.Service, functionDao *dao.Function, menuSync *FunctionMenuSync, generatedMenuDao *dao.GeneratedFunctionMenu, releaseLogic *ReleaseLogic, authorizer *FunctionAuthorizer, publicConfigLogic *PublicConfigLogic, executionLogs *dao.FunctionExecutionLog) *GeneratedAppLogic {
	return &GeneratedAppLogic{runtime: runtime, previewRuntime: previewRuntime, functionDao: functionDao, menuSync: menuSync, generatedMenuDao: generatedMenuDao, releaseLogic: releaseLogic, authorizer: authorizer, publicConfigLogic: publicConfigLogic, executionLogs: executionLogs}
}

func (g *GeneratedAppLogic) PreviewFrontend(ctx context.Context, functionID, requestedPath string) (string, error) {
	if err := g.authorizer.Require(ctx, functionPermissionEdit); err != nil {
		return "", err
	}
	functionItem, err := g.functionDao.Get(ctx, strings.TrimSpace(functionID))
	if err != nil {
		return "", err
	}
	if g.previewRuntime == nil || strings.TrimSpace(functionItem.GeneratedAppID) == "" {
		return "", errors.New("generated app preview is not ready")
	}
	if _, err := g.previewRuntime.LoadDraftApp(ctx, functionItem.GeneratedAppID); err != nil {
		return "", err
	}
	return g.previewRuntime.FrontendCode(functionItem.GeneratedAppID, requestedPath)
}

func (g *GeneratedAppLogic) PreviewBundle(ctx context.Context, functionID string) (*funcoperation.FrontendBundle, error) {
	if err := g.authorizer.Require(ctx, functionPermissionEdit); err != nil {
		return nil, err
	}
	functionItem, err := g.functionDao.Get(ctx, strings.TrimSpace(functionID))
	if err != nil {
		return nil, err
	}
	if g.previewRuntime == nil || strings.TrimSpace(functionItem.GeneratedAppID) == "" {
		return nil, errors.New("generated app preview is not ready")
	}
	if _, err := g.previewRuntime.LoadDraftApp(ctx, functionItem.GeneratedAppID); err != nil {
		return nil, err
	}
	version, entry, modules, err := g.previewRuntime.FrontendBundle(functionItem.GeneratedAppID)
	if err != nil {
		return nil, err
	}
	return &funcoperation.FrontendBundle{Version: version, Entry: entry, Modules: modules}, nil
}

func (g *GeneratedAppLogic) PreviewInvoke(ctx context.Context, functionID string, payload json.RawMessage) (generatedapp.InvokeResult, error) {
	return g.previewInvoke(ctx, functionID, payload, nil)
}

func (g *GeneratedAppLogic) PreviewInvokeWithObserver(ctx context.Context, functionID string, payload json.RawMessage, observer generatedapp.ExecutionLogObserver) (generatedapp.InvokeResult, error) {
	return g.previewInvoke(ctx, functionID, payload, observer)
}

func (g *GeneratedAppLogic) previewInvoke(ctx context.Context, functionID string, payload json.RawMessage, observer generatedapp.ExecutionLogObserver) (generatedapp.InvokeResult, error) {
	if err := g.authorizer.Require(ctx, functionPermissionEdit); err != nil {
		return generatedapp.InvokeResult{}, err
	}
	functionItem, err := g.functionDao.Get(ctx, strings.TrimSpace(functionID))
	if err != nil {
		return generatedapp.InvokeResult{}, err
	}
	if g.previewRuntime == nil || strings.TrimSpace(functionItem.GeneratedAppID) == "" {
		return generatedapp.InvokeResult{}, errors.New("generated app preview is not ready")
	}
	if _, err := g.previewRuntime.LoadDraftApp(ctx, functionItem.GeneratedAppID); err != nil {
		return generatedapp.InvokeResult{}, err
	}
	ctx = ensureExecutionRequestID(ctx)
	if configKeys, requested, requestErr := publicConfigRequestKeys(payload); requested {
		if requestErr != nil {
			return generatedapp.InvokeResult{}, requestErr
		}
		result, invokeErr := g.resolvePublicConfigInvoke(ctx, functionItem.GeneratedAppID, configKeys)
		return g.recordExecution(ctx, functionItem, "preview", payload, result, invokeErr), invokeErr
	}
	if err := g.previewRuntime.PrepareDraftDataModels(ctx, functionItem.GeneratedAppID); err != nil {
		return generatedapp.InvokeResult{}, err
	}
	actionKey := extractInvokeActionKey(payload)
	if actionKey != "" && actionKey != publicConfigResolveAction && generatedapp.ActionRequiresAuthorization(actionKey) {
		inspection, inspectErr := g.previewRuntime.InspectApp(functionItem.GeneratedAppID)
		if inspectErr != nil || !inspection.Exists || !actionDeclaredInManifest(inspection.Actions, actionKey) {
			return generatedapp.InvokeResult{}, errors.New("preview action is not declared in manifest")
		}
		ctx = generatedapp.WithAuthorizedAction(ctx, actionKey)
	}
	ctx = generatedapp.WithInvocationType(ctx, "preview")
	result, invokeErr := g.previewRuntime.InvokeWithObserver(ctx, functionItem.GeneratedAppID, payload, observer)
	return g.recordExecution(ctx, functionItem, "preview", payload, result, invokeErr), invokeErr
}

func (g *GeneratedAppLogic) CallCapability(ctx context.Context, callerAppID string, req generatedapp.CapabilityCallRequest) (any, error) {
	if strings.TrimSpace(callerAppID) == strings.TrimSpace(req.AppID) {
		return nil, errors.New("capability target must be a different function")
	}
	if _, err := g.releaseLogic.RequireActiveArtifact(ctx, req.AppID); err != nil {
		return nil, err
	}
	target, err := g.functionForApp(ctx, req.AppID)
	if err != nil {
		return nil, err
	}
	if err := g.requireRuntimeAccess(ctx, target, ""); err != nil {
		return nil, err
	}
	return g.runtime.RunDeclaredCapability(ctx, req.AppID, req.Capability, req.Params)
}

func (g *GeneratedAppLogic) ListGeneratedApps(ctx context.Context, req *funcoperation.ListGeneratedAppsReq) ([]*funcoperation.GeneratedApp, error) {
	if g.runtime == nil {
		return nil, errors.New("generated app runtime is not configured")
	}
	apps, err := g.runtime.ListApps(ctx)
	if err != nil {
		return nil, err
	}
	publishedIDs, err := g.releaseLogic.PublishedAppIDs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*funcoperation.GeneratedApp, 0, len(apps))
	for i := range apps {
		if _, ok := publishedIDs[apps[i].UUID]; !ok {
			continue
		}
		functionItem, functionErr := g.functionForApp(ctx, apps[i].UUID)
		if functionErr != nil {
			// Historical deletes may have left a release behind. It must not
			// make every valid published application unavailable.
			if errors.Is(functionErr, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, functionErr
		}
		allowed, accessErr := g.canAccessRuntime(ctx, functionItem, "")
		if accessErr != nil {
			return nil, accessErr
		}
		if !allowed {
			continue
		}
		out = append(out, modelGeneratedAppToProto(&apps[i]))
	}
	return out, nil
}

func (g *GeneratedAppLogic) ReloadGeneratedApps(ctx context.Context, req *funcoperation.ReloadGeneratedAppsReq) ([]*funcoperation.GeneratedApp, error) {
	if err := g.authorizer.Require(ctx, functionPermissionEdit); err != nil {
		return nil, err
	}
	if g.runtime == nil {
		return nil, errors.New("generated app runtime is not configured")
	}
	if err := g.releaseLogic.RevokeOrphanedPublished(ctx); err != nil {
		return nil, err
	}
	if err := g.releaseLogic.RestorePublished(ctx); err != nil {
		return nil, err
	}
	if err := g.SyncPublishedFunctionActions(ctx); err != nil {
		return nil, err
	}
	if _, err := g.CleanupGeneratedFunctionMenus(ctx, &funcoperation.CleanupGeneratedFunctionMenusReq{}); err != nil {
		return nil, err
	}
	return g.ListGeneratedApps(ctx, &funcoperation.ListGeneratedAppsReq{})
}

// SyncPublishedFunctionActions reconciles menus and action permissions for
// every published function. It is safe to run at service startup.
func (g *GeneratedAppLogic) SyncPublishedFunctionActions(ctx context.Context) error {
	return g.syncPublishedFunctionActions(ctx)
}

func (g *GeneratedAppLogic) CleanupGeneratedFunctionMenus(ctx context.Context, req *funcoperation.CleanupGeneratedFunctionMenusReq) (*funcoperation.CleanupGeneratedFunctionMenusResp, error) {
	if err := g.authorizer.Require(ctx, functionPermissionEdit); err != nil {
		return nil, err
	}
	if g == nil || g.functionDao == nil || g.generatedMenuDao == nil {
		return &funcoperation.CleanupGeneratedFunctionMenusResp{RemovedCount: 0}, nil
	}
	menus, err := g.generatedMenuDao.ListGeneratedFunctionMenus(ctx)
	if err != nil {
		return nil, err
	}
	removed := int32(0)
	for i := range menus {
		menu := menus[i]
		if menu.MenuType != 2 {
			continue
		}
		exists, err := g.functionDao.Exists(ctx, strings.TrimSpace(menu.SourceID))
		if err != nil {
			return nil, err
		}
		if exists {
			continue
		}
		for j := range menus {
			child := menus[j]
			if child.ParentID != menu.ID {
				continue
			}
			if err := g.generatedMenuDao.DeleteByID(ctx, child.ID); err != nil {
				return nil, err
			}
		}
		if err := g.generatedMenuDao.DeleteByID(ctx, menu.ID); err != nil {
			return nil, err
		}
		removed++
	}
	return &funcoperation.CleanupGeneratedFunctionMenusResp{RemovedCount: removed}, nil
}

func (g *GeneratedAppLogic) GetFrontend(ctx context.Context, req *funcoperation.GetGeneratedAppFrontendReq) (string, error) {
	if g.runtime == nil {
		return "", errors.New("generated app runtime is not configured")
	}
	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return "", errors.New("app id is required")
	}
	if _, err := g.releaseLogic.RequireActiveArtifact(ctx, id); err != nil {
		return "", err
	}
	functionItem, err := g.functionForApp(ctx, id)
	if err != nil {
		return "", err
	}
	if err := g.requireRuntimeAccess(ctx, functionItem, ""); err != nil {
		return "", err
	}
	return g.runtime.FrontendCode(id, req.GetPath())
}

func (g *GeneratedAppLogic) GetFrontendBundle(ctx context.Context, appID string) (*funcoperation.FrontendBundle, error) {
	if g.runtime == nil {
		return nil, errors.New("generated app runtime is not configured")
	}
	id := strings.TrimSpace(appID)
	if _, err := g.releaseLogic.RequireActiveArtifact(ctx, id); err != nil {
		return nil, err
	}
	functionItem, err := g.functionForApp(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := g.requireRuntimeAccess(ctx, functionItem, ""); err != nil {
		return nil, err
	}
	version, entry, modules, err := g.runtime.FrontendBundle(id)
	if err != nil {
		return nil, err
	}
	return &funcoperation.FrontendBundle{Version: version, Entry: entry, Modules: modules}, nil
}

func (g *GeneratedAppLogic) Invoke(ctx context.Context, req *funcoperation.InvokeGeneratedAppReq) (*funcoperation.InvokeGeneratedAppResp, error) {
	if g.runtime == nil {
		return nil, errors.New("generated app runtime is not configured")
	}
	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return nil, errors.New("app id is required")
	}
	if _, err := g.releaseLogic.RequireActiveArtifact(ctx, id); err != nil {
		return nil, err
	}
	functionItem, err := g.functionForApp(ctx, id)
	if err != nil {
		return nil, err
	}
	payload := json.RawMessage(req.GetPayload())
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	actionKey := extractInvokeActionKey(payload)
	checkAction := ""
	if actionKey != "" && actionKey != publicConfigResolveAction {
		if generatedapp.ActionRequiresAuthorization(actionKey) {
			if !g.runtime.DeclaresAction(id, actionKey) {
				return &funcoperation.InvokeGeneratedAppResp{
					AppId: id, ErrorCode: "FORBIDDEN", ErrorMessage: "功能动作未声明", Stage: "auth",
					Hint: "请发布包含该动作权限声明的功能版本",
				}, nil
			}
			checkAction = actionKey
		}
	}
	if err := g.requireRuntimeAccess(ctx, functionItem, checkAction); err != nil {
		return &funcoperation.InvokeGeneratedAppResp{
			AppId:        id,
			ErrorCode:    "FORBIDDEN",
			ErrorMessage: "当前用户无权访问此功能",
			Stage:        "auth",
			Hint:         runtimeAccessHint(checkAction),
		}, nil
	}
	ctx = ensureExecutionRequestID(ctx)
	if configKeys, requested, requestErr := publicConfigRequestKeys(payload); requested {
		if requestErr != nil {
			return &funcoperation.InvokeGeneratedAppResp{AppId: id, ErrorCode: "BAD_REQUEST", ErrorMessage: requestErr.Error(), Stage: "config"}, nil
		}
		resolved, resolveErr := g.resolvePublicConfigInvoke(ctx, id, configKeys)
		resolved = g.recordExecution(ctx, functionItem, "published", payload, resolved, resolveErr)
		if resolveErr != nil {
			return &funcoperation.InvokeGeneratedAppResp{AppId: id, ErrorCode: "CONFIG_UNAVAILABLE", ErrorMessage: resolveErr.Error(), Stage: "config", Hint: "请检查公共配置是否存在且格式正确", ExecutionLogId: resolved.ExecutionLogID}, nil
		}
		return &funcoperation.InvokeGeneratedAppResp{AppId: id, Response: resolved.Response, ExecutionLogId: resolved.ExecutionLogID}, nil
	}
	if checkAction != "" {
		ctx = generatedapp.WithAuthorizedAction(ctx, checkAction)
	}
	ctx = generatedapp.WithInvocationType(ctx, "published")
	result, invokeErr := g.runtime.Invoke(ctx, id, payload)
	result = g.recordExecution(ctx, functionItem, "published", payload, result, invokeErr)
	if invokeErr != nil {
		detail := classifyInvokeError(invokeErr)
		return &funcoperation.InvokeGeneratedAppResp{
			AppId:          id,
			ErrorCode:      detail.ErrorCode,
			ErrorMessage:   detail.ErrorMessage,
			Stage:          detail.Stage,
			Hint:           detail.Hint,
			ExecutionLogId: result.ExecutionLogID,
		}, nil
	}
	return &funcoperation.InvokeGeneratedAppResp{
		AppId:          result.AppID,
		Version:        result.Version,
		Export:         result.Export,
		Result:         result.Result,
		Response:       result.Response,
		Duration:       result.Duration,
		Runtime:        result.Runtime,
		ModuleLen:      int32(result.ModuleLen),
		BackendSource:  result.BackendSource,
		BackendModule:  result.BackendModule,
		ExecutionLogId: result.ExecutionLogID,
	}, nil
}

func (g *GeneratedAppLogic) ListExecutionLogs(ctx context.Context, req *funcoperation.ListFunctionExecutionLogsReq) ([]*funcoperation.FunctionExecutionLog, int64, error) {
	if err := g.authorizer.Require(ctx, functionPermissionEdit); err != nil {
		return nil, 0, err
	}
	functionID := strings.TrimSpace(req.GetFunctionId())
	if _, err := g.functionDao.Get(ctx, functionID); err != nil {
		return nil, 0, err
	}
	if g.executionLogs == nil {
		return nil, 0, errors.New("execution log store is not configured")
	}
	items, total, err := g.executionLogs.ListByFunctionID(ctx, functionID, dao.ExecutionLogFilter{
		Limit: int(req.GetLimit()), Page: int(req.GetPage()), Query: req.GetQuery(), Status: req.GetStatus(), InvocationType: req.GetInvocationType(),
		RequestID: req.GetRequestId(), StartTime: req.GetStartTime(), EndTime: req.GetEndTime(),
	})
	if err != nil {
		return nil, 0, err
	}
	out := make([]*funcoperation.FunctionExecutionLog, 0, len(items))
	for i := range items {
		out = append(out, executionLogProto(&items[i]))
	}
	return out, total, nil
}

func (g *GeneratedAppLogic) GetExecutionLog(ctx context.Context, functionID, id string) (*funcoperation.FunctionExecutionLog, error) {
	if err := g.authorizer.Require(ctx, functionPermissionEdit); err != nil {
		return nil, err
	}
	if _, err := g.functionDao.Get(ctx, strings.TrimSpace(functionID)); err != nil {
		return nil, err
	}
	if g.executionLogs == nil {
		return nil, errors.New("execution log store is not configured")
	}
	item, err := g.executionLogs.GetByFunctionID(ctx, strings.TrimSpace(functionID), strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	return executionLogProto(item), nil
}

func (g *GeneratedAppLogic) recordExecution(ctx context.Context, functionItem *model.Function, invocationType string, payload json.RawMessage, result generatedapp.InvokeResult, invokeErr error) generatedapp.InvokeResult {
	if g.executionLogs == nil || functionItem == nil {
		return result
	}
	logsJSON, marshalErr := json.Marshal(result.Logs)
	if marshalErr != nil {
		logsJSON = []byte("[]")
	}
	status, stage, errorCode, errorMessage := "succeeded", "", "", ""
	if invokeErr != nil {
		detail := classifyInvokeError(invokeErr)
		status, stage, errorCode, errorMessage = "failed", detail.Stage, detail.ErrorCode, detail.ErrorMessage
	}
	durationMS := int64(0)
	if duration, parseErr := time.ParseDuration(result.Duration); parseErr == nil {
		durationMS = duration.Milliseconds()
	}
	ctx = ensureExecutionRequestID(ctx)
	userID, _ := authctx.RequireUserID(ctx)
	item := &model.FunctionExecutionLog{
		UUID: uuid.NewString(), FunctionID: functionItem.UUID, AppID: firstNonEmpty(result.AppID, functionItem.GeneratedAppID), UserID: userID,
		RequestID:      logger.GetTraceID(ctx),
		InvocationType: invocationType, Version: result.Version, Status: status, Stage: stage, ErrorCode: errorCode, ErrorMessage: errorMessage,
		InputJSON: string(payload), OutputJSON: result.Response, LogsJSON: string(logsJSON), LogsTruncated: result.LogsTruncated,
		DurationMS: durationMS, CreatedAt: time.Now().UnixMicro(),
	}
	// Observability must not make a successful function invocation fail.
	if err := g.executionLogs.Create(ctx, item); err == nil {
		result.ExecutionLogID = item.UUID
	}
	return result
}

// QuickGo uses one trace ID for the HTTP request ID and the distributed trace.
// Direct/internal invocations still receive one so every execution can be
// correlated with server-side logs.
func ensureExecutionRequestID(ctx context.Context) context.Context {
	if logger.GetTraceID(ctx) != "" {
		return ctx
	}
	return logger.WithTraceID(ctx, logger.GenerateTraceID())
}

func executionLogProto(item *model.FunctionExecutionLog) *funcoperation.FunctionExecutionLog {
	if item == nil {
		return nil
	}
	var events []generatedapp.ExecutionLogEvent
	_ = json.Unmarshal([]byte(item.LogsJSON), &events)
	logs := make([]*funcoperation.ExecutionLogEvent, 0, len(events))
	for _, event := range events {
		logs = append(logs, &funcoperation.ExecutionLogEvent{Stream: event.Stream, Content: event.Content, Timestamp: event.Timestamp})
	}
	return &funcoperation.FunctionExecutionLog{
		Id: item.UUID, FunctionId: item.FunctionID, AppId: item.AppID, UserId: item.UserID, RequestId: item.RequestID, InvocationType: item.InvocationType,
		Version: item.Version, Status: item.Status, Stage: item.Stage, ErrorCode: item.ErrorCode, ErrorMessage: item.ErrorMessage,
		InputJson: item.InputJSON, OutputJson: item.OutputJSON, Logs: logs, LogsTruncated: item.LogsTruncated,
		DurationMs: item.DurationMS, CreatedAt: item.CreatedAt,
	}
}

const publicConfigResolveAction = "__platform.config.get"

func publicConfigRequestKeys(payload json.RawMessage) ([]string, bool, error) {
	if len(payload) == 0 {
		return nil, false, nil
	}
	var request struct {
		Action     string   `json:"action"`
		ConfigKeys []string `json:"configKeys"`
	}
	if err := json.Unmarshal(payload, &request); err != nil {
		return nil, false, nil
	}
	if strings.TrimSpace(request.Action) != publicConfigResolveAction {
		return nil, false, nil
	}
	if len(request.ConfigKeys) == 0 {
		return nil, true, errors.New("configKeys is required")
	}
	return request.ConfigKeys, true, nil
}

func (g *GeneratedAppLogic) resolvePublicConfigInvoke(ctx context.Context, appID string, keys []string) (generatedapp.InvokeResult, error) {
	if g == nil || g.publicConfigLogic == nil {
		return generatedapp.InvokeResult{}, errors.New("public config runtime is not configured")
	}
	allowed := g.runtime != nil && g.runtime.AllowsConfigKeys(appID, keys)
	if !allowed && g.previewRuntime != nil {
		allowed = g.previewRuntime.AllowsConfigKeys(appID, keys)
	}
	if !allowed {
		return generatedapp.InvokeResult{}, errors.New("requested public config is not declared by this function")
	}
	values, err := g.publicConfigLogic.ResolveValues(ctx, keys)
	if err != nil {
		return generatedapp.InvokeResult{}, err
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return generatedapp.InvokeResult{}, err
	}
	return generatedapp.InvokeResult{AppID: appID, Response: string(encoded), Runtime: "platform-config"}, nil
}

func (g *GeneratedAppLogic) functionForApp(ctx context.Context, appID string) (*model.Function, error) {
	if g == nil || g.functionDao == nil {
		return nil, nil
	}
	functionItem, err := g.functionDao.GetByGeneratedAppID(ctx, strings.TrimSpace(appID))
	if err != nil {
		return nil, err
	}
	return functionItem, nil
}

func (g *GeneratedAppLogic) requireRuntimeAccess(ctx context.Context, functionItem *model.Function, action string) error {
	allowed, err := g.canAccessRuntime(ctx, functionItem, action)
	if err != nil {
		return err
	}
	if !allowed {
		return errors.New("permission denied")
	}
	return nil
}

func (g *GeneratedAppLogic) canAccessRuntime(ctx context.Context, functionItem *model.Function, action string) (bool, error) {
	if functionItem == nil {
		return false, nil
	}
	if functionItem.Status != model.FunctionStatusPublished {
		return g.authorizer.Allowed(ctx, functionPermissionEdit)
	}
	if g.menuSync == nil {
		return false, errors.New("generated app access control is not configured")
	}
	userID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return false, nil
	}
	allowed, err := g.menuSync.CheckAccess(ctx, userID, functionItem.UUID, strings.TrimSpace(action))
	if err != nil {
		return false, err
	}
	return allowed, nil
}

func runtimeAccessHint(action string) string {
	if strings.TrimSpace(action) != "" {
		return "请联系管理员为当前角色分配对应按钮权限"
	}
	return "请联系管理员在角色管理中分配功能菜单权限"
}

func extractInvokeActionKey(payload json.RawMessage) string {
	if len(payload) == 0 {
		return ""
	}
	var request struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal(payload, &request); err != nil {
		return ""
	}
	return strings.TrimSpace(request.Action)
}

func actionDeclaredInManifest(actions []string, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	for _, action := range actions {
		if strings.TrimSpace(action) == target {
			return true
		}
	}
	return false
}

func modelGeneratedAppToProto(app *model.GeneratedApp) *funcoperation.GeneratedApp {
	if app == nil {
		return nil
	}
	return &funcoperation.GeneratedApp{
		Id:            app.UUID,
		Name:          app.Name,
		Version:       app.Version,
		Description:   app.Description,
		Export:        app.Export,
		FrontendEntry: app.FrontendEntry,
		BackendSource: app.BackendSource,
		BackendModule: app.BackendModule,
		CreatedAt:     app.CreatedAt,
		UpdatedAt:     app.UpdatedAt,
	}
}

func (g *GeneratedAppLogic) syncPublishedFunctionActions(ctx context.Context) error {
	if g == nil || g.functionDao == nil || g.menuSync == nil || g.runtime == nil {
		return nil
	}
	functions, err := g.functionDao.List(ctx)
	if err != nil {
		return err
	}
	for i := range functions {
		functionItem := functions[i]
		if functionItem.Status != model.FunctionStatusPublished {
			continue
		}
		if strings.TrimSpace(functionItem.GeneratedAppID) == "" || strings.TrimSpace(functionItem.MenuParentID) == "" {
			continue
		}
		inspection, inspectErr := g.runtime.InspectApp(strings.TrimSpace(functionItem.GeneratedAppID))
		if inspectErr != nil || !inspection.Exists {
			continue
		}
		menuID, syncErr := g.menuSync.SyncPublishedActions(
			ctx,
			functionItem.UUID,
			functionItem.Name,
			functionItem.MenuParentID,
			inspection.Actions,
		)
		if syncErr != nil {
			return syncErr
		}
		if menuID != "" && menuID != functionItem.MenuID {
			functionItem.MenuID = menuID
			functionItem.UpdatedAt = nowUnixMicro()
			if err := g.functionDao.Update(ctx, &functionItem); err != nil {
				return err
			}
		}
	}
	return nil
}
