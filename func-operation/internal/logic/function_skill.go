package logic

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/dao"
	"github.com/gly-hub/ai-dandelion/func-operation/internal/model"
	"github.com/gly-hub/ai-dandelion/func-operation/internal/runtime/generatedapp"
	funcoperation "github.com/gly-hub/ai-dandelion/proto/func-operation"
	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const functionSkillGrantTTL = 15 * time.Minute
const functionSkillApprovalTTL = 5 * time.Minute

type FunctionSkillLogic struct {
	skills     *dao.FunctionSkill
	releases   *dao.FunctionSkillRelease
	grants     *dao.FunctionSkillGrant
	approvals  *dao.FunctionSkillApproval
	executions *dao.FunctionSkillExecution
	functions  *dao.Function
	apps       *GeneratedAppLogic
	authorizer *FunctionAuthorizer
}

func NewFunctionSkillLogic(skills *dao.FunctionSkill, releases *dao.FunctionSkillRelease, grants *dao.FunctionSkillGrant, approvals *dao.FunctionSkillApproval, executions *dao.FunctionSkillExecution, functions *dao.Function, apps *GeneratedAppLogic, authorizer *FunctionAuthorizer) *FunctionSkillLogic {
	return &FunctionSkillLogic{skills: skills, releases: releases, grants: grants, approvals: approvals, executions: executions, functions: functions, apps: apps, authorizer: authorizer}
}

