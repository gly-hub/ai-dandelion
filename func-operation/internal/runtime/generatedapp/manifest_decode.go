package generatedapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/dao"
)

type manifestDataModelFlexible struct {
	Name        string                    `json:"name"`
	Label       string                    `json:"label"`
	Fields      json.RawMessage           `json:"fields"`
	Validations map[string]map[string]any `json:"validations"`
	Indexes     []string                  `json:"indexes"`
}

type manifestRelationFlexible struct {
	Name       string `json:"name"`
	From       string `json:"from"`
	To         string `json:"to"`
	Type       string `json:"type"`
	ForeignKey string `json:"foreignKey"`
}

type manifestActionFlexible struct {
	Key    string `json:"key"`
	Action string `json:"action"`
	Name   string `json:"name"`
	Mode   string `json:"mode"`
}

func decodeManifest(raw []byte) (manifest, []byte, bool, error) {
	return decodeManifestWithDir("", raw)
}

func decodeManifestWithDir(appDir string, raw []byte) (manifest, []byte, bool, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return manifest{}, nil, false, err
	}

	dataModelsRaw := envelope["dataModels"]
	relationsRaw := envelope["relations"]
	queriesRaw := envelope["queries"]
	actionsRaw := envelope["actions"]
	delete(envelope, "dataModels")
	delete(envelope, "relations")
	delete(envelope, "queries")
	delete(envelope, "actions")

	trimmed, err := json.Marshal(envelope)
	if err != nil {
		return manifest{}, nil, false, err
	}

	var item manifest
	if err := json.Unmarshal(trimmed, &item); err != nil {
		return manifest{}, nil, false, err
	}
	if strings.TrimSpace(item.SchemaVersion) == "" {
		item.SchemaVersion = manifestSchemaVersion
	}

	normalized := false
	if len(dataModelsRaw) > 0 {
		item.DataModels, normalized, err = parseManifestDataModels(dataModelsRaw)
		if err != nil {
			return manifest{}, nil, false, fmt.Errorf("parse dataModels: %w", err)
		}
	}
	if len(relationsRaw) > 0 {
		var relationNormalized bool
		item.Relations, relationNormalized, err = parseManifestRelations(relationsRaw)
		if err != nil {
			return manifest{}, nil, false, fmt.Errorf("parse relations: %w", err)
		}
		normalized = normalized || relationNormalized
	}
	if len(queriesRaw) > 0 {
		if err := json.Unmarshal(queriesRaw, &item.Queries); err != nil {
			return manifest{}, nil, false, fmt.Errorf("parse queries: %w", err)
		}
	}
	if len(actionsRaw) > 0 {
		var actionNormalized bool
		item.Actions, actionNormalized, err = parseManifestActions(actionsRaw)
		if err != nil {
			return manifest{}, nil, false, fmt.Errorf("parse actions: %w", err)
		}
		normalized = normalized || actionNormalized
	}
	if appDir == "" {
		appDir = "."
	}
	normalizedItem, actionNormalized, err := normalizeManifestActions(filepath.Clean(appDir), item)
	if err != nil {
		return manifest{}, nil, false, fmt.Errorf("normalize actions: %w", err)
	}
	item = normalizedItem
	normalized = normalized || actionNormalized

	normalizedRaw, err := marshalManifest(item)
	if err != nil {
		return manifest{}, nil, false, err
	}
	changed := normalized || !jsonEqual(raw, normalizedRaw)
	return item, normalizedRaw, changed, nil
}

func parseManifestActions(raw json.RawMessage) ([]string, bool, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, false, nil
	}

	var actions []string
	if err := json.Unmarshal(raw, &actions); err == nil {
		return normalizeStringSlice(actions), false, nil
	}

	var flex []manifestActionFlexible
	if err := json.Unmarshal(raw, &flex); err == nil {
		out := make([]string, 0, len(flex))
		for _, item := range flex {
			mode := strings.TrimSpace(item.Mode)
			if mode != "" && mode != "button_controlled" && mode != "controlled" {
				continue
			}
			key := firstNonEmpty(item.Key, item.Action, item.Name)
			if key != "" {
				out = append(out, key)
			}
		}
		return normalizeStringSlice(out), true, nil
	}

	return nil, false, fmt.Errorf("json: cannot unmarshal actions into supported schema")
}

