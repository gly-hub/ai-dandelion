package logic

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
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

var externalAPIKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,159}$`)

type ExternalAPILogic struct {
	dao        *dao.ExternalAPI
	authorizer *FunctionAuthorizer
}

const externalAPIMaxResponseBytes = 1 << 20

func NewExternalAPILogic(d *dao.ExternalAPI, a *FunctionAuthorizer) *ExternalAPILogic {
	return &ExternalAPILogic{dao: d, authorizer: a}
}
func (l *ExternalAPILogic) ListClients(ctx context.Context, _ *funcoperation.ListExternalAPIClientsReq) ([]*funcoperation.ExternalAPIClient, error) {
	if err := l.authorizer.Require(ctx, externalAPIPermissionView); err != nil {
		return nil, err
	}
	rows, err := l.dao.ListClients(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*funcoperation.ExternalAPIClient, 0, len(rows))
	for i := range rows {
		out = append(out, clientToProto(&rows[i]))
	}
	return out, nil
}
func (l *ExternalAPILogic) CreateClient(ctx context.Context, req *funcoperation.CreateExternalAPIClientReq) (*funcoperation.ExternalAPIClient, error) {
	if err := l.authorizer.Require(ctx, externalAPIPermissionCreate); err != nil {
		return nil, err
	}
	key, base, headers, err := validateClient(req.GetClientKey(), req.GetBaseUrl(), req.GetDefaultHeadersJson())
	if err != nil {
		return nil, err
	}
	user, err := authctx.RequireUserID(ctx)
	if err != nil {
		return nil, err
	}
	importKey, importKeyHash, err := newSwaggerImportKey()
	if err != nil {
		return nil, err
	}
	now := nowUnixMicro()
	row := &model.ExternalAPIClient{UUID: uuid.NewString(), ClientKey: key, Name: requiredName(req.GetName()), BaseURL: base, DefaultHeadersJSON: headers, Description: strings.TrimSpace(req.GetDescription()), PreRequestScript: strings.TrimSpace(req.GetPreRequestScript()), PostResponseScript: strings.TrimSpace(req.GetPostResponseScript()), Status: "enabled", CreatedBy: user, UpdatedBy: user, CreatedAt: now, UpdatedAt: now, SwaggerImportKeyHash: importKeyHash}
	if row.Name == "" {
		return nil, errors.New("name is required")
	}
	if err := l.dao.CreateClient(ctx, row); err != nil {
		return nil, err
	}
	response := clientToProto(row)
	response.SwaggerImportKey = importKey
	return response, nil
}
func (l *ExternalAPILogic) UpdateClient(ctx context.Context, req *funcoperation.UpdateExternalAPIClientReq) (*funcoperation.ExternalAPIClient, error) {
	if err := l.authorizer.Require(ctx, externalAPIPermissionUpdate); err != nil {
		return nil, err
	}
	key, base, headers, err := validateClient(req.GetClientKey(), req.GetBaseUrl(), req.GetDefaultHeadersJson())
	if err != nil {
		return nil, err
	}
	row, err := l.dao.GetClient(ctx, key)
	if err != nil {
		return nil, err
	}
	user, err := authctx.RequireUserID(ctx)
	if err != nil {
		return nil, err
	}
	row.Name = requiredName(req.GetName())
	if row.Name == "" {
		return nil, errors.New("name is required")
	}
	row.BaseURL = base
	row.DefaultHeadersJSON = headers
	row.Description = strings.TrimSpace(req.GetDescription())
	row.PreRequestScript = strings.TrimSpace(req.GetPreRequestScript())
	row.PostResponseScript = strings.TrimSpace(req.GetPostResponseScript())
	row.Status = normalizeStatus(req.GetStatus())
	row.UpdatedBy = user
	row.UpdatedAt = nowUnixMicro()
	if err := l.dao.UpdateClient(ctx, row); err != nil {
		return nil, err
	}
	return clientToProto(row), nil
}
func (l *ExternalAPILogic) DeleteClient(ctx context.Context, req *funcoperation.DeleteExternalAPIClientReq) error {
	if err := l.authorizer.Require(ctx, externalAPIPermissionUpdate); err != nil {
		return err
	}
	key, err := normalizeExternalKey(req.GetClientKey())
	if err != nil {
		return err
	}
	if _, err = l.dao.GetClient(ctx, key); err != nil {
		return err
	}
	user, err := authctx.RequireUserID(ctx)
	if err != nil {
		return err
	}
	return l.dao.SoftDeleteClientAssets(ctx, key, user, nowUnixMicro())
}
func (l *ExternalAPILogic) ListDeletedClients(ctx context.Context, _ *funcoperation.ListDeletedExternalAPIClientsReq) ([]*funcoperation.ExternalAPIClient, error) {
	if err := l.authorizer.Require(ctx, externalAPIPermissionView); err != nil {
		return nil, err
	}
	rows, err := l.dao.ListDeletedClients(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*funcoperation.ExternalAPIClient, 0, len(rows))
	for index := range rows {
		out = append(out, clientToProto(&rows[index]))
	}
	return out, nil
}
func (l *ExternalAPILogic) PurgeClient(ctx context.Context, req *funcoperation.PurgeExternalAPIClientReq) error {
	if err := l.authorizer.Require(ctx, externalAPIPermissionUpdate); err != nil {
		return err
	}
	key, err := normalizeExternalKey(req.GetClientKey())
	if err != nil {
		return err
	}
	return l.dao.PurgeClientAssets(ctx, key)
}
func (l *ExternalAPILogic) RotateImportKey(ctx context.Context, req *funcoperation.RotateExternalAPIImportKeyReq) (string, error) {
	if err := l.authorizer.Require(ctx, externalAPIPermissionUpdate); err != nil {
		return "", err
	}
	key, err := normalizeExternalKey(req.GetClientKey())
	if err != nil {
		return "", err
	}
	row, err := l.dao.GetClient(ctx, key)
	if err != nil {
		return "", err
	}
	user, err := authctx.RequireUserID(ctx)
	if err != nil {
		return "", err
	}
	importKey, importKeyHash, err := newSwaggerImportKey()
	if err != nil {
		return "", err
	}
	row.SwaggerImportKeyHash = importKeyHash
	row.UpdatedBy = user
	row.UpdatedAt = nowUnixMicro()
	if err := l.dao.UpdateClient(ctx, row); err != nil {
		return "", err
	}
	return importKey, nil
}
func (l *ExternalAPILogic) ListAPIs(ctx context.Context, req *funcoperation.ListExternalAPIsReq) ([]*funcoperation.ExternalAPI, error) {
	if err := l.authorizer.Require(ctx, externalAPIPermissionView); err != nil {
		return nil, err
	}
	key, err := normalizeExternalKey(req.GetClientKey())
	if err != nil {
		return nil, err
	}
	user, err := authctx.RequireUserID(ctx)
	if err != nil {
		return nil, err
	}
	if _, err = l.ensureDefaultGroup(ctx, key, user); err != nil {
		return nil, err
	}
	rows, err := l.dao.ListAPIs(ctx, key)
	if err != nil {
		return nil, err
	}
	out := make([]*funcoperation.ExternalAPI, 0, len(rows))
	for i := range rows {
		out = append(out, apiToProto(&rows[i]))
	}
	return out, nil
}
func (l *ExternalAPILogic) ListGroups(ctx context.Context, req *funcoperation.ListExternalAPIGroupsReq) ([]*funcoperation.ExternalAPIGroup, error) {
	if err := l.authorizer.Require(ctx, externalAPIPermissionView); err != nil {
		return nil, err
	}
	key, err := normalizeExternalKey(req.GetClientKey())
	if err != nil {
		return nil, err
	}
	if _, err = l.dao.GetClient(ctx, key); err != nil {
		return nil, err
	}
	user, err := authctx.RequireUserID(ctx)
	if err != nil {
		return nil, err
	}
	if _, err = l.ensureDefaultGroup(ctx, key, user); err != nil {
		return nil, err
	}
	rows, err := l.dao.ListGroups(ctx, key)
	if err != nil {
		return nil, err
	}
	out := make([]*funcoperation.ExternalAPIGroup, 0, len(rows))
	for index := range rows {
		out = append(out, groupToProto(&rows[index]))
	}
	return out, nil
}
func (l *ExternalAPILogic) CreateGroup(ctx context.Context, req *funcoperation.CreateExternalAPIGroupReq) (*funcoperation.ExternalAPIGroup, error) {
	if err := l.authorizer.Require(ctx, externalAPIPermissionCreate); err != nil {
		return nil, err
	}
	clientKey, err := normalizeExternalKey(req.GetClientKey())
	if err != nil {
		return nil, err
	}
	if _, err = l.dao.GetClient(ctx, clientKey); err != nil {
		return nil, err
	}
	name := requiredName(req.GetName())
	if name == "" {
		return nil, errors.New("group name is required")
	}
	user, err := authctx.RequireUserID(ctx)
	if err != nil {
		return nil, err
	}
	parentID := strings.TrimSpace(req.GetParentId())
	if parentID != "" {
		if _, err = l.dao.GetGroup(ctx, clientKey, parentID); err != nil {
			return nil, errors.New("parent group does not exist")
		}
	}
	now := nowUnixMicro()
	row := &model.ExternalAPIGroup{UUID: uuid.NewString(), ClientKey: clientKey, ParentID: parentID, Name: name, Description: strings.TrimSpace(req.GetDescription()), Sort: req.GetSort(), CreatedBy: user, UpdatedBy: user, CreatedAt: now, UpdatedAt: now}
	if err := l.dao.CreateGroup(ctx, row); err != nil {
		return nil, err
	}
	return groupToProto(row), nil
}
func (l *ExternalAPILogic) CreateAPI(ctx context.Context, req *funcoperation.CreateExternalAPIReq) (*funcoperation.ExternalAPI, error) {
	if err := l.authorizer.Require(ctx, externalAPIPermissionCreate); err != nil {
		return nil, err
	}
	row, err := l.apiFromCreate(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := l.dao.CreateAPI(ctx, row); err != nil {
		return nil, err
	}
	return apiToProto(row), nil
}
func (l *ExternalAPILogic) UpdateAPI(ctx context.Context, req *funcoperation.UpdateExternalAPIReq) (*funcoperation.ExternalAPI, error) {
	if err := l.authorizer.Require(ctx, externalAPIPermissionUpdate); err != nil {
		return nil, err
	}
	clientKey, err := normalizeExternalKey(req.GetClientKey())
	if err != nil {
		return nil, err
	}
	apiKey, err := normalizeExternalKey(req.GetApiKey())
	if err != nil {
		return nil, err
	}
	row, err := l.dao.GetAPI(ctx, clientKey, apiKey)
	if err != nil {
		return nil, err
	}
	method, path, headers, requestSchema, responseSchema, err := validateAPI(req.GetMethod(), req.GetPath(), req.GetHeadersJson(), req.GetRequestSchemaJson(), req.GetResponseSchemaJson())
	if err != nil {
		return nil, err
	}
	user, err := authctx.RequireUserID(ctx)
	if err != nil {
		return nil, err
	}
	row.Name = requiredName(req.GetName())
	if row.Name == "" {
		return nil, errors.New("name is required")
	}
	row.Method = method
	row.Path = path
	row.HeadersJSON = headers
	row.RequestSchemaJSON = requestSchema
	row.ResponseSchemaJSON = responseSchema
	row.Description = strings.TrimSpace(req.GetDescription())
	row.Status = normalizeStatus(req.GetStatus())
	if groupID := strings.TrimSpace(req.GetGroupId()); groupID != "" {
		if _, err = l.dao.GetGroup(ctx, clientKey, groupID); err != nil {
			return nil, errors.New("API group does not exist")
		}
		row.GroupID = groupID
	}
	row.UpdatedBy = user
	row.UpdatedAt = nowUnixMicro()
	if err := l.dao.UpdateAPI(ctx, row); err != nil {
		return nil, err
	}
	return apiToProto(row), nil
}
func (l *ExternalAPILogic) DeleteAPI(ctx context.Context, req *funcoperation.DeleteExternalAPIReq) error {
	if err := l.authorizer.Require(ctx, externalAPIPermissionUpdate); err != nil {
		return err
	}
	clientKey, err := normalizeExternalKey(req.GetClientKey())
	if err != nil {
		return err
	}
	apiKey, err := normalizeExternalKey(req.GetApiKey())
	if err != nil {
		return err
	}
	row, err := l.dao.GetAPI(ctx, clientKey, apiKey)
	if err != nil {
		return err
	}
	return l.dao.DeleteAPI(ctx, row)
}
func (l *ExternalAPILogic) apiFromCreate(ctx context.Context, req *funcoperation.CreateExternalAPIReq) (*model.ExternalAPI, error) {
	clientKey, err := normalizeExternalKey(req.GetClientKey())
	if err != nil {
		return nil, err
	}
	if _, err = l.dao.GetClient(ctx, clientKey); err != nil {
		return nil, err
	}
	method, path, headers, requestSchema, responseSchema, err := validateAPI(req.GetMethod(), req.GetPath(), req.GetHeadersJson(), req.GetRequestSchemaJson(), req.GetResponseSchemaJson())
	if err != nil {
		return nil, err
	}
	apiKey := strings.TrimSpace(req.GetApiKey())
	if apiKey == "" {
		apiKey = importedAPIKey(clientKey, method, path)
	}
	apiKey, err = normalizeExternalKey(apiKey)
	if err != nil {
		return nil, err
	}
	user, err := authctx.RequireUserID(ctx)
	if err != nil {
		return nil, err
	}
	name := requiredName(req.GetName())
	if name == "" {
		return nil, errors.New("name is required")
	}
	now := nowUnixMicro()
	groupID := strings.TrimSpace(req.GetGroupId())
	if groupID == "" {
		group, groupErr := l.ensureDefaultGroup(ctx, clientKey, user)
		if groupErr != nil {
			return nil, groupErr
		}
		groupID = group.UUID
	} else if _, err = l.dao.GetGroup(ctx, clientKey, groupID); err != nil {
		return nil, errors.New("API group does not exist")
	}
	return &model.ExternalAPI{UUID: uuid.NewString(), ClientKey: clientKey, GroupID: groupID, APIKey: apiKey, Name: name, Method: method, Path: path, HeadersJSON: headers, RequestSchemaJSON: requestSchema, ResponseSchemaJSON: responseSchema, Description: strings.TrimSpace(req.GetDescription()), Status: "enabled", CreatedBy: user, UpdatedBy: user, CreatedAt: now, UpdatedAt: now}, nil
}

func (l *ExternalAPILogic) ensureDefaultGroup(ctx context.Context, clientKey, user string) (*model.ExternalAPIGroup, error) {
	group, err := l.dao.GetGroupByName(ctx, clientKey, "未分组")
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		now := nowUnixMicro()
		group = &model.ExternalAPIGroup{UUID: uuid.NewString(), ClientKey: clientKey, Name: "未分组", Description: "历史接口与未归类接口", Sort: 999999, CreatedBy: user, UpdatedBy: user, CreatedAt: now, UpdatedAt: now}
		if err = l.dao.CreateGroup(ctx, group); err != nil {
			// A simultaneous ListGroups/ListAPIs request may have won creation.
			group, err = l.dao.GetGroupByName(ctx, clientKey, "未分组")
			if err != nil {
				return nil, err
			}
		}
	}
	if err = l.dao.AssignUngroupedAPIs(ctx, clientKey, group.UUID); err != nil {
		return nil, err
	}
	return group, nil
}

// ImportDocument accepts OpenAPI 3.x and Swagger 2 JSON. It converts the
// source into the editable endpoint fields already used by the runtime; the
// original document is never treated as the only source of truth.
func (l *ExternalAPILogic) ImportDocument(ctx context.Context, req *funcoperation.ImportExternalAPIDocumentReq) (int32, int32, []*funcoperation.ExternalAPIGroup, []*funcoperation.ExternalAPI, error) {
	if err := l.authorizer.Require(ctx, externalAPIPermissionCreate); err != nil {
		return 0, 0, nil, nil, err
	}
	clientKey, err := normalizeExternalKey(req.GetClientKey())
	if err != nil {
		return 0, 0, nil, nil, err
	}
	if _, err = l.dao.GetClient(ctx, clientKey); err != nil {
		return 0, 0, nil, nil, err
	}
	if mode := strings.TrimSpace(req.GetMode()); mode != "" && mode != "upsert" {
		return 0, 0, nil, nil, errors.New("only upsert import mode is supported")
	}
	user, err := authctx.RequireUserID(ctx)
	if err != nil {
		return 0, 0, nil, nil, err
	}
	return l.importDocument(ctx, clientKey, req.GetDocumentJson(), user)
}

// UploadDocument accepts an untrusted external upload after authenticating it
// with the API key bound to the target interface client.
func (l *ExternalAPILogic) UploadDocument(ctx context.Context, req *funcoperation.UploadExternalAPIDocumentReq) (int32, int32, error) {
	clientKey, err := normalizeExternalKey(req.GetClientKey())
	if err != nil {
		return 0, 0, err
	}
	client, err := l.dao.GetClient(ctx, clientKey)
	if err != nil {
		return 0, 0, err
	}
	if client.Status != "enabled" {
		return 0, 0, errors.New("external API client is disabled")
	}
	providedHash := swaggerImportKeyHash(strings.TrimSpace(req.GetApiKey()))
	if client.SwaggerImportKeyHash == "" || subtle.ConstantTimeCompare([]byte(client.SwaggerImportKeyHash), []byte(providedHash)) != 1 {
		return 0, 0, errors.New("invalid Swagger import API key")
	}
	created, updated, _, _, err := l.importDocument(ctx, clientKey, req.GetDocumentJson(), "swagger-upload")
	return created, updated, err
}

func (l *ExternalAPILogic) importDocument(ctx context.Context, clientKey, documentJSON, user string) (int32, int32, []*funcoperation.ExternalAPIGroup, []*funcoperation.ExternalAPI, error) {
	if len(documentJSON) > 4<<20 {
		return 0, 0, nil, nil, errors.New("OpenAPI document exceeds 4 MB limit")
	}
	operations, err := parseExternalAPIDocument(documentJSON)
	if err != nil {
		return 0, 0, nil, nil, err
	}
	now := nowUnixMicro()
	groupMap := map[string]model.ExternalAPIGroup{}
	apiRows := make([]model.ExternalAPI, 0, len(operations))
	for _, operation := range operations {
		groupName := operation.group
		if _, ok := groupMap[groupName]; !ok {
			groupMap[groupName] = model.ExternalAPIGroup{UUID: uuid.NewString(), ClientKey: clientKey, Name: groupName, Sort: int32(len(groupMap) * 10), CreatedBy: user, UpdatedBy: user, CreatedAt: now, UpdatedAt: now}
		}
		apiRows = append(apiRows, model.ExternalAPI{
			UUID: uuid.NewString(), ClientKey: clientKey, GroupID: groupName,
			APIKey: importedAPIKey(clientKey, operation.method, operation.path), Name: operation.name,
			Method: operation.method, Path: operation.path, HeadersJSON: operation.headersJSON,
			RequestSchemaJSON: operation.requestJSON, ResponseSchemaJSON: operation.responseJSON,
			Description: operation.description, Status: "enabled", CreatedBy: user, UpdatedBy: user, CreatedAt: now, UpdatedAt: now,
		})
	}
	groups := make([]model.ExternalAPIGroup, 0, len(groupMap))
	for _, group := range groupMap {
		groups = append(groups, group)
	}
	created, updated, err := l.dao.ImportDocument(ctx, groups, apiRows)
	if err != nil {
		return 0, 0, nil, nil, err
	}
	savedGroups, err := l.dao.ListGroups(ctx, clientKey)
	if err != nil {
		return 0, 0, nil, nil, err
	}
	savedAPIs, err := l.dao.ListAPIs(ctx, clientKey)
	if err != nil {
		return 0, 0, nil, nil, err
	}
	groupOut := make([]*funcoperation.ExternalAPIGroup, 0, len(savedGroups))
	for index := range savedGroups {
		groupOut = append(groupOut, groupToProto(&savedGroups[index]))
	}
	apiOut := make([]*funcoperation.ExternalAPI, 0, len(savedAPIs))
	for index := range savedAPIs {
		apiOut = append(apiOut, apiToProto(&savedAPIs[index]))
	}
	return int32(created), int32(updated), groupOut, apiOut, nil
}

type importedAPIOperation struct {
	group, name, method, path, description, headersJSON, requestJSON, responseJSON string
}

func parseExternalAPIDocument(raw string) ([]importedAPIOperation, error) {
	var document map[string]any
	if err := json.Unmarshal([]byte(raw), &document); err != nil {
		return nil, errors.New("OpenAPI document must be valid JSON")
	}
	if stringField(document, "openapi") == "" && stringField(document, "swagger") != "2.0" {
		return nil, errors.New("document must be OpenAPI 3.x or Swagger 2.0 JSON")
	}
	paths, ok := document["paths"].(map[string]any)
	if !ok || len(paths) == 0 {
		return nil, errors.New("document has no paths")
	}
	normalizer := newOpenAPIDocumentNormalizer(document)
	operations := make([]importedAPIOperation, 0)
	for path, rawItem := range paths {
		item, ok := rawItem.(map[string]any)
		if !ok || !strings.HasPrefix(path, "/") {
			continue
		}
		pathParameters := normalizer.parameters(anySlice(item["parameters"]))
		for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
			rawOperation, ok := item[strings.ToLower(method)].(map[string]any)
			if !ok {
				continue
			}
			parameters := append(append([]any{}, pathParameters...), normalizer.parameters(anySlice(rawOperation["parameters"]))...)
			request := map[string]any{"openapi": "3.0", "parameters": parameters}
			if body, ok := rawOperation["requestBody"]; ok {
				request["requestBody"] = normalizer.requestBody(body)
			} else if body := normalizer.swaggerBody(parameters); body != nil {
				request["requestBody"] = body
			}
			response := map[string]any{"openapi": "3.0", "responses": normalizer.responses(objectField(rawOperation, "responses"))}
			requestJSON, _ := json.Marshal(request)
			responseJSON, _ := json.Marshal(response)
			group := firstString(anySlice(rawOperation["tags"]))
			if group == "" {
				group = "未分组"
			}
			name := stringField(rawOperation, "summary")
			if name == "" {
				name = stringField(rawOperation, "operationId")
			}
			if name == "" {
				name = method + " " + path
			}
			operations = append(operations, importedAPIOperation{group: group, name: name, method: method, path: path, description: stringField(rawOperation, "description"), headersJSON: importedHeaders(parameters), requestJSON: string(requestJSON), responseJSON: string(responseJSON)})
		}
	}
	if len(operations) == 0 {
		return nil, errors.New("document has no supported operations")
	}
	return operations, nil
}

// openAPIDocumentNormalizer converts the Swagger 2 shapes into the OpenAPI 3
// shapes stored by this module. It also inlines local component references so
// the document can remain useful after it has been imported on its own.
type openAPIDocumentNormalizer struct {
	schemas             map[string]any
	parametersByName    map[string]any
	responsesByName     map[string]any
	requestBodiesByName map[string]any
}

func newOpenAPIDocumentNormalizer(document map[string]any) *openAPIDocumentNormalizer {
	components := objectField(document, "components")
	schemas := objectField(document, "definitions")
	if componentSchemas := objectField(components, "schemas"); len(componentSchemas) > 0 {
		schemas = componentSchemas
	}
	parameters := objectField(document, "parameters")
	if componentParameters := objectField(components, "parameters"); len(componentParameters) > 0 {
		parameters = componentParameters
	}
	responses := objectField(document, "responses")
	if componentResponses := objectField(components, "responses"); len(componentResponses) > 0 {
		responses = componentResponses
	}
	return &openAPIDocumentNormalizer{
		schemas: schemas, parametersByName: parameters, responsesByName: responses,
		requestBodiesByName: objectField(components, "requestBodies"),
	}
}

func (n *openAPIDocumentNormalizer) parameters(values []any) []any {
	result := make([]any, 0, len(values))
	for _, value := range values {
		parameter := n.parameter(value)
		if parameter != nil {
			result = append(result, parameter)
		}
	}
	return result
}

func (n *openAPIDocumentNormalizer) parameter(raw any) map[string]any {
	parameter := n.resolveObject(raw, n.parametersByName, "#/parameters/", "#/components/parameters/")
	if parameter == nil {
		return nil
	}
	result := copyObject(parameter)
	if schema, ok := result["schema"]; ok {
		result["schema"] = n.schema(schema, map[string]bool{})
	} else if stringField(result, "in") != "body" {
		// Swagger 2 keeps primitive parameter type information on the parameter.
		if kind := stringField(result, "type"); kind != "" {
			schema := map[string]any{"type": kind}
			if format := stringField(result, "format"); format != "" {
				schema["format"] = format
			}
			result["schema"] = schema
		}
	}
	return result
}

func (n *openAPIDocumentNormalizer) requestBody(raw any) any {
	body := n.resolveObject(raw, n.requestBodiesByName, "#/components/requestBodies/")
	if body == nil {
		return raw
	}
	result := copyObject(body)
	result["content"] = n.content(objectField(body, "content"))
	return result
}

func (n *openAPIDocumentNormalizer) swaggerBody(parameters []any) any {
	for _, raw := range parameters {
		parameter, ok := raw.(map[string]any)
		if !ok || stringField(parameter, "in") != "body" {
			continue
		}
		body := map[string]any{"content": map[string]any{"application/json": map[string]any{"schema": n.schema(parameter["schema"], map[string]bool{})}}}
		if example, ok := parameter["x-example"]; ok {
			body["content"].(map[string]any)["application/json"].(map[string]any)["example"] = example
		}
		return body
	}
	return nil
}

func (n *openAPIDocumentNormalizer) responses(raw map[string]any) map[string]any {
	result := make(map[string]any, len(raw))
	for status, value := range raw {
		response := n.resolveObject(value, n.responsesByName, "#/responses/", "#/components/responses/")
		if response == nil {
			continue
		}
		normalized := copyObject(response)
		content := n.content(objectField(response, "content"))
		if schema, ok := response["schema"]; ok {
			media := map[string]any{"schema": n.schema(schema, map[string]bool{})}
			if examples := objectField(response, "examples"); examples != nil {
				if example, exists := examples["application/json"]; exists {
					media["example"] = example
				}
			}
			content["application/json"] = media
			delete(normalized, "schema")
		}
		if len(content) > 0 {
			normalized["content"] = content
		}
		result[status] = normalized
	}
	return result
}

func (n *openAPIDocumentNormalizer) content(raw map[string]any) map[string]any {
	result := make(map[string]any, len(raw))
	for mediaType, value := range raw {
		media, ok := value.(map[string]any)
		if !ok {
			continue
		}
		normalized := copyObject(media)
		if schema, ok := media["schema"]; ok {
			normalized["schema"] = n.schema(schema, map[string]bool{})
		}
		result[mediaType] = normalized
	}
	return result
}

func (n *openAPIDocumentNormalizer) schema(raw any, visited map[string]bool) any {
	schema, ok := raw.(map[string]any)
	if !ok {
		return raw
	}
	if ref := stringField(schema, "$ref"); ref != "" {
		if target, ok := n.reference(ref, n.schemas, "#/definitions/", "#/components/schemas/"); ok && !visited[ref] {
			nextVisited := copyVisited(visited)
			nextVisited[ref] = true
			resolved, _ := n.schema(target, nextVisited).(map[string]any)
			result := copyObject(resolved)
			for key, value := range schema {
				if key != "$ref" {
					result[key] = value
				}
			}
			return n.schema(result, nextVisited)
		}
	}
	result := copyObject(schema)
	if properties := objectField(schema, "properties"); properties != nil {
		normalized := make(map[string]any, len(properties))
		for name, value := range properties {
			normalized[name] = n.schema(value, copyVisited(visited))
		}
		result["properties"] = normalized
	}
	if items, ok := schema["items"]; ok {
		result["items"] = n.schema(items, copyVisited(visited))
	}
	if additional, ok := schema["additionalProperties"].(map[string]any); ok {
		result["additionalProperties"] = n.schema(additional, copyVisited(visited))
	}
	if allOf := anySlice(schema["allOf"]); len(allOf) > 0 {
		delete(result, "allOf")
		for _, part := range allOf {
			if normalized, ok := n.schema(part, copyVisited(visited)).(map[string]any); ok {
				mergeSchemas(result, normalized)
			}
		}
	}
	return result
}

func (n *openAPIDocumentNormalizer) resolveObject(raw any, source map[string]any, prefixes ...string) map[string]any {
	return n.resolveObjectWithVisited(raw, source, map[string]bool{}, prefixes...)
}

func (n *openAPIDocumentNormalizer) resolveObjectWithVisited(raw any, source map[string]any, visited map[string]bool, prefixes ...string) map[string]any {
	item, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	if ref := stringField(item, "$ref"); ref != "" && !visited[ref] {
		for _, prefix := range prefixes {
			if target, ok := n.reference(ref, source, prefix); ok {
				nextVisited := copyVisited(visited)
				nextVisited[ref] = true
				resolved := n.resolveObjectWithVisited(target, source, nextVisited, prefixes...)
				result := copyObject(resolved)
				for key, value := range item {
					if key != "$ref" {
						result[key] = value
					}
				}
				return result
			}
		}
	}
	return item
}

func (n *openAPIDocumentNormalizer) reference(ref string, source map[string]any, prefixes ...string) (any, bool) {
	for _, prefix := range prefixes {
		if prefix != "" && strings.HasPrefix(ref, prefix) {
			value, ok := source[strings.TrimPrefix(ref, prefix)]
			return value, ok
		}
	}
	return nil, false
}

func copyObject(source map[string]any) map[string]any {
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func copyVisited(source map[string]bool) map[string]bool {
	result := make(map[string]bool, len(source)+1)
	for key, value := range source {
		result[key] = value
	}
	return result
}

func mergeSchemas(destination, source map[string]any) {
	for key, value := range source {
		switch key {
		case "properties":
			properties := objectField(destination, "properties")
			if properties == nil {
				properties = map[string]any{}
			}
			for name, property := range objectField(source, "properties") {
				properties[name] = property
			}
			destination["properties"] = properties
		case "required":
			required := anySlice(destination["required"])
			seen := make(map[string]bool, len(required))
			for _, item := range required {
				seen[stringValue(item)] = true
			}
			for _, item := range anySlice(value) {
				if name := stringValue(item); name != "" && !seen[name] {
					required = append(required, name)
					seen[name] = true
				}
			}
			if len(required) > 0 {
				destination["required"] = required
			}
		default:
			if _, exists := destination[key]; !exists {
				destination[key] = value
			}
		}
	}
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func objectField(source map[string]any, name string) map[string]any {
	value, _ := source[name].(map[string]any)
	if value == nil {
		return map[string]any{}
	}
	return value
}
func anySlice(value any) []any { values, _ := value.([]any); return values }
func firstString(values []any) string {
	for _, value := range values {
		if item, ok := value.(string); ok && strings.TrimSpace(item) != "" {
			return strings.TrimSpace(item)
		}
	}
	return ""
}
func stringField(source map[string]any, name string) string {
	value, _ := source[name].(string)
	return strings.TrimSpace(value)
}
func swaggerBody(parameters []any) any {
	for _, raw := range parameters {
		parameter, ok := raw.(map[string]any)
		if !ok || stringField(parameter, "in") != "body" {
			continue
		}
		body := map[string]any{"content": map[string]any{"application/json": map[string]any{"schema": parameter["schema"]}}}
		if example, ok := parameter["x-example"]; ok {
			body["content"].(map[string]any)["application/json"].(map[string]any)["example"] = example
		}
		return body
	}
	return nil
}
func importedHeaders(parameters []any) string {
	headers := map[string]string{}
	for _, raw := range parameters {
		parameter, ok := raw.(map[string]any)
		if !ok || stringField(parameter, "in") != "header" {
			continue
		}
		name := stringField(parameter, "name")
		if name == "" {
			continue
		}
		if example, ok := parameter["example"].(string); ok {
			headers[name] = example
		} else if example, ok := parameter["x-example"].(string); ok {
			headers[name] = example
		}
	}
	encoded, _ := json.Marshal(headers)
	return string(encoded)
}
func importedAPIKey(clientKey, method, path string) string {
	digest := sha256.Sum256([]byte(strings.ToUpper(method) + " " + path))
	return fmt.Sprintf("%s.%s.%x", clientKey, strings.ToLower(method), digest[:8])
}

func newSwaggerImportKey() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := cryptorand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("generate Swagger import API key: %w", err)
	}
	key := "dnd_sw_" + hex.EncodeToString(bytes)
	return key, swaggerImportKeyHash(key), nil
}

func swaggerImportKeyHash(key string) string {
	digest := sha256.Sum256([]byte(key))
	return hex.EncodeToString(digest[:])
}

func (l *ExternalAPILogic) CallExternalAPI(ctx context.Context, _ string, req generatedapp.ExternalAPICallRequest) (any, error) {
	apiKey, err := normalizeExternalKey(req.APIKey)
	if err != nil {
		return nil, err
	}
	endpoint, err := l.dao.GetAPIByKey(ctx, apiKey)
	if err != nil {
		return nil, err
	}
	client, err := l.dao.GetClient(ctx, endpoint.ClientKey)
	if err != nil {
		return nil, err
	}
	httpClient, err := newGeneratedExternalAPIHTTPClient(client.BaseURL)
	if err != nil {
		return nil, err
	}
	result, err := executeExternalAPIWithClient(ctx, endpoint, client, req.Query, req.Headers, req.Body, httpClient)
	if err != nil {
		return nil, err
	}
	return map[string]any{"success": result.StatusCode >= 200 && result.StatusCode < 300, "statusCode": result.StatusCode, "data": result.Data}, nil
}

// Generated functions are an untrusted code boundary. They must not turn the
// platform's configured headers into an SSRF proxy for local or cloud-metadata
// endpoints. Administrator-initiated API tests use executeExternalAPI directly
// and retain local-environment support.
func newGeneratedExternalAPIHTTPClient(rawURL string) (*http.Client, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Hostname() == "" {
		return nil, errors.New("external API client base URL is invalid")
	}
	host := strings.TrimSpace(parsed.Hostname())
	if strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".localhost") {
		return nil, errors.New("generated functions cannot call localhost external APIs")
	}
	addresses := []net.IP{net.ParseIP(host)}
	if addresses[0] == nil {
		addresses, err = net.LookupIP(host)
		if err != nil || len(addresses) == 0 {
			return nil, errors.New("external API host could not be resolved")
		}
	}
	var target net.IP
	for _, address := range addresses {
		if isPublicExternalIP(address) {
			target = address
			break
		}
	}
	if target == nil {
		return nil, errors.New("generated functions cannot call private external APIs")
	}
	port := parsed.Port()
	if port == "" {
		if parsed.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	dialTarget := net.JoinHostPort(target.String(), port)
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
		return dialer.DialContext(ctx, "tcp", dialTarget)
	}
	return &http.Client{Timeout: 10 * time.Second, Transport: transport, CheckRedirect: func(next *http.Request, via []*http.Request) error {
		if next.URL.Hostname() != parsed.Hostname() {
			return errors.New("cross-host redirect is not allowed")
		}
		return nil
	}}, nil
}

func isPublicExternalIP(ip net.IP) bool {
	return ip != nil && !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsUnspecified() && !ip.IsMulticast()
}

type externalAPIHTTPResult struct {
	StatusCode int
	Headers    map[string][]string
	Data       any
}

func (l *ExternalAPILogic) TestExternalAPI(ctx context.Context, req *funcoperation.TestExternalAPIReq) (*funcoperation.TestExternalAPIResp, error) {
	if err := l.authorizer.Require(ctx, externalAPIPermissionUpdate); err != nil {
		return nil, err
	}
	clientKey, err := normalizeExternalKey(req.GetClientKey())
	if err != nil {
		return nil, err
	}
	apiKey, err := normalizeExternalKey(req.GetApiKey())
	if err != nil {
		return nil, err
	}
	endpoint, err := l.dao.GetAPI(ctx, clientKey, apiKey)
	if err != nil {
		return nil, err
	}
	client, err := l.dao.GetClient(ctx, clientKey)
	if err != nil {
		return nil, err
	}
	query, err := parseObject(req.GetQueryJson(), "query")
	if err != nil {
		return nil, err
	}
	headers, err := parseObject(req.GetHeadersJson(), "headers")
	if err != nil {
		return nil, err
	}
	body, err := parseJSONValue(req.GetBodyJson())
	if err != nil {
		return nil, errors.New("body must be valid JSON")
	}
	started := time.Now()
	result, err := executeExternalAPI(ctx, endpoint, client, query, headers, body)
	if err != nil {
		return nil, err
	}
	responseHeaders, _ := json.Marshal(result.Headers)
	responseBody, _ := json.Marshal(result.Data)
	return &funcoperation.TestExternalAPIResp{StatusCode: int32(result.StatusCode), ResponseHeadersJson: string(responseHeaders), ResponseBodyJson: string(responseBody), DurationMs: time.Since(started).Milliseconds()}, nil
}

func executeExternalAPI(ctx context.Context, endpoint *model.ExternalAPI, client *model.ExternalAPIClient, values, headers map[string]any, payload any) (*externalAPIHTTPResult, error) {
	return executeExternalAPIWithClient(ctx, endpoint, client, values, headers, payload, nil)
}

func executeExternalAPIWithClient(ctx context.Context, endpoint *model.ExternalAPI, client *model.ExternalAPIClient, values, headers map[string]any, payload any, httpClient *http.Client) (*externalAPIHTTPResult, error) {
	if endpoint.Status != "enabled" {
		return nil, errors.New("external API is disabled")
	}
	if client.Status != "enabled" {
		return nil, errors.New("external API client is disabled")
	}
	baseURL, err := url.Parse(client.BaseURL)
	if err != nil {
		return nil, errors.New("external API client base URL is invalid")
	}
	path, queryValues := expandEndpointPath(endpoint.Path, values)
	pathURL, err := url.Parse(path)
	if err != nil || pathURL.IsAbs() {
		return nil, errors.New("external API path is invalid")
	}
	requestURL := baseURL.ResolveReference(pathURL)
	query := requestURL.Query()
	for key, value := range queryValues {
		query.Set(key, fmt.Sprint(value))
	}
	requestURL.RawQuery = query.Encode()
	var body io.Reader
	requestBodyText := ""
	if payload != nil {
		encoded, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return nil, errors.New("external API body is not JSON serializable")
		}
		requestBodyText = string(encoded)
		body = strings.NewReader(requestBodyText)
	}
	httpReq, err := http.NewRequestWithContext(ctx, endpoint.Method, requestURL.String(), body)
	if err != nil {
		return nil, err
	}
	applyDynamicHeaders(httpReq, headers)
	// Platform-owned values are applied last and cannot be overridden by a
	// generated app or an interactive test request.
	if err := applyStaticHeaders(httpReq, client.DefaultHeadersJSON); err != nil {
		return nil, err
	}
	if err := applyStaticHeaders(httpReq, endpoint.HeadersJSON); err != nil {
		return nil, err
	}
	preResult, err := runPreRequestScript(client.PreRequestScript, httpReq, payload, requestBodyText)
	if err != nil {
		return nil, err
	}
	if preResult != nil {
		requestBodyText = preResult.bodyText
		if !preResult.rawBody {
			encoded, marshalErr := json.Marshal(preResult.body)
			if marshalErr != nil {
				return nil, errors.New("pre-request script body is not JSON serializable")
			}
			requestBodyText = string(encoded)
		}
		setExternalAPIRequestBody(httpReq, requestBodyText)
	}
	if requestBodyText != "" && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(next *http.Request, via []*http.Request) error {
			if next.URL.Host != baseURL.Host {
				return errors.New("cross-host redirect is not allowed")
			}
			return nil
		}}
	}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, externalAPIMaxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > externalAPIMaxResponseBytes {
		return nil, errors.New("external API response exceeds size limit")
	}
	postResult, err := runPostResponseScript(client.PostResponseScript, httpReq, requestBodyText, resp, raw)
	if err != nil {
		return nil, err
	}
	if postResult != nil {
		return &externalAPIHTTPResult{StatusCode: postResult.status, Headers: postResult.headers, Data: postResult.data}, nil
	}
	return &externalAPIHTTPResult{StatusCode: resp.StatusCode, Headers: resp.Header, Data: decodeExternalAPIBody(raw)}, nil
}

func setExternalAPIRequestBody(req *http.Request, value string) {
	req.Body = io.NopCloser(strings.NewReader(value))
	req.ContentLength = int64(len(value))
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader(value)), nil }
}

func expandEndpointPath(path string, values map[string]any) (string, map[string]any) {
	query := make(map[string]any, len(values))
	for key, value := range values {
		placeholder := "{" + key + "}"
		if strings.Contains(path, placeholder) {
			path = strings.ReplaceAll(path, placeholder, url.PathEscape(fmt.Sprint(value)))
			continue
		}
		query[key] = value
	}
	return path, query
}

func applyDynamicHeaders(req *http.Request, headers map[string]any) {
	for key, value := range headers {
		if strings.TrimSpace(key) != "" {
			req.Header.Set(key, fmt.Sprint(value))
		}
	}
}

func parseObject(raw, label string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}, nil
	}
	var value map[string]any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, fmt.Errorf("%s must be a JSON object", label)
	}
	return value, nil
}
func parseJSONValue(raw string) (any, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, err
	}
	return value, nil
}

func applyStaticHeaders(req *http.Request, raw string) error {
	var values map[string]string
	if err := json.Unmarshal([]byte(defaultJSON(raw, "{}")), &values); err != nil {
		return errors.New("configured external API headers are invalid")
	}
	for key, value := range values {
		req.Header.Set(key, value)
	}
	return nil
}
func normalizeExternalKey(raw string) (string, error) {
	key := strings.ToLower(strings.TrimSpace(raw))
	if !externalAPIKeyPattern.MatchString(key) {
		return "", errors.New("key must start with a lowercase letter and contain only lowercase letters, numbers, dots, hyphens, or underscores")
	}
	return key, nil
}
func requiredName(raw string) string { return strings.TrimSpace(raw) }
func normalizeStatus(raw string) string {
	if strings.EqualFold(strings.TrimSpace(raw), "disabled") {
		return "disabled"
	}
	return "enabled"
}
func validateClient(key, rawURL, rawHeaders string) (string, string, string, error) {
	key, err := normalizeExternalKey(key)
	if err != nil {
		return "", "", "", err
	}
	parsed, err := url.ParseRequestURI(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", "", "", errors.New("base URL must be an absolute HTTP or HTTPS URL")
	}
	headers, err := normalizeJSONObject(rawHeaders)
	if err != nil {
		return "", "", "", errors.New("default headers must be a JSON object")
	}
	return key, strings.TrimRight(parsed.String(), "/"), headers, nil
}
func validateAPI(method, path, headers, requestSchema, responseSchema string) (string, string, string, string, string, error) {
	method = strings.ToUpper(strings.TrimSpace(method))
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE":
	default:
		return "", "", "", "", "", errors.New("method must be GET, POST, PUT, PATCH, or DELETE")
	}
	path = strings.TrimSpace(path)
	if !strings.HasPrefix(path, "/") || strings.Contains(path, "://") {
		return "", "", "", "", "", errors.New("path must start with /")
	}
	headers, err := normalizeJSONObject(headers)
	if err != nil {
		return "", "", "", "", "", errors.New("headers must be a JSON object")
	}
	requestSchema, err = normalizeJSON(requestSchema)
	if err != nil {
		return "", "", "", "", "", errors.New("request schema must be valid JSON")
	}
	responseSchema, err = normalizeJSON(responseSchema)
	if err != nil {
		return "", "", "", "", "", errors.New("response schema must be valid JSON")
	}
	return method, path, headers, requestSchema, responseSchema, nil
}
func normalizeJSONObject(raw string) (string, error) {
	var value map[string]any
	if err := json.Unmarshal([]byte(defaultJSON(raw, "{}")), &value); err != nil {
		return "", err
	}
	out, err := json.Marshal(value)
	return string(out), err
}
func normalizeJSON(raw string) (string, error) {
	var value any
	if err := json.Unmarshal([]byte(defaultJSON(raw, "{}")), &value); err != nil {
		return "", err
	}
	out, err := json.Marshal(value)
	return string(out), err
}
func defaultJSON(raw, fallback string) string {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	return raw
}
func clientToProto(row *model.ExternalAPIClient) *funcoperation.ExternalAPIClient {
	if row == nil {
		return nil
	}
	return &funcoperation.ExternalAPIClient{Id: row.UUID, ClientKey: row.ClientKey, Name: row.Name, BaseUrl: row.BaseURL, DefaultHeadersJson: row.DefaultHeadersJSON, Description: row.Description, PreRequestScript: row.PreRequestScript, PostResponseScript: row.PostResponseScript, SwaggerImportKeyConfigured: row.SwaggerImportKeyHash != "", Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, DeletedAt: row.DeletedAt}
}
func apiToProto(row *model.ExternalAPI) *funcoperation.ExternalAPI {
	if row == nil {
		return nil
	}
	return &funcoperation.ExternalAPI{Id: row.UUID, ApiKey: row.APIKey, ClientKey: row.ClientKey, GroupId: row.GroupID, Name: row.Name, Method: row.Method, Path: row.Path, HeadersJson: row.HeadersJSON, RequestSchemaJson: row.RequestSchemaJSON, ResponseSchemaJson: row.ResponseSchemaJSON, Description: row.Description, Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}
func groupToProto(row *model.ExternalAPIGroup) *funcoperation.ExternalAPIGroup {
	if row == nil {
		return nil
	}
	return &funcoperation.ExternalAPIGroup{Id: row.UUID, ClientKey: row.ClientKey, ParentId: row.ParentID, Name: row.Name, Description: row.Description, Sort: row.Sort, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}