func (l *FunctionSkillLogic) SyncPublished(ctx context.Context, function *model.Function, release *model.FunctionRelease) error {
	if l == nil || function == nil || release == nil {
		return nil
	}
	contract, actions, err := generatedapp.ParseAgentSkillManifest(release.ManifestJSON)
	if err != nil || contract == nil {
		_ = l.RevokeFunction(ctx, function.UUID)
		return nil
	}
	if err := generatedapp.ValidateAgentSkillContract(contract, actions); err != nil {
		_ = l.RevokeFunction(ctx, function.UUID)
		return nil
	}
	now := nowUnixMicro()
	item, getErr := l.skills.GetByFunctionID(ctx, function.UUID)
	if getErr != nil && !errors.Is(getErr, gorm.ErrRecordNotFound) {
		return getErr
	}
	if item == nil {
		item = &model.FunctionSkill{UUID: uuid.NewString(), FunctionID: function.UUID, Status: model.FunctionSkillStatusEnabled, CreatedAt: now}
	}
	if strings.TrimSpace(item.ToolPrefix) != "" {
		contract.ToolPrefix = item.ToolPrefix
	}
	item.Name, item.Description, item.ToolPrefix, item.UpdatedAt = contract.Name, contract.Description, contract.ToolPrefix, now
	if item.Status == "" {
		item.Status = model.FunctionSkillStatusEnabled
	}
	if err := l.skills.Upsert(ctx, item); err != nil {
		return err
	}
	// The generated UUID is preserved by the function_id upsert path. Reload it
	// so the release snapshot always references the actual stable skill identity.
	item, err = l.skills.GetByFunctionID(ctx, function.UUID)
	if err != nil {
		return err
	}
	contractRaw, err := json.Marshal(contract)
	if err != nil {
		return err
	}
	if _, err := l.releases.ActiveByFunctionRelease(ctx, release.UUID); err == nil {
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return l.releases.Create(ctx, &model.FunctionSkillRelease{UUID: uuid.NewString(), SkillID: item.UUID, FunctionID: function.UUID, FunctionReleaseID: release.UUID, AppID: release.AppID, ArtifactSHA256: release.ArtifactSHA256, ContractJSON: string(contractRaw), Status: model.FunctionSkillReleaseStatusActive, CreatedAt: now, UpdatedAt: now})
}

func (l *FunctionSkillLogic) RevokeFunction(ctx context.Context, functionID string) error {
	if l == nil {
		return nil
	}
	functionID = strings.TrimSpace(functionID)
	now := nowUnixMicro()
	if err := l.releases.RevokeByFunctionID(ctx, functionID, now); err != nil {
		return err
	}
	skill, err := l.skills.GetByFunctionID(ctx, functionID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return l.grants.RevokeBySkillID(ctx, skill.UUID, now)
}

func (l *FunctionSkillLogic) List(ctx context.Context, _ *funcoperation.ListFunctionSkillsReq) ([]*funcoperation.FunctionSkill, error) {
	items, err := l.skills.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*funcoperation.FunctionSkill, 0, len(items))
	for i := range items {
		function, err := l.functions.Get(ctx, items[i].FunctionID)
		if err != nil {
			continue
		}
		allowed, err := l.apps.canAccessRuntime(ctx, function, "")
		if err != nil || !allowed {
			continue
		}
		release, err := l.releases.ActiveBySkill(ctx, items[i].UUID)
		if err != nil {
			continue
		}
		contract, err := generatedapp.ParseAgentSkillContract(release.ContractJSON)
		if err != nil {
			continue
		}
		out = append(out, functionSkillToProto(&items[i], release, contract))
	}
	return out, nil
}

func (l *FunctionSkillLogic) SetEnabled(ctx context.Context, req *funcoperation.SetFunctionSkillEnabledReq) (*funcoperation.FunctionSkill, error) {
	if err := l.authorizer.Require(ctx, functionPermissionSkillConfigure); err != nil {
		return nil, err
	}
	functionID := strings.TrimSpace(req.GetFunctionId())
	if functionID == "" {
		return nil, errors.New("function id is required")
	}
	if _, err := l.functions.Get(ctx, functionID); err != nil {
		return nil, err
	}
	status := model.FunctionSkillStatusDisabled
	if req.GetEnabled() {
		status = model.FunctionSkillStatusEnabled
	}
	if err := l.skills.SetStatus(ctx, functionID, status, nowUnixMicro()); err != nil {
		return nil, err
	}
	item, err := l.skills.GetByFunctionID(ctx, functionID)
	if err != nil {
		return nil, err
	}
	if !req.GetEnabled() {
		if err := l.grants.RevokeBySkillID(ctx, item.UUID, nowUnixMicro()); err != nil {
			return nil, err
		}
	}
	release, _ := l.releases.ActiveBySkill(ctx, item.UUID)
	var contract *generatedapp.AgentSkillContract
	if release != nil {
		contract, _ = generatedapp.ParseAgentSkillContract(release.ContractJSON)
	}
	return functionSkillToProto(item, release, contract), nil
}

func (l *FunctionSkillLogic) IssueGrant(ctx context.Context, req *funcoperation.IssueFunctionSkillGrantReq) (string, int64, error) {
	userID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return "", 0, err
	}
	sessionID := strings.TrimSpace(req.GetSessionId())
	if sessionID == "" {
		return "", 0, errors.New("session id is required")
	}
	ids := uniqueFunctionSkillIDs(req.GetSkillIds())
	if len(ids) == 0 {
		return "", 0, errors.New("function skills are required")
	}
	for _, skillID := range ids {
		if _, _, err := l.resolveSkillForUser(ctx, skillID, userID); err != nil {
			return "", 0, err
		}
	}
	token, err := newFunctionSkillToken()
	if err != nil {
		return "", 0, err
	}
	now, expires := nowUnixMicro(), time.Now().Add(functionSkillGrantTTL).UnixMicro()
	rawIDs, _ := json.Marshal(ids)
	if err := l.grants.Create(ctx, &model.FunctionSkillGrant{UUID: uuid.NewString(), TokenHash: hashFunctionSkillToken(token), UserID: userID, SessionID: sessionID, SkillIDs: string(rawIDs), ExpiresAt: expires, CreatedAt: now}); err != nil {
		return "", 0, err
	}
	return token, expires, nil
}