func parseManifestDataModels(raw json.RawMessage) ([]dao.DataModel, bool, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, false, nil
	}

	var models []dao.DataModel
	if err := json.Unmarshal(raw, &models); err == nil {
		if !hasStructuredDataModels(models) {
			return nil, false, fmt.Errorf("structured dataModels must declare at least one model field")
		}
		return models, false, nil
	}

	var flex []manifestDataModelFlexible
	if err := json.Unmarshal(raw, &flex); err == nil {
		converted, convErr := convertFlexibleDataModels(flex)
		if convErr != nil {
			return nil, false, convErr
		}
		return converted, true, nil
	}

	var hints []string
	if err := json.Unmarshal(raw, &hints); err == nil {
		return nil, false, fmt.Errorf("dataModels must use structured schema; hint strings belong in dataModelHints")
	}

	return nil, false, fmt.Errorf("json: cannot unmarshal dataModels into supported schema")
}

func hasStructuredDataModels(models []dao.DataModel) bool {
	if len(models) == 0 {
		return true
	}
	for _, model := range models {
		if strings.TrimSpace(model.Name) == "" {
			return false
		}
		if len(model.Fields) == 0 {
			return false
		}
	}
	return true
}

func convertFlexibleDataModels(items []manifestDataModelFlexible) ([]dao.DataModel, error) {
	out := make([]dao.DataModel, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		fields, err := parseFlexibleModelFields(item.Fields, item.Validations)
		if err != nil {
			return nil, fmt.Errorf("model %q: %w", name, err)
		}
		if len(fields) == 0 {
			return nil, fmt.Errorf("model %q: fields are required", name)
		}
		out = append(out, dao.DataModel{
			Name:    name,
			Label:   strings.TrimSpace(item.Label),
			Fields:  fields,
			Indexes: normalizeStringSlice(item.Indexes),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no data models found")
	}
	return out, nil
}

func parseFlexibleModelFields(raw json.RawMessage, validations map[string]map[string]any) ([]dao.DataField, error) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return nil, nil
	}

	var arrayFields []dao.DataField
	if err := json.Unmarshal(raw, &arrayFields); err == nil && len(arrayFields) > 0 {
		applyValidationsToFields(arrayFields, validations)
		return arrayFields, nil
	}

	var typeMap map[string]string
	if err := json.Unmarshal(raw, &typeMap); err == nil && len(typeMap) > 0 {
		names := make([]string, 0, len(typeMap))
		for name := range typeMap {
			names = append(names, name)
		}
		sort.Strings(names)

		fields := make([]dao.DataField, 0, len(names))
		for _, name := range names {
			field := dao.DataField{
				Name: strings.TrimSpace(name),
				Type: strings.TrimSpace(typeMap[name]),
			}
			applyValidationToField(&field, validations[name])
			fields = append(fields, field)
		}
		return fields, nil
	}

	return nil, fmt.Errorf("unsupported fields format")
}

func applyValidationsToFields(fields []dao.DataField, validations map[string]map[string]any) {
	for index := range fields {
		applyValidationToField(&fields[index], validations[fields[index].Name])
	}
}

func applyValidationToField(field *dao.DataField, rules map[string]any) {
	if field == nil || len(rules) == 0 {
		return
	}
	if required, ok := rules["required"].(bool); ok && required {
		field.Required = true
	}
	if maxLength, ok := asInt(rules["maxLength"]); ok {
		field.MaxLength = maxLength
	}
	if minValue, ok := asFloat(rules["min"]); ok {
		field.Min = &minValue
	}
	if maxValue, ok := asFloat(rules["max"]); ok {
		field.Max = &maxValue
	}
	if enumValues := asStringSlice(rules["enum"]); len(enumValues) > 0 {
		field.Type = "enum"
		field.Values = enumValues
	}
}

