package logic

import (
	"encoding/json"
	"testing"
)

func TestNormalizeOptionValueJSON(t *testing.T) {
	value, err := normalizeOptionValueJSON(`[
  {"value":" chengdu ","label":" 成都 ","extra":"kept"},
  {"value":"shanghai","label":"上海"}
]`)
	if err != nil {
		t.Fatalf("normalizeOptionValueJSON() error = %v", err)
	}
	var items []map[string]any
	if err := json.Unmarshal([]byte(value), &items); err != nil {
		t.Fatalf("decode normalized value: %v", err)
	}
	if len(items) != 2 || items[0]["value"] != "chengdu" || items[0]["label"] != "成都" || items[0]["extra"] != "kept" {
		t.Fatalf("normalized items = %#v", items)
	}
}

func TestNormalizeOptionValueJSONRejectsInvalidOptions(t *testing.T) {
	cases := []string{
		`{}`,
		`[]`,
		`[{"value":"chengdu"}]`,
		`[{"value":"chengdu","label":"成都"},{"value":"chengdu","label":"成都二"}]`,
	}
	for _, input := range cases {
		if _, err := normalizeOptionValueJSON(input); err == nil {
			t.Fatalf("normalizeOptionValueJSON(%s) error = nil", input)
		}
	}
}

func TestPublicConfigRequestKeys(t *testing.T) {
	keys, requested, err := publicConfigRequestKeys([]byte(`{"action":"__platform.config.get","configKeys":["country","status"]}`))
	if err != nil || !requested || len(keys) != 2 || keys[0] != "country" {
		t.Fatalf("publicConfigRequestKeys() = %#v, %v, %v", keys, requested, err)
	}
	if _, requested, err := publicConfigRequestKeys([]byte(`{"action":"list"}`)); err != nil || requested {
		t.Fatalf("ordinary application action = requested:%v err:%v", requested, err)
	}
	if _, requested, err := publicConfigRequestKeys([]byte(`{"action":"__platform.config.get"}`)); !requested || err == nil {
		t.Fatalf("missing config keys = requested:%v err:%v", requested, err)
	}
}