func (l *FunctionSkillLogic) GetTools(ctx context.Context, req *funcoperation.GetFunctionSkillToolsReq) ([]*funcoperation.FunctionSkillTool, error) {
	grant, ids, err := l.resolveGrant(ctx, req.GetGrantToken())
	if err != nil {
		return nil, err
	}
	tools := make([]*funcoperation.FunctionSkillTool, 0)
	for _, skillID := range ids {
		skill, release, err := l.resolveSkillForUser(ctx, skillID, grant.UserID)
		if err != nil {
			continue
		}
		contract, err := generatedapp.ParseAgentSkillContract(release.ContractJSON)
		if err != nil {
			continue
		}
		for _, operation := range contract.Operations {
			if operation.Effect != "read" {
				function, getErr := l.functions.Get(ctx, skill.FunctionID)
				if getErr != nil {
					continue
				}
				allowed, accessErr := l.apps.canAccessRuntime(authctx.ContextWithUser(context.Background(), authctx.User{ID: grant.UserID}), function, operation.Action)
				if accessErr != nil || !allowed {
					continue
				}
			}
			schema, schemaErr := functionSkillInputSchema(operation)
			if schemaErr != nil {
				continue
			}
			tools = append(tools, &funcoperation.FunctionSkillTool{Name: functionSkillToolName(contract.ToolPrefix, operation.Key), Description: firstNonEmpty(operation.Description, contract.Description, contract.Name), InputSchemaJson: schema, AutoExecute: operation.AutoExecute, Effect: operation.Effect})
		}
	}
	return tools, nil
}

func (l *FunctionSkillLogic) CreateApproval(ctx context.Context, req *funcoperation.CreateFunctionSkillApprovalReq) (string, error) {
	grant, _, err := l.resolveGrant(ctx, req.GetGrantToken())
	if err != nil {
		return "", err
	}
	resolved, err := l.resolveOperation(ctx, grant, req.GetToolName())
	if err != nil {
		return "", err
	}
	if !requiresFunctionSkillApproval(resolved.operation) {
		return "", errors.New("tool does not require approval")
	}
	input, err := normalizeFunctionSkillInput(req.GetInputJson(), resolved.operation)
	if err != nil {
		return "", err
	}
	token, err := newFunctionSkillToken()
	if err != nil {
		return "", err
	}
	now := nowUnixMicro()
	if err := l.approvals.Create(ctx, &model.FunctionSkillApproval{UUID: uuid.NewString(), TokenHash: hashFunctionSkillToken(token), GrantID: grant.UUID, ToolName: strings.TrimSpace(req.GetToolName()), ToolUseID: strings.TrimSpace(req.GetToolUseId()), InputHash: hashFunctionSkillInput(input), ExpiresAt: time.Now().Add(functionSkillApprovalTTL).UnixMicro(), CreatedAt: now}); err != nil {
		return "", err
	}
	return token, nil
}

