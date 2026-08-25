package generatedapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/dao"
	"github.com/tetratelabs/wazero/api"
)

type appIDContextKey struct{}
type invocationIDContextKey struct{}
type privilegedActionContextKey struct{}
type invocationTypeContextKey struct{}
type invocationActionContextKey struct{}
type executionLogCollectorContextKey struct{}

// WithAuthorizedAction marks an invocation whose action has already passed
// the generated-function menu check in the service layer. Guest code cannot
// create this server-side context value.
func WithAuthorizedAction(ctx context.Context, action string) context.Context {
	return context.WithValue(ctx, privilegedActionContextKey{}, strings.TrimSpace(action))
}

// WithInvocationType annotates host-side execution logs without granting any
// capability to guest WASM code.
func WithInvocationType(ctx context.Context, invocationType string) context.Context {
	return context.WithValue(ctx, invocationTypeContextKey{}, strings.TrimSpace(invocationType))
}

func requireAuthorizedAction(ctx context.Context) error {
	if action, ok := ctx.Value(privilegedActionContextKey{}).(string); ok && strings.TrimSpace(action) != "" {
		return nil
	}
	return errors.New("privileged platform capability requires an authorized action")
}

func (s *Service) instantiatePlatformHost(ctx context.Context) error {
	_, err := s.runtime.NewHostModuleBuilder("platform").
		NewFunctionBuilder().WithFunc(s.hostDataList).Export("data_list").
		NewFunctionBuilder().WithFunc(s.hostDataGet).Export("data_get").
		NewFunctionBuilder().WithFunc(s.hostDataCreate).Export("data_create").
		NewFunctionBuilder().WithFunc(s.hostDataUpdate).Export("data_update").
		NewFunctionBuilder().WithFunc(s.hostDataDelete).Export("data_delete").
		NewFunctionBuilder().WithFunc(s.hostDataJoinQuery).Export("data_join_query").
		NewFunctionBuilder().WithFunc(s.hostDataRunQuery).Export("data_run_query").
		NewFunctionBuilder().WithFunc(s.hostCallCapability).Export("call_capability").
		NewFunctionBuilder().WithFunc(s.hostExternalAPICall).Export("external_api_call").
		NewFunctionBuilder().WithFunc(s.hostLog).Export("log").
		NewFunctionBuilder().WithFunc(s.hostResultLen).Export("result_len").
		NewFunctionBuilder().WithFunc(s.hostResultRead).Export("result_read").
		NewFunctionBuilder().WithFunc(s.hostResultStore).Export("result_store").
		Instantiate(ctx)
	return err
}

func (s *Service) hostExternalAPICall(ctx context.Context, module api.Module, ptr uint32, byteCount uint32) uint64 {
	if err := requireAuthorizedAction(ctx); err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	callerID, _ := ctx.Value(appIDContextKey{}).(string)
	var request ExternalAPICallRequest
	if err := decodeGuestJSON(module, ptr, byteCount, &request); err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	manifest, err := s.activeManifestForApp(callerID)
	if err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	allowed := false
	for _, declared := range manifest.ExternalAPIs {
		if strings.TrimSpace(declared.APIKey) == strings.TrimSpace(request.APIKey) {
			allowed = true
			break
		}
	}
	if !allowed {
		return s.storeResult(ctx, map[string]any{"error": "external API is not declared in manifest"})
	}
	s.mu.RLock()
	executor := s.externalAPIExecutor
	s.mu.RUnlock()
	if executor == nil {
		return s.storeResult(ctx, map[string]any{"error": "external API executor is not configured"})
	}
	result, err := executor.CallExternalAPI(ctx, callerID, request)
	if err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	return s.storeResult(ctx, result)
}

