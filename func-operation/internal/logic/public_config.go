package logic

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/dao"
	"github.com/gly-hub/ai-dandelion/func-operation/internal/model"
	funcoperation "github.com/gly-hub/ai-dandelion/proto/func-operation"
	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	publicConfigSourceCreate   = "create"
	publicConfigSourceUpdate   = "update"
	publicConfigSourceRollback = "rollback"
	publicConfigSourceUpload   = "api-upload"
)

var publicConfigKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,119}$`)

type PublicConfigLogic struct {
	configDao  *dao.PublicConfig
	authorizer *FunctionAuthorizer
}

func NewPublicConfigLogic(configDao *dao.PublicConfig, authorizer *FunctionAuthorizer) *PublicConfigLogic {
	return &PublicConfigLogic{configDao: configDao, authorizer: authorizer}
}

func (p *PublicConfigLogic) List(ctx context.Context, req *funcoperation.ListPublicConfigsReq) ([]*funcoperation.PublicConfig, error) {
	if err := p.authorizer.Require(ctx, publicConfigPermissionView); err != nil {
		return nil, err
	}
	items, err := p.configDao.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*funcoperation.PublicConfig, 0, len(items))
	for i := range items {
		out = append(out, publicConfigToProto(&items[i]))
	}
	return out, nil
}

func (p *PublicConfigLogic) Create(ctx context.Context, req *funcoperation.CreatePublicConfigReq) (*funcoperation.PublicConfig, error) {
	if err := p.authorizer.Require(ctx, publicConfigPermissionCreate); err != nil {
		return nil, err
	}
	configKey, err := normalizePublicConfigKey(req.GetConfigKey())
	if err != nil {
		return nil, err
	}
	valueJSON, err := normalizeOptionValueJSON(req.GetValueJson())
	if err != nil {
		return nil, err
	}
	operatorID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return nil, err
	}
	now := nowUnixMicro()
	item := &model.PublicConfig{
		UUID:        uuid.NewString(),
		ConfigKey:   configKey,
		Name:        defaultPublicConfigName(req.GetName(), configKey),
		Description: strings.TrimSpace(req.GetDescription()),
		ValueJSON:   valueJSON,
		Version:     1,
		CreatedBy:   operatorID,
		UpdatedBy:   operatorID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	version := &model.PublicConfigVersion{
		UUID:       uuid.NewString(),
		ConfigID:   item.UUID,
		ConfigKey:  item.ConfigKey,
		Version:    item.Version,
		ValueJSON:  item.ValueJSON,
		OperatorID: operatorID,
		Source:     publicConfigSourceCreate,
		CreatedAt:  now,
	}
	if err := p.configDao.Create(ctx, item, version); err != nil {
		return nil, err
	}
	return publicConfigToProto(item), nil
}

func (p *PublicConfigLogic) Update(ctx context.Context, req *funcoperation.UpdatePublicConfigReq) (*funcoperation.PublicConfig, error) {
	if err := p.authorizer.Require(ctx, publicConfigPermissionUpdate); err != nil {
		return nil, err
	}
	configKey, err := normalizePublicConfigKey(req.GetConfigKey())
	if err != nil {
		return nil, err
	}
	current, err := p.configDao.Get(ctx, configKey)
	if err != nil {
		return nil, err
	}
	valueJSON, err := normalizeOptionValueJSON(req.GetValueJson())
	if err != nil {
		return nil, err
	}
	operatorID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return nil, err
	}
	updated, err := p.configDao.UpdateValue(
		ctx,
		configKey,
		defaultPublicConfigName(req.GetName(), current.Name),
		strings.TrimSpace(req.GetDescription()),
		valueJSON,
		operatorID,
		publicConfigSourceUpdate,
		nowUnixMicro(),
	)
	if err != nil {
		return nil, err
	}
	return publicConfigToProto(updated), nil
}

func (p *PublicConfigLogic) ListVersions(ctx context.Context, req *funcoperation.ListPublicConfigVersionsReq) ([]*funcoperation.PublicConfigVersion, error) {
	if err := p.authorizer.Require(ctx, publicConfigPermissionView); err != nil {
		return nil, err
	}
	configKey, err := normalizePublicConfigKey(req.GetConfigKey())
	if err != nil {
		return nil, err
	}
	items, err := p.configDao.ListVersions(ctx, configKey)
	if err != nil {
		return nil, err
	}
	out := make([]*funcoperation.PublicConfigVersion, 0, len(items))
	for i := range items {
		out = append(out, publicConfigVersionToProto(&items[i]))
	}
	return out, nil
}

func (p *PublicConfigLogic) Rollback(ctx context.Context, req *funcoperation.RollbackPublicConfigReq) (*funcoperation.PublicConfig, error) {
	if err := p.authorizer.Require(ctx, publicConfigPermissionRollback); err != nil {
		return nil, err
	}
	configKey, err := normalizePublicConfigKey(req.GetConfigKey())
	if err != nil {
		return nil, err
	}
	if req.GetVersion() <= 0 {
		return nil, errors.New("version is required")
	}
	current, err := p.configDao.Get(ctx, configKey)
	if err != nil {
		return nil, err
	}
	target, err := p.configDao.GetVersion(ctx, configKey, req.GetVersion())
	if err != nil {
		return nil, err
	}
	operatorID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return nil, err
	}
	updated, err := p.configDao.UpdateValue(ctx, configKey, current.Name, current.Description, target.ValueJSON, operatorID, publicConfigSourceRollback, nowUnixMicro())
	if err != nil {
		return nil, err
	}
	return publicConfigToProto(updated), nil
}

func (p *PublicConfigLogic) RotateImportKey(ctx context.Context, _ *funcoperation.RotatePublicConfigImportKeyReq) (string, error) {
	if err := p.authorizer.Require(ctx, publicConfigPermissionUpdate); err != nil {
		return "", err
	}
	operatorID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return "", err
	}
	key, keyHash, err := newSwaggerImportKey()
	if err != nil {
		return "", err
	}
	if err := p.configDao.UpsertImportKey(ctx, keyHash, operatorID, nowUnixMicro()); err != nil {
		return "", err
	}
	return key, nil
}

// Import accepts a key-to-options JSON object and upserts every declared
// public config under the single public-config import key.
func (p *PublicConfigLogic) Import(ctx context.Context, req *funcoperation.ImportPublicConfigsReq) ([]*funcoperation.PublicConfig, error) {
	importKey, err := p.configDao.GetImportKey(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("public config import API key has not been created")
		}
		return nil, err
	}
	providedHash := swaggerImportKeyHash(strings.TrimSpace(req.GetApiKey()))
	if importKey.KeyHash == "" || subtle.ConstantTimeCompare([]byte(importKey.KeyHash), []byte(providedHash)) != 1 {
		return nil, errors.New("invalid public config import API key")
	}
	var input map[string]json.RawMessage
	if err := json.Unmarshal([]byte(req.GetConfigsJson()), &input); err != nil || len(input) == 0 {
		return nil, errors.New("import body must be a non-empty JSON object of config key to option array")
	}
	values := make(map[string]string, len(input))
	for rawKey, rawValue := range input {
		configKey, err := normalizePublicConfigKey(rawKey)
		if err != nil {
			return nil, err
		}
		valueJSON, err := normalizeOptionValueJSON(string(rawValue))
		if err != nil {
			return nil, fmt.Errorf("config %q: %w", configKey, err)
		}
		values[configKey] = valueJSON
	}
	updated, err := p.configDao.ImportValues(ctx, values, "api-upload", publicConfigSourceUpload, nowUnixMicro())
	if err != nil {
		return nil, err
	}
	out := make([]*funcoperation.PublicConfig, 0, len(updated))
	for index := range updated {
		out = append(out, publicConfigToProto(&updated[index]))
	}
	return out, nil
}

// ResolveValues is used only after the caller's generated-function access has
// been authorized. Values stay as raw JSON so generated pages retain their
// typed option array rather than receiving an ad-hoc server-side conversion.
func (p *PublicConfigLogic) ResolveValues(ctx context.Context, keys []string) (map[string]json.RawMessage, error) {
	if p == nil || p.configDao == nil {
		return nil, errors.New("public config is not configured")
	}
	values := make(map[string]json.RawMessage, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for _, rawKey := range keys {
		configKey, err := normalizePublicConfigKey(rawKey)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[configKey]; ok {
			continue
		}
		seen[configKey] = struct{}{}
		item, err := p.configDao.Get(ctx, configKey)
		if err != nil {
			return nil, fmt.Errorf("load public config %q: %w", configKey, err)
		}
		values[configKey] = json.RawMessage(item.ValueJSON)
	}
	return values, nil
}

func normalizePublicConfigKey(value string) (string, error) {
	key := strings.ToLower(strings.TrimSpace(value))
	if !publicConfigKeyPattern.MatchString(key) {
		return "", errors.New("config key must start with a lowercase letter and contain only lowercase letters, numbers, hyphens, or underscores")
	}
	return key, nil
}

func defaultPublicConfigName(value string, fallback string) string {
	if name := strings.TrimSpace(value); name != "" {
		return name
	}
	return strings.TrimSpace(fallback)
}

func normalizeOptionValueJSON(raw string) (string, error) {
	var values []map[string]any
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&values); err != nil {
		return "", errors.New("config value must be a JSON array of options")
	}
	if len(values) == 0 {
		return "", errors.New("config value must contain at least one option")
	}
	seenValues := make(map[string]struct{}, len(values))
	for index, item := range values {
		value, valueOK := item["value"].(string)
		label, labelOK := item["label"].(string)
		value = strings.TrimSpace(value)
		label = strings.TrimSpace(label)
		if !valueOK || value == "" || !labelOK || label == "" {
			return "", fmt.Errorf("option %d must have non-empty string value and label", index+1)
		}
		if _, exists := seenValues[value]; exists {
			return "", fmt.Errorf("option value %q is duplicated", value)
		}
		seenValues[value] = struct{}{}
		item["value"] = value
		item["label"] = label
	}
	normalized, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("encode config value: %w", err)
	}
	return string(normalized), nil
}

func publicConfigToProto(item *model.PublicConfig) *funcoperation.PublicConfig {
	if item == nil {
		return nil
	}
	return &funcoperation.PublicConfig{
		Id: item.UUID, ConfigKey: item.ConfigKey, Name: item.Name, Description: item.Description,
		ValueJson: item.ValueJSON, Version: item.Version, UpdatedBy: item.UpdatedBy,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}

func publicConfigVersionToProto(item *model.PublicConfigVersion) *funcoperation.PublicConfigVersion {
	if item == nil {
		return nil
	}
	return &funcoperation.PublicConfigVersion{
		Id: item.UUID, ConfigKey: item.ConfigKey, Version: item.Version, ValueJson: item.ValueJSON,
		OperatorId: item.OperatorID, Source: item.Source, CreatedAt: item.CreatedAt,
	}
}