func (l *FunctionSkillLogic) Execute(ctx context.Context, req *funcoperation.ExecuteFunctionSkillReq) (*funcoperation.ExecuteFunctionSkillResp, error) {
	grant, _, err := l.resolveGrant(ctx, req.GetGrantToken())
	if err != nil {
		return nil, err
	}
	resolved, err := l.resolveOperation(ctx, grant, req.GetToolName())
	if err != nil {
		return nil, err
	}
	input, err := normalizeFunctionSkillInput(req.GetInputJson(), resolved.operation)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetToolUseId()) == "" {
		return nil, errors.New("tool use id is required")
	}
	if cached, getErr := l.executions.GetByIdempotency(ctx, resolved.release.UUID, req.GetToolUseId()); getErr == nil {
		return executionToResponse(cached), nil
	} else if !errors.Is(getErr, gorm.ErrRecordNotFound) {
		return nil, getErr
	}
	if requiresFunctionSkillApproval(resolved.operation) {
		if err := l.consumeApproval(ctx, req.GetApprovalToken(), grant.UUID, req.GetToolName(), req.GetToolUseId(), input); err != nil {
			return nil, err
		}
	}
	function, err := l.functions.Get(ctx, resolved.skill.FunctionID)
	if err != nil {
		return nil, err
	}
	userCtx := authctx.ContextWithUser(context.Background(), authctx.User{ID: grant.UserID})
	action := ""
	if resolved.operation.Effect != "read" {
		action = resolved.operation.Action
	}
	if err := l.apps.requireRuntimeAccess(userCtx, function, action); err != nil {
		return nil, err
	}
	payload := make(map[string]any, len(input)+1)
	payload["action"] = resolved.operation.Action
	for key, value := range input {
		payload[key] = value
	}
	payloadRaw, _ := json.Marshal(payload)
	invokeOut, invokeErr := l.apps.Invoke(userCtx, &funcoperation.InvokeGeneratedAppReq{Id: resolved.release.AppID, Payload: string(payloadRaw), UserId: grant.UserID})
	response := &funcoperation.ExecuteFunctionSkillResp{}
	if invokeErr != nil {
		response.IsError, response.ErrorMessage = true, invokeErr.Error()
	} else if invokeOut.GetErrorCode() != "" {
		response.IsError, response.ErrorMessage = true, invokeOut.GetErrorMessage()
	} else {
		response.ResultJson = firstNonEmpty(invokeOut.GetResponse(), "{}")
	}
	resultJSON := sanitizeFunctionSkillJSON(response.GetResultJson())
	if response.GetIsError() {
		resultJSON = ""
	}
	now := nowUnixMicro()
	status := "succeeded"
	if response.GetIsError() {
		status = "failed"
	}
	execution := &model.FunctionSkillExecution{UUID: uuid.NewString(), FunctionID: resolved.skill.FunctionID, SkillReleaseID: resolved.release.UUID, UserID: grant.UserID, SessionID: grant.SessionID, ToolName: req.GetToolName(), ToolUseID: req.GetToolUseId(), InputJSON: string(mustJSON(sanitizeFunctionSkillMap(input))), ResultJSON: resultJSON, Status: status, ErrorMessage: response.GetErrorMessage(), CreatedAt: now, UpdatedAt: now}
	if err := l.executions.Create(ctx, execution); err != nil {
		if cached, getErr := l.executions.GetByIdempotency(ctx, resolved.release.UUID, req.GetToolUseId()); getErr == nil {
			return executionToResponse(cached), nil
		}
		return nil, err
	}
	return response, nil
}

func (l *FunctionSkillLogic) ListExecutions(ctx context.Context, req *funcoperation.ListFunctionSkillExecutionsReq) ([]*funcoperation.FunctionSkillExecution, error) {
	if err := l.authorizer.Require(ctx, functionPermissionSkillConfigure); err != nil {
		return nil, err
	}
	items, err := l.executions.ListByFunctionID(ctx, strings.TrimSpace(req.GetFunctionId()), int(req.GetLimit()))
	if err != nil {
		return nil, err
	}
	out := make([]*funcoperation.FunctionSkillExecution, 0, len(items))
	for i := range items {
		out = append(out, &funcoperation.FunctionSkillExecution{Id: items[i].UUID, FunctionId: items[i].FunctionID, SkillReleaseId: items[i].SkillReleaseID, ToolName: items[i].ToolName, Status: items[i].Status, ErrorMessage: items[i].ErrorMessage, CreatedAt: items[i].CreatedAt})
	}
	return out, nil
}

type resolvedFunctionSkillOperation struct {
	skill     *model.FunctionSkill
	release   *model.FunctionSkillRelease
	operation generatedapp.AgentSkillOperation
}

func (l *FunctionSkillLogic) resolveOperation(ctx context.Context, grant *model.FunctionSkillGrant, toolName string) (*resolvedFunctionSkillOperation, error) {
	_, ids, err := l.resolveGrantByModel(grant)
	if err != nil {
		return nil, err
	}
	for _, skillID := range ids {
		skill, release, err := l.resolveSkillForUser(ctx, skillID, grant.UserID)
		if err != nil {
			continue
		}
		contract, err := generatedapp.ParseAgentSkillContract(release.ContractJSON)
		if err != nil {
			continue
		}
		for _, operation := range contract.Operations {
			if functionSkillToolName(contract.ToolPrefix, operation.Key) == strings.TrimSpace(toolName) {
				return &resolvedFunctionSkillOperation{skill: skill, release: release, operation: operation}, nil
			}
		}
	}
	return nil, errors.New("function skill tool is not available")
}