func (s *Service) hostCallCapability(ctx context.Context, module api.Module, ptr uint32, byteCount uint32) uint64 {
	if err := requireAuthorizedAction(ctx); err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	callerID, _ := ctx.Value(appIDContextKey{}).(string)
	var request CapabilityCallRequest
	if err := decodeGuestJSON(module, ptr, byteCount, &request); err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	manifest, err := s.activeManifestForApp(callerID)
	if err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	allowed := false
	for _, consume := range manifest.Consumes {
		if strings.TrimSpace(consume.AppID) == strings.TrimSpace(request.AppID) && strings.TrimSpace(consume.Capability) == strings.TrimSpace(request.Capability) {
			allowed = true
			break
		}
	}
	if !allowed {
		return s.storeResult(ctx, map[string]any{"error": "capability is not declared in consumes"})
	}
	s.mu.RLock()
	broker := s.capabilityBroker
	s.mu.RUnlock()
	if broker == nil {
		return s.storeResult(ctx, map[string]any{"error": "capability broker is not configured"})
	}
	result, err := broker.CallCapability(ctx, callerID, request)
	if err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	return s.storeResult(ctx, result)
}

func (s *Service) hostDataList(ctx context.Context, module api.Module, ptr uint32, byteCount uint32) uint64 {
	appID, _ := ctx.Value(appIDContextKey{}).(string)
	var request dao.DataListRequest
	if err := decodeGuestJSON(module, ptr, byteCount, &request); err != nil {
		logDataOperationFailure(ctx, "data_list", "model", "unknown", time.Now(), "request_decode_failed")
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	startedAt := logDataOperationStart(ctx, "data_list", "model", request.Model)
	models, err := s.dataModelsForApp(appID)
	if err != nil {
		logDataOperationFailure(ctx, "data_list", "model", request.Model, startedAt, "model_resolve_failed")
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	result, err := s.dataStore.DataList(ctx, appID, models, request)
	if err != nil {
		logDataOperationFailure(ctx, "data_list", "model", request.Model, startedAt, "data_list_failed")
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	logDataOperationSuccess(ctx, "data_list", "model", request.Model, startedAt, fmt.Sprintf("rows=%d total=%d", len(result.Rows), result.Total))
	return s.storeResult(ctx, result)
}

func (s *Service) hostDataJoinQuery(ctx context.Context, module api.Module, ptr uint32, byteCount uint32) uint64 {
	appID, _ := ctx.Value(appIDContextKey{}).(string)
	var request dao.DataJoinRequest
	if err := decodeGuestJSON(module, ptr, byteCount, &request); err != nil {
		logDataOperationFailure(ctx, "data_join_query", "model", "unknown", time.Now(), "request_decode_failed")
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	startedAt := logDataOperationStart(ctx, "data_join_query", "model", request.From)
	models, err := s.dataModelsForApp(appID)
	if err != nil {
		logDataOperationFailure(ctx, "data_join_query", "model", request.From, startedAt, "model_resolve_failed")
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	relations, err := s.dataRelationsForApp(appID)
	if err != nil {
		logDataOperationFailure(ctx, "data_join_query", "model", request.From, startedAt, "relation_resolve_failed")
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	result, err := s.dataStore.DataJoinQuery(ctx, appID, models, relations, request)
	if err != nil {
		logDataOperationFailure(ctx, "data_join_query", "model", request.From, startedAt, "data_join_query_failed")
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	logDataOperationSuccess(ctx, "data_join_query", "model", request.From, startedAt, fmt.Sprintf("rows=%d total=%d", len(result.Rows), result.Total))
	return s.storeResult(ctx, result)
}

func (s *Service) hostDataRunQuery(ctx context.Context, module api.Module, ptr uint32, byteCount uint32) uint64 {
	appID, _ := ctx.Value(appIDContextKey{}).(string)
	var request dao.DataRunQueryRequest
	if err := decodeGuestJSON(module, ptr, byteCount, &request); err != nil {
		logDataOperationFailure(ctx, "data_run_query", "query", "unknown", time.Now(), "request_decode_failed")
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	startedAt := logDataOperationStart(ctx, "data_run_query", "query", request.Query)
	models, err := s.dataModelsForApp(appID)
	if err != nil {
		logDataOperationFailure(ctx, "data_run_query", "query", request.Query, startedAt, "model_resolve_failed")
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	relations, err := s.dataRelationsForApp(appID)
	if err != nil {
		logDataOperationFailure(ctx, "data_run_query", "query", request.Query, startedAt, "relation_resolve_failed")
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	queries, err := s.dataQueriesForApp(appID)
	if err != nil {
		logDataOperationFailure(ctx, "data_run_query", "query", request.Query, startedAt, "query_resolve_failed")
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	result, err := s.dataStore.DataRunQuery(ctx, appID, models, relations, queries, request)
	if err != nil {
		logDataOperationFailure(ctx, "data_run_query", "query", request.Query, startedAt, "data_run_query_failed")
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	logDataOperationSuccess(ctx, "data_run_query", "query", request.Query, startedAt, fmt.Sprintf("rows=%d total=%d", len(result.Rows), result.Total))
	return s.storeResult(ctx, result)
}

func (s *Service) hostDataGet(ctx context.Context, module api.Module, ptr uint32, byteCount uint32) uint64 {
	appID, _ := ctx.Value(appIDContextKey{}).(string)
	var request dao.DataGetRequest
	if err := decodeGuestJSON(module, ptr, byteCount, &request); err != nil {
		logDataOperationFailure(ctx, "data_get", "model", "unknown", time.Now(), "request_decode_failed")
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	startedAt := logDataOperationStart(ctx, "data_get", "model", request.Model)
	models, err := s.dataModelsForApp(appID)
	if err != nil {
		logDataOperationFailure(ctx, "data_get", "model", request.Model, startedAt, "model_resolve_failed")
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	record, err := s.dataStore.DataGet(ctx, appID, models, request)
	if err != nil {
		logDataOperationFailure(ctx, "data_get", "model", request.Model, startedAt, "data_get_failed")
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	logDataOperationSuccess(ctx, "data_get", "model", request.Model, startedAt, fmt.Sprintf("found=%t", record != nil))
	return s.storeResult(ctx, map[string]any{"record": record})
}

func (s *Service) hostDataCreate(ctx context.Context, module api.Module, ptr uint32, byteCount uint32) uint64 {
	if err := requireAuthorizedAction(ctx); err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	appID, _ := ctx.Value(appIDContextKey{}).(string)
	var request dao.DataWriteRequest
	if err := decodeGuestJSON(module, ptr, byteCount, &request); err != nil {
		logDataOperationFailure(ctx, "data_create", "model", "unknown", time.Now(), "request_decode_failed")
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	startedAt := logDataOperationStart(ctx, "data_create", "model", request.Model)
	models, err := s.dataModelsForApp(appID)
	if err != nil {
		logDataOperationFailure(ctx, "data_create", "model", request.Model, startedAt, "model_resolve_failed")
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	result, err := s.dataStore.DataCreate(ctx, appID, models, request)
	if err != nil {
		logDataOperationFailure(ctx, "data_create", "model", request.Model, startedAt, "data_create_failed")
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	logDataOperationSuccess(ctx, "data_create", "model", request.Model, startedAt, fmt.Sprintf("rows_affected=%d", result.RowsAffected))
	return s.storeResult(ctx, result)
}

func (s *Service) hostDataUpdate(ctx context.Context, module api.Module, ptr uint32, byteCount uint32) uint64 {
	if err := requireAuthorizedAction(ctx); err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	appID, _ := ctx.Value(appIDContextKey{}).(string)
	var request dao.DataWriteRequest
	if err := decodeGuestJSON(module, ptr, byteCount, &request); err != nil {
		logDataOperationFailure(ctx, "data_update", "model", "unknown", time.Now(), "request_decode_failed")
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	startedAt := logDataOperationStart(ctx, "data_update", "model", request.Model)
	models, err := s.dataModelsForApp(appID)
	if err != nil {
		logDataOperationFailure(ctx, "data_update", "model", request.Model, startedAt, "model_resolve_failed")
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	result, err := s.dataStore.DataUpdate(ctx, appID, models, request)
	if err != nil {
		logDataOperationFailure(ctx, "data_update", "model", request.Model, startedAt, "data_update_failed")
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	logDataOperationSuccess(ctx, "data_update", "model", request.Model, startedAt, fmt.Sprintf("rows_affected=%d", result.RowsAffected))
	return s.storeResult(ctx, result)
}

func (s *Service) hostDataDelete(ctx context.Context, module api.Module, ptr uint32, byteCount uint32) uint64 {
	if err := requireAuthorizedAction(ctx); err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	appID, _ := ctx.Value(appIDContextKey{}).(string)
	var request dao.DataGetRequest
	if err := decodeGuestJSON(module, ptr, byteCount, &request); err != nil {
		logDataOperationFailure(ctx, "data_delete", "model", "unknown", time.Now(), "request_decode_failed")
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	startedAt := logDataOperationStart(ctx, "data_delete", "model", request.Model)
	models, err := s.dataModelsForApp(appID)
	if err != nil {
		logDataOperationFailure(ctx, "data_delete", "model", request.Model, startedAt, "model_resolve_failed")
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	result, err := s.dataStore.DataDelete(ctx, appID, models, request)
	if err != nil {
		logDataOperationFailure(ctx, "data_delete", "model", request.Model, startedAt, "data_delete_failed")
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	logDataOperationSuccess(ctx, "data_delete", "model", request.Model, startedAt, fmt.Sprintf("rows_affected=%d", result.RowsAffected))
	return s.storeResult(ctx, result)
}

func (s *Service) hostResultLen(ctx context.Context, handle uint64) uint32 {
	data, ok := s.lookupResult(ctx, handle)
	if !ok {
		return 0
	}
	return uint32(len(data))
}

func (s *Service) hostResultRead(ctx context.Context, module api.Module, handle uint64, ptr uint32) uint32 {
	data, ok := s.lookupResult(ctx, handle)
	if !ok || len(data) == 0 {
		return 0
	}
	if !module.Memory().Write(ptr, data) {
		return 0
	}
	return uint32(len(data))
}

func (s *Service) hostResultStore(ctx context.Context, module api.Module, ptr uint32, byteCount uint32) uint64 {
	data, ok := module.Memory().Read(ptr, byteCount)
	if !ok {
		return s.storeResult(ctx, map[string]any{"error": "read guest memory failed"})
	}
	if !json.Valid(data) {
		return s.storeResult(ctx, map[string]any{"error": "result_store requires valid JSON"})
	}
	s.logGuestResult(ctx, data)
	return s.storeRawResult(ctx, data)
}

// hostLog is the bounded guest-to-host logging boundary. It accepts only a
// severity and text, preserving the outer invocation context and request ID.
func (s *Service) hostLog(ctx context.Context, module api.Module, level uint32, ptr uint32, byteCount uint32) {
	if byteCount == 0 {
		return
	}
	truncated := byteCount > maxGuestLogMessageBytes
	if truncated {
		byteCount = maxGuestLogMessageBytes
	}
	data, ok := module.Memory().Read(ptr, byteCount)
	if !ok {
		emitRuntimeLog(ctx, "ERROR wasm logger could not read guest memory")
		return
	}
	message := strings.TrimSpace(string(data))
	if message == "" {
		return
	}
	if truncated {
		message += " [truncated]"
	}
	emitRuntimeLog(ctx, guestLogLevel(level)+" "+message)
}

func guestLogLevel(level uint32) string {
	switch level {
	case 0:
		return "DEBUG"
	case 2:
		return "WARN"
	case 3:
		return "ERROR"
	default:
		return "INFO"
	}
}

// logGuestResult records the generated function's final business result. It
// gives every existing app a useful console trace even when its source does
// not explicitly use Go's log package.
func (s *Service) logGuestResult(ctx context.Context, data []byte) {
	action, _ := ctx.Value(invocationActionContextKey{}).(string)
	var response struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return
	}
	if strings.TrimSpace(response.Error) != "" {
		emitRuntimeLog(ctx, "ERROR action="+runtimeLogValue(action)+" result=failed code=guest_response_error")
		return
	}
	if response.Success {
		emitRuntimeLog(ctx, "INFO action="+runtimeLogValue(action)+" result=success")
	}
}

func logDataOperationStart(ctx context.Context, operation, targetKind, target string) time.Time {
	startedAt := time.Now()
	emitRuntimeLog(ctx, "INFO action="+runtimeLogAction(ctx)+" operation="+runtimeLogValue(operation)+" "+runtimeLogValue(targetKind)+"="+runtimeLogValue(target)+" phase=start")
	return startedAt
}

func logDataOperationSuccess(ctx context.Context, operation, targetKind, target string, startedAt time.Time, result string) {
	emitRuntimeLog(ctx, "INFO action="+runtimeLogAction(ctx)+" operation="+runtimeLogValue(operation)+" "+runtimeLogValue(targetKind)+"="+runtimeLogValue(target)+" result=success "+result+" duration_ms="+fmt.Sprint(time.Since(startedAt).Milliseconds()))
}

func logDataOperationFailure(ctx context.Context, operation, targetKind, target string, startedAt time.Time, code string) {
	emitRuntimeLog(ctx, "ERROR action="+runtimeLogAction(ctx)+" operation="+runtimeLogValue(operation)+" "+runtimeLogValue(targetKind)+"="+runtimeLogValue(target)+" result=failed code="+runtimeLogValue(code)+" duration_ms="+fmt.Sprint(time.Since(startedAt).Milliseconds()))
}

func runtimeLogAction(ctx context.Context) string {
	action, _ := ctx.Value(invocationActionContextKey{}).(string)
	return runtimeLogValue(action)
}

// runtimeLogValue prevents host diagnostic values from breaking a single-line
// key=value event. It deliberately omits request values and record contents.
func runtimeLogValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-', r == '.', r == ':':
			return r
		default:
			return '_'
		}
	}, value)
}

func emitRuntimeLog(ctx context.Context, content string) {
	collector, _ := ctx.Value(executionLogCollectorContextKey{}).(*executionLogCollector)
	if collector != nil && strings.TrimSpace(content) != "" {
		collector.Emit("runtime", content)
	}
}

func (s *Service) resultPayload(ctx context.Context, handle uint64) (string, error) {
	data, ok := s.lookupResult(ctx, handle)
	if !ok {
		return "", errors.New("result handle does not belong to current invocation")
	}
	return string(data), nil
}

func (s *Service) storeResult(ctx context.Context, payload any) uint64 {
	data, err := json.Marshal(payload)
	if err != nil {
		data = []byte(`{"error":"encode platform result failed"}`)
	}
	return s.storeRawResult(ctx, data)
}

func (s *Service) storeRawResult(ctx context.Context, data []byte) uint64 {
	if s.maxResultBytes > 0 && len(data) > s.maxResultBytes {
		data = []byte(`{"error":"platform result exceeds the configured size limit"}`)
	}
	appID, _ := ctx.Value(appIDContextKey{}).(string)
	invocationID, _ := ctx.Value(invocationIDContextKey{}).(uint64)
	s.mu.Lock()
	defer s.mu.Unlock()
	handle := s.nextID
	s.nextID++
	s.results[handle] = resultRecord{
		appID:        appID,
		invocationID: invocationID,
		data:         append([]byte(nil), data...),
	}
	return handle
}

func (s *Service) beginInvocation() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	invocationID := s.nextInvocationID
	s.nextInvocationID++
	return invocationID
}

func (s *Service) lookupResult(ctx context.Context, handle uint64) ([]byte, bool) {
	appID, _ := ctx.Value(appIDContextKey{}).(string)
	invocationID, _ := ctx.Value(invocationIDContextKey{}).(uint64)
	s.mu.RLock()
	record, ok := s.results[handle]
	s.mu.RUnlock()
	if !ok || record.appID != appID || record.invocationID != invocationID {
		return nil, false
	}
	return append([]byte(nil), record.data...), true
}

func (s *Service) cleanupInvocationResults(invocationID uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for handle, record := range s.results {
		if record.invocationID == invocationID {
			delete(s.results, handle)
		}
	}
}

func decodeGuestJSON(module api.Module, ptr uint32, byteCount uint32, target any) error {
	data, ok := module.Memory().Read(ptr, byteCount)
	if !ok {
		return errors.New("read guest memory failed")
	}
	return json.Unmarshal(data, target)
}
