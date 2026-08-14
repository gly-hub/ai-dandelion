package generatedapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/team-dandelion/ai-dandelion/func-operation/internal/dao"
)

func TestDecodeManifestAcceptsLegacyStringDataModels(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"id": "07750d4e-d971-4028-b79f-93c71b6b0474",
		"name": "图书管理",
		"version": "v0.1.0",
		"dataModels": ["### 3.1 图书（Book）"]
	}`)

	if _, _, _, err := decodeManifest(raw); err == nil {
		t.Fatalf("decodeManifest() expected error for legacy string dataModels")
	}
}

func TestDecodeManifestParsesStructuredDataModels(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"id": "07750d4e-d971-4028-b79f-93c71b6b0474",
		"name": "图书管理",
		"actions": ["create", "delete"],
		"dataModels": [{
			"name": "Book",
			"label": "图书",
			"fields": [
				{ "name": "title", "type": "string", "required": true, "maxLength": 120 }
			]
		}]
	}`)

	item, _, changed, err := decodeManifest(raw)
	if err != nil {
		t.Fatalf("decodeManifest() error = %v", err)
	}
	if !changed {
		t.Fatalf("structured manifest without schemaVersion should be normalized")
	}
	if len(item.DataModels) != 1 || item.DataModels[0].Name != "Book" {
		t.Fatalf("unexpected data models: %#v", item.DataModels)
	}
	if len(item.Actions) != 2 || item.Actions[0] != "create" {
		t.Fatalf("unexpected actions: %#v", item.Actions)
	}
	if len(item.DataModels[0].Fields) != 1 || item.DataModels[0].Fields[0].Name != "title" {
		t.Fatalf("unexpected fields: %#v", item.DataModels[0].Fields)
	}
}

func TestDecodeManifestNormalizesObjectActions(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"id": "28b15ba6-8b83-47ef-b148-c90570008f4b",
		"name": "班级管理",
		"actions": [
			{ "key": "class_create", "label": "新建班级", "mode": "button_controlled" },
			{ "key": "class_update", "label": "编辑班级", "mode": "button_controlled" },
			{ "key": "class_list", "label": "列表查询", "mode": "read_default" }
		],
		"dataModels": [{
			"name": "class",
			"label": "班级",
			"fields": [
				{ "name": "name", "type": "string", "required": true, "maxLength": 30 }
			]
		}]
	}`)

	item, normalizedRaw, changed, err := decodeManifest(raw)
	if err != nil {
		t.Fatalf("decodeManifest() error = %v", err)
	}
	if !changed {
		t.Fatalf("object actions should be normalized")
	}
	if strings.Join(item.Actions, ",") != "class_create,class_update" {
		t.Fatalf("unexpected actions: %#v", item.Actions)
	}
	normalizedText := string(normalizedRaw)
	if strings.Contains(normalizedText, "\"key\"") || strings.Contains(normalizedText, "class_list") {
		t.Fatalf("normalized manifest should keep actions as controlled key array only: %s", normalizedText)
	}
}

func TestDecodeManifestParsesCompactDataModelsAndRelations(t *testing.T) {
	t.Parallel()

	manifestPath := filepath.Join(
		"..", "..", "..", "..", "generated_apps",
		"07750d4e-d971-4028-b79f-93c71b6b0474", "manifest.json",
	)
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Skipf("sample manifest unavailable: %v", err)
	}

	item, normalizedRaw, changed, err := decodeManifest(raw)
	if err != nil {
		t.Fatalf("decodeManifest() error = %v", err)
	}
	if !changed {
		t.Fatalf("compact manifest should be normalized")
	}
	if len(item.DataModels) != 2 {
		t.Fatalf("expected 2 data models, got %#v", item.DataModels)
	}
	titleField := findField(item.DataModels[0].Fields, "title")
	if titleField == nil || !titleField.Required || titleField.MaxLength != 120 {
		t.Fatalf("unexpected title field: %#v", titleField)
	}
	if len(item.Relations) != 1 || item.Relations[0].From != "Book.id" {
		t.Fatalf("unexpected relation: %#v", item.Relations)
	}
	if !strings.Contains(string(normalizedRaw), "\"schemaVersion\": \"v2\"") {
		t.Fatalf("normalized manifest missing schemaVersion: %s", string(normalizedRaw))
	}
	if strings.Contains(string(normalizedRaw), "\"validations\"") {
		t.Fatalf("normalized manifest should not keep compact validations: %s", string(normalizedRaw))
	}
}

func findField(fields []dao.DataField, name string) *dao.DataField {
	for index := range fields {
		if fields[index].Name == name {
			return &fields[index]
		}
	}
	return nil
}