func (l *FunctionSkillLogic) resolveSkillForUser(ctx context.Context, skillID, userID string) (*model.FunctionSkill, *model.FunctionSkillRelease, error) {
	skill, err := l.skills.Get(ctx, strings.TrimSpace(skillID))
	if err != nil {
		return nil, nil, err
	}
	if skill.Status != model.FunctionSkillStatusEnabled {
		return nil, nil, errors.New("function skill is disabled")
	}
	function, err := l.functions.Get(ctx, skill.FunctionID)
	if err != nil {
		return nil, nil, err
	}
	allowed, err := l.apps.canAccessRuntime(authctx.ContextWithUser(context.Background(), authctx.User{ID: userID}), function, "")
	if err != nil {
		return nil, nil, err
	}
	if !allowed {
		return nil, nil, errors.New("function skill access denied")
	}
	release, err := l.releases.ActiveBySkill(ctx, skill.UUID)
	if err != nil {
		return nil, nil, err
	}
	return skill, release, nil
}

func (l *FunctionSkillLogic) resolveGrant(ctx context.Context, token string) (*model.FunctionSkillGrant, []string, error) {
	grant, err := l.grants.GetByTokenHash(ctx, hashFunctionSkillToken(token))
	if err != nil {
		return nil, nil, err
	}
	return l.resolveGrantByModel(grant)
}
func (l *FunctionSkillLogic) resolveGrantByModel(grant *model.FunctionSkillGrant) (*model.FunctionSkillGrant, []string, error) {
	if grant == nil || grant.RevokedAt != 0 || grant.ExpiresAt <= time.Now().UnixMicro() {
		return nil, nil, errors.New("function skill grant has expired")
	}
	var ids []string
	if err := json.Unmarshal([]byte(grant.SkillIDs), &ids); err != nil {
		return nil, nil, err
	}
	return grant, uniqueFunctionSkillIDs(ids), nil
}

func (l *FunctionSkillLogic) consumeApproval(ctx context.Context, token, grantID, toolName, toolUseID string, input map[string]any) error {
	if strings.TrimSpace(token) == "" {
		return errors.New("function skill operation requires confirmation")
	}
	approval, err := l.approvals.Consume(ctx, hashFunctionSkillToken(token), nowUnixMicro())
	if err != nil {
		return errors.New("function skill approval is invalid")
	}
	if approval.ExpiresAt <= time.Now().UnixMicro() || approval.GrantID != grantID || approval.ToolName != strings.TrimSpace(toolName) || approval.ToolUseID != strings.TrimSpace(toolUseID) || approval.InputHash != hashFunctionSkillInput(input) {
		return errors.New("function skill approval does not match this tool call")
	}
	return nil
}

