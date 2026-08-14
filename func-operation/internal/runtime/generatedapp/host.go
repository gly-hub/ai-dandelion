package generatedapp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/team-dandelion/ai-dandelion/func-operation/internal/dao"
	"github.com/tetratelabs/wazero/api"
)

type appIDContextKey struct{}
type invocationIDContextKey struct{}
type privilegedActionContextKey struct{}

// WithAuthorizedAction marks an invocation whose action has already passed
// the generated-function menu check in the service layer. Guest code cannot
// create this server-side context value.
func WithAuthorizedAction(ctx context.Context, action string) context.Context {
	return context.WithValue(ctx, privilegedActionContextKey{}, strings.TrimSpace(action))
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
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	models, err := s.dataModelsForApp(appID)
	if err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	result, err := s.dataStore.DataList(ctx, appID, models, request)
	if err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	return s.storeResult(ctx, result)
}

func (s *Service) hostDataJoinQuery(ctx context.Context, module api.Module, ptr uint32, byteCount uint32) uint64 {
	appID, _ := ctx.Value(appIDContextKey{}).(string)
	var request dao.DataJoinRequest
	if err := decodeGuestJSON(module, ptr, byteCount, &request); err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	models, err := s.dataModelsForApp(appID)
	if err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	relations, err := s.dataRelationsForApp(appID)
	if err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	result, err := s.dataStore.DataJoinQuery(ctx, appID, models, relations, request)
	if err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	return s.storeResult(ctx, result)
}

func (s *Service) hostDataRunQuery(ctx context.Context, module api.Module, ptr uint32, byteCount uint32) uint64 {
	appID, _ := ctx.Value(appIDContextKey{}).(string)
	var request dao.DataRunQueryRequest
	if err := decodeGuestJSON(module, ptr, byteCount, &request); err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	models, err := s.dataModelsForApp(appID)
	if err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	relations, err := s.dataRelationsForApp(appID)
	if err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	queries, err := s.dataQueriesForApp(appID)
	if err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	result, err := s.dataStore.DataRunQuery(ctx, appID, models, relations, queries, request)
	if err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	return s.storeResult(ctx, result)
}

func (s *Service) hostDataGet(ctx context.Context, module api.Module, ptr uint32, byteCount uint32) uint64 {
	appID, _ := ctx.Value(appIDContextKey{}).(string)
	var request dao.DataGetRequest
	if err := decodeGuestJSON(module, ptr, byteCount, &request); err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	models, err := s.dataModelsForApp(appID)
	if err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	record, err := s.dataStore.DataGet(ctx, appID, models, request)
	if err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	return s.storeResult(ctx, map[string]any{"record": record})
}

func (s *Service) hostDataCreate(ctx context.Context, module api.Module, ptr uint32, byteCount uint32) uint64 {
	if err := requireAuthorizedAction(ctx); err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	appID, _ := ctx.Value(appIDContextKey{}).(string)
	var request dao.DataWriteRequest
	if err := decodeGuestJSON(module, ptr, byteCount, &request); err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	models, err := s.dataModelsForApp(appID)
	if err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	result, err := s.dataStore.DataCreate(ctx, appID, models, request)
	if err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	return s.storeResult(ctx, result)
}

func (s *Service) hostDataUpdate(ctx context.Context, module api.Module, ptr uint32, byteCount uint32) uint64 {
	if err := requireAuthorizedAction(ctx); err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	appID, _ := ctx.Value(appIDContextKey{}).(string)
	var request dao.DataWriteRequest
	if err := decodeGuestJSON(module, ptr, byteCount, &request); err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	models, err := s.dataModelsForApp(appID)
	if err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	result, err := s.dataStore.DataUpdate(ctx, appID, models, request)
	if err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	return s.storeResult(ctx, result)
}

func (s *Service) hostDataDelete(ctx context.Context, module api.Module, ptr uint32, byteCount uint32) uint64 {
	if err := requireAuthorizedAction(ctx); err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	appID, _ := ctx.Value(appIDContextKey{}).(string)
	var request dao.DataGetRequest
	if err := decodeGuestJSON(module, ptr, byteCount, &request); err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	models, err := s.dataModelsForApp(appID)
	if err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
	result, err := s.dataStore.DataDelete(ctx, appID, models, request)
	if err != nil {
		return s.storeResult(ctx, map[string]any{"error": err.Error()})
	}
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
	return s.storeRawResult(ctx, data)
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