func parseManifestRelations(raw json.RawMessage) ([]dao.DataRelation, bool, error) {
	var relations []dao.DataRelation
	if err := json.Unmarshal(raw, &relations); err == nil {
		if relationsUseQualifiedFields(relations) {
			return relations, false, nil
		}
	}

	var flex []manifestRelationFlexible
	if err := json.Unmarshal(raw, &flex); err != nil {
		return nil, false, err
	}

	out := make([]dao.DataRelation, 0, len(flex))
	for _, item := range flex {
		relation, err := normalizeFlexibleRelation(item)
		if err != nil {
			return nil, false, err
		}
		out = append(out, relation)
	}
	return out, true, nil
}

func relationsUseQualifiedFields(relations []dao.DataRelation) bool {
	if len(relations) == 0 {
		return true
	}
	for _, relation := range relations {
		if strings.TrimSpace(relation.Name) == "" {
			return false
		}
		if !strings.Contains(relation.From, ".") || !strings.Contains(relation.To, ".") {
			return false
		}
	}
	return true
}

func normalizeFlexibleRelation(item manifestRelationFlexible) (dao.DataRelation, error) {
	name := strings.TrimSpace(item.Name)
	from := strings.TrimSpace(item.From)
	to := strings.TrimSpace(item.To)
	if name == "" || from == "" || to == "" {
		return dao.DataRelation{}, fmt.Errorf("relation name, from and to are required")
	}

	relationType := strings.TrimSpace(item.Type)
	if relationType == "" {
		relationType = "manyToOne"
	}

	fromField := "id"
	toField := strings.TrimSpace(item.ForeignKey)
	if toField == "" {
		toField = snakeCaseModelRef(from) + "_id"
	}

	switch strings.ToLower(relationType) {
	case "has_many", "hasmany", "one_to_many", "onetomany":
		return dao.DataRelation{
			Name: name,
			From: from + "." + fromField,
			To:   to + "." + toField,
			Type: "oneToMany",
		}, nil
	case "belongs_to", "belongsto", "many_to_one", "manytoone":
		return dao.DataRelation{
			Name: name,
			From: from + "." + toField,
			To:   to + "." + fromField,
			Type: "manyToOne",
		}, nil
	default:
		if strings.Contains(from, ".") && strings.Contains(to, ".") {
			return dao.DataRelation{
				Name: name,
				From: from,
				To:   to,
				Type: relationType,
			}, nil
		}
		return dao.DataRelation{
			Name: name,
			From: from + "." + fromField,
			To:   to + "." + toField,
			Type: relationType,
		}, nil
	}
}

func marshalManifest(item manifest) ([]byte, error) {
	item.SchemaVersion = firstNonEmpty(strings.TrimSpace(item.SchemaVersion), manifestSchemaVersion)
	item.Actions = normalizeDeclaredActions(item.Actions)
	item.DataModelHints = normalizeStringSlice(item.DataModelHints)

	raw, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal normalized manifest: %w", err)
	}
	return append(raw, '\n'), nil
}

func normalizeDeclaredActions(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return []string{}
	}
	return out
}

func jsonEqual(left []byte, right []byte) bool {
	return bytes.Equal(bytes.TrimSpace(left), bytes.TrimSpace(right))
}

func normalizeStringSlice(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func snakeCaseModelRef(modelName string) string {
	if modelName == "" {
		return ""
	}
	var builder strings.Builder
	for index, r := range modelName {
		if r >= 'A' && r <= 'Z' {
			if index > 0 {
				builder.WriteByte('_')
			}
			builder.WriteRune(r + ('a' - 'A'))
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

func asInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	default:
		return 0, false
	}
}

func asFloat(value any) (float64, bool) {
	switch typed := value.(type) {
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case float64:
		return typed, true
	default:
		return 0, false
	}
}

func asStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return normalizeStringSlice(typed)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}
