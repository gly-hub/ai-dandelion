package logic

import (
	"context"
	"encoding/json"
	"strings"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
	claudeagentsdk "github.com/gly-hub/claude-agent-sdk-go"
)

const navigationMCPServerID = "navigation"

// NavigationRuntime exposes permission-filtered application navigation to the
// model. It returns semantic target references; the browser remains the only
// component that resolves those references into concrete routes.
type NavigationRuntime struct {
	client systemproto.SystemServiceClient
}

func NewNavigationRuntime(client systemproto.SystemServiceClient) *NavigationRuntime {
	return &NavigationRuntime{client: client}
}

func (r *NavigationRuntime) Server(ctx context.Context) (claudeagentsdk.SDKMCPServerConfig, error) {
	if r == nil || r.client == nil {
		return claudeagentsdk.SDKMCPServerConfig{}, nil
	}
	callCtx := authctx.ForwardUserContext(context.WithoutCancel(ctx))
	tools := []claudeagentsdk.MCPTool{
		claudeagentsdk.NewMCPTool(
			"list_navigation_targets",
			"List pages the current user can access. Use this before navigation when the target is described by a natural-language name. Choose a returned target id; never invent ids or URLs.",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query":  map[string]any{"type": "string", "description": "A page name, alias, or business description to search for."},
					"module": map[string]any{"type": "string", "description": "Optional module filter, such as system or func-operation."},
					"limit":  map[string]any{"type": "integer", "minimum": 1, "maximum": 20},
				},
			},
			func(input map[string]any) (claudeagentsdk.MCPToolResult, error) {
				return r.listTargets(callCtx, input)
			},
		),
		claudeagentsdk.NewMCPTool(
			"navigate_to_target",
			"Request browser navigation to an exact target id returned by list_navigation_targets. This cannot navigate to an unlisted URL.",
			map[string]any{
				"type":     "object",
				"required": []string{"targetId"},
				"properties": map[string]any{
					"targetId": map[string]any{"type": "string"},
				},
			},
			func(input map[string]any) (claudeagentsdk.MCPToolResult, error) {
				return r.navigate(callCtx, input)
			},
		),
	}
	return claudeagentsdk.CreateSDKMCPServer(navigationMCPServerID, "1.0.0", tools), nil
}

func (r *NavigationRuntime) listTargets(ctx context.Context, input map[string]any) (claudeagentsdk.MCPToolResult, error) {
	if r == nil || r.client == nil {
		return navigationResult(map[string]any{"error": "navigation service is unavailable"}, true), nil
	}
	userID, _ := authctx.RequireUserID(ctx)
	module := stringInput(input, "module")
	query := strings.ToLower(strings.TrimSpace(stringInput(input, "query")))
	limit := intInput(input, "limit", 10)
	if limit < 1 {
		limit = 1
	}
	if limit > 20 {
		limit = 20
	}
	resp, err := r.client.GetNavMenus(ctx, &systemproto.GetNavMenusReq{Module: module, UserId: userID})
	if err != nil {
		return navigationResult(map[string]any{"error": err.Error()}, true), nil
	}
	targets := flattenNavigationTargets(resp.GetMenus())
	if query != "" {
		filtered := targets[:0]
		for _, target := range targets {
			if navigationTargetMatches(target, query) {
				filtered = append(filtered, target)
			}
		}
		targets = filtered
	}
	if len(targets) > limit {
		targets = targets[:limit]
	}
	return navigationResult(map[string]any{"targets": targets, "count": len(targets)}, false), nil
}

func (r *NavigationRuntime) navigate(ctx context.Context, input map[string]any) (claudeagentsdk.MCPToolResult, error) {
	if r == nil || r.client == nil {
		return navigationResult(map[string]any{"error": "navigation service is unavailable"}, true), nil
	}
	targetID := strings.TrimSpace(stringInput(input, "targetId"))
	if targetID == "" {
		return navigationResult(map[string]any{"error": "targetId is required"}, true), nil
	}
	userID, _ := authctx.RequireUserID(ctx)
	resp, err := r.client.GetNavMenus(ctx, &systemproto.GetNavMenusReq{UserId: userID})
	if err != nil {
		return navigationResult(map[string]any{"error": err.Error()}, true), nil
	}
	var target map[string]any
	for _, item := range flattenNavigationTargets(resp.GetMenus()) {
		if item["id"] == targetID {
			target = item
			break
		}
	}
	if target == nil {
		return navigationResult(map[string]any{"error": "navigation target is unavailable or not permitted", "targetId": targetID}, true), nil
	}
	uiTarget := map[string]any{
		"targetId":   targetID,
		"module":     target["module"],
		"viewKey":    target["viewKey"],
		"sourceType": target["sourceType"],
		"sourceId":   target["sourceId"],
	}
	return navigationResult(map[string]any{
		"status": "accepted",
		"uiAction": map[string]any{
			"action": "navigate",
			"target": uiTarget,
		},
	}, false), nil
}

func flattenNavigationTargets(menus []*systemproto.Menu) []map[string]any {
	targets := make([]map[string]any, 0)
	var walk func([]*systemproto.Menu)
	walk = func(items []*systemproto.Menu) {
		for _, item := range items {
			if item == nil {
				continue
			}
			// Menu type 2 is the existing page/menu type. Directories and
			// buttons are intentionally excluded from navigation targets.
			if item.GetMenuType() == 2 && strings.TrimSpace(item.GetId()) != "" {
				targets = append(targets, map[string]any{
					"id":          item.GetId(),
					"label":       item.GetName(),
					"description": item.GetRemark(),
					"module":      item.GetModule(),
					"viewKey":     item.GetViewKey(),
					"code":        item.GetCode(),
					"sourceType":  item.GetSourceType(),
					"sourceId":    item.GetSourceId(),
				})
			}
			walk(item.GetChildren())
		}
	}
	walk(menus)
	return targets
}

func navigationTargetMatches(target map[string]any, query string) bool {
	for _, key := range []string{"label", "description", "module", "viewKey", "code", "sourceId"} {
		if strings.Contains(strings.ToLower(strings.TrimSpace(navStringValue(target[key]))), query) {
			return true
		}
	}
	return false
}

func navigationResult(value map[string]any, isError bool) claudeagentsdk.MCPToolResult {
	data, err := json.Marshal(value)
	if err != nil {
		data = []byte(`{"error":"encode navigation result failed"}`)
		isError = true
	}
	return claudeagentsdk.MCPToolResult{Content: []claudeagentsdk.MCPToolContent{{Type: "text", Text: string(data)}}, IsError: isError}
}

func stringInput(input map[string]any, key string) string {
	value, _ := input[key].(string)
	return strings.TrimSpace(value)
}

func intInput(input map[string]any, key string, fallback int) int {
	value, ok := input[key].(float64)
	if !ok {
		return fallback
	}
	return int(value)
}

func navStringValue(value any) string {
	text, _ := value.(string)
	return text
}