func functionSkillToProto(skill *model.FunctionSkill, release *model.FunctionSkillRelease, contract *generatedapp.AgentSkillContract) *funcoperation.FunctionSkill {
	out := &funcoperation.FunctionSkill{Id: skill.UUID, FunctionId: skill.FunctionID, Name: skill.Name, Description: skill.Description, ToolPrefix: skill.ToolPrefix, Enabled: skill.Status == model.FunctionSkillStatusEnabled, UpdatedAt: skill.UpdatedAt}
	if release != nil {
		out.ReleaseId = release.UUID
	}
	if contract != nil {
		for _, op := range contract.Operations {
			out.Operations = append(out.Operations, operationToProto(op))
		}
	}
	return out
}
func operationToProto(op generatedapp.AgentSkillOperation) *funcoperation.FunctionSkillOperation {
	out := &funcoperation.FunctionSkillOperation{Key: op.Key, Action: op.Action, Effect: op.Effect, Description: op.Description, AutoExecute: op.AutoExecute}
	for _, field := range op.Fields {
		out.Fields = append(out.Fields, &funcoperation.FunctionSkillField{Key: field.Key, Label: field.Label, Type: field.Type, Required: field.Required, EnumValues: field.EnumValues, Description: field.Description})
	}
	return out
}
func functionSkillToolName(prefix, key string) string {
	return strings.TrimSpace(prefix) + "__" + strings.TrimSpace(key)
}
func requiresFunctionSkillApproval(op generatedapp.AgentSkillOperation) bool {
	return op.Effect != "read" && !(op.Effect == "create" && op.AutoExecute)
}
func functionSkillInputSchema(op generatedapp.AgentSkillOperation) (string, error) {
	properties := map[string]any{}
	required := make([]string, 0)
	for _, field := range op.Fields {
		item := map[string]any{"type": field.Type}
		if field.Type == "enum" {
			item["type"] = "string"
			item["enum"] = field.EnumValues
		}
		if field.Description != "" {
			item["description"] = field.Description
		}
		properties[field.Key] = item
		if field.Required {
			required = append(required, field.Key)
		}
	}
	raw, err := json.Marshal(map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false})
	return string(raw), err
}
func normalizeFunctionSkillInput(raw string, op generatedapp.AgentSkillOperation) (map[string]any, error) {
	input := map[string]any{}
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &input); err != nil {
			return nil, errors.New("tool input must be a JSON object")
		}
	}
	allowed := map[string]generatedapp.AgentSkillField{}
	for _, field := range op.Fields {
		allowed[field.Key] = field
		if field.Required {
			if _, ok := input[field.Key]; !ok {
				return nil, fmt.Errorf("%s is required", field.Key)
			}
		}
	}
	for key, value := range input {
		field, ok := allowed[key]
		if !ok {
			return nil, fmt.Errorf("unexpected tool field %q", key)
		}
		if err := validateFunctionSkillValue(value, field); err != nil {
			return nil, err
		}
	}
	return input, nil
}
func validateFunctionSkillValue(value any, field generatedapp.AgentSkillField) error {
	switch field.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("%s must be a string", field.Key)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%s must be a boolean", field.Key)
		}
	case "integer":
		number, ok := value.(float64)
		if !ok || number != float64(int64(number)) {
			return fmt.Errorf("%s must be an integer", field.Key)
		}
	case "number":
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("%s must be a number", field.Key)
		}
	case "enum":
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("%s must be an enum value", field.Key)
		}
		found := false
		for _, item := range field.EnumValues {
			if text == item {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("%s has an unsupported value", field.Key)
		}
	}
	return nil
}
func executionToResponse(item *model.FunctionSkillExecution) *funcoperation.ExecuteFunctionSkillResp {
	return &funcoperation.ExecuteFunctionSkillResp{ResultJson: item.ResultJSON, IsError: item.Status != "succeeded", ErrorMessage: item.ErrorMessage}
}
func uniqueFunctionSkillIDs(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
func newFunctionSkillToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
func hashFunctionSkillToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}
func hashFunctionSkillInput(input map[string]any) string {
	return hashFunctionSkillToken(string(mustJSON(input)))
}

func sanitizeFunctionSkillJSON(raw string) string {
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return raw
	}
	clean := sanitizeFunctionSkillValue(value)
	data, err := json.Marshal(clean)
	if err != nil {
		return raw
	}
	return string(data)
}

func sanitizeFunctionSkillMap(input map[string]any) map[string]any {
	clean, _ := sanitizeFunctionSkillValue(input).(map[string]any)
	if clean == nil {
		return map[string]any{}
	}
	return clean
}

func sanitizeFunctionSkillValue(value any) any {
	switch item := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(item))
		for key, child := range item {
			lower := strings.ToLower(key)
			if strings.Contains(lower, "token") || strings.Contains(lower, "password") || strings.Contains(lower, "secret") || strings.Contains(lower, "apikey") {
				out[key] = "[REDACTED]"
				continue
			}
			out[key] = sanitizeFunctionSkillValue(child)
		}
		return out
	case []any:
		out := make([]any, len(item))
		for i, child := range item {
			out[i] = sanitizeFunctionSkillValue(child)
		}
		return out
	default:
		return value
	}
}
func mustJSON(value any) []byte { raw, _ := json.Marshal(value); return raw }
