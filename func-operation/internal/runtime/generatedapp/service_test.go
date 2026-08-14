package generatedapp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/team-dandelion/ai-dandelion/func-operation/internal/dao"
	"github.com/team-dandelion/ai-dandelion/func-operation/internal/model"
	"github.com/tetratelabs/wazero"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLoadApprovedArtifactsSkipsUnapprovedDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "3658889f-3b80-45a9-88e5-8de80fa287b5"), 0o755); err != nil {
		t.Fatalf("mkdir app dir: %v", err)
	}
	service := &Service{
		rootDir:  root,
		store:    dao.NewGeneratedApp(nil),
		runtime:  wazero.NewRuntime(context.Background()),
		compiled: map[string]wazero.CompiledModule{},
		apps:     map[string]model.GeneratedApp{},
	}
	defer service.runtime.Close(context.Background())

	if err := service.LoadApprovedArtifacts(context.Background(), map[string]string{}); err != nil {
		t.Fatalf("LoadApprovedArtifacts() error = %v", err)
	}
	if len(service.apps) != 0 || len(service.compiled) != 0 {
		t.Fatalf("LoadApprovedArtifacts() should skip unapproved app dirs, apps=%d compiled=%d", len(service.apps), len(service.compiled))
	}
}

func TestCreateAppScaffoldDoesNotInjectDefaultDataModels(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	appID := "3658889f-3b80-45a9-88e5-8de80fa287b5"
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.GeneratedApp{}, &model.AppRecord{}); err != nil {
		t.Fatalf("migrate generated app tables: %v", err)
	}
	templateDir := filepath.Join(root, templateAppID)
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("mkdir template dir: %v", err)
	}
	minimalWASM := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	if err := os.WriteFile(filepath.Join(templateDir, "backend.wasm"), minimalWASM, 0o644); err != nil {
		t.Fatalf("write template wasm: %v", err)
	}

	service := &Service{
		rootDir:  root,
		store:    dao.NewGeneratedApp(db),
		runtime:  wazero.NewRuntime(context.Background()),
		compiled: map[string]wazero.CompiledModule{},
		apps:     map[string]model.GeneratedApp{},
	}
	defer service.runtime.Close(context.Background())
	if _, err := service.CreateAppScaffold(context.Background(), ScaffoldInput{
		AppID:       appID,
		Name:        "Test App",
		Description: "for manifest assertions",
	}); err != nil {
		t.Fatalf("CreateAppScaffold() error = %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(root, appID, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	if got := manifest["schemaVersion"]; got != manifestSchemaVersion {
		t.Fatalf("expected schemaVersion %q, got %#v", manifestSchemaVersion, got)
	}
	actionsValue, exists := manifest["actions"]
	if !exists {
		t.Fatalf("expected manifest actions field")
	}
	actions, ok := actionsValue.([]any)
	if !ok {
		t.Fatalf("manifest actions has unexpected type %T", actionsValue)
	}
	if len(actions) != 0 {
		t.Fatalf("expected empty default actions, got %d", len(actions))
	}
	modelsValue, exists := manifest["dataModels"]
	if !exists || modelsValue == nil {
		return
	}
	models, ok := modelsValue.([]any)
	if !ok {
		t.Fatalf("manifest dataModels has unexpected type %T", modelsValue)
	}
	if len(models) != 0 {
		t.Fatalf("expected no default data models, got %d", len(models))
	}
}

func TestLoadManifestFileNormalizesCompactManifest(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	appID := "07750d4e-d971-4028-b79f-93c71b6b0474"
	appDir := filepath.Join(root, appID)
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("mkdir app dir: %v", err)
	}

	sampleManifest, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "generated_apps", appID, "manifest.json"))
	if err != nil {
		t.Skipf("read sample manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "manifest.json"), sampleManifest, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	minimalWASM := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	if err := os.WriteFile(filepath.Join(appDir, "backend.wasm"), minimalWASM, 0o644); err != nil {
		t.Fatalf("write wasm: %v", err)
	}

	item, err := loadManifestFile(filepath.Join(appDir, "manifest.json"))
	if err != nil {
		t.Fatalf("loadManifestFile() error = %v", err)
	}
	if len(item.DataModels) != 2 {
		t.Fatalf("expected 2 data models, got %#v", item.DataModels)
	}

	normalizedRaw, err := os.ReadFile(filepath.Join(appDir, "manifest.json"))
	if err != nil {
		t.Fatalf("read normalized manifest: %v", err)
	}
	normalizedText := string(normalizedRaw)
	if !strings.Contains(normalizedText, "\"schemaVersion\": \"v2\"") {
		t.Fatalf("normalized manifest missing schemaVersion: %s", normalizedText)
	}
	if strings.Contains(normalizedText, "\"validations\"") {
		t.Fatalf("normalized manifest should not keep compact validations: %s", normalizedText)
	}
	if !strings.Contains(normalizedText, "\"from\": \"Book.id\"") {
		t.Fatalf("normalized manifest should qualify relation fields: %s", normalizedText)
	}
}

func TestLoadManifestFileInfersPermissionActionsFromCode(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	appID := "43007d4c-c8cc-4f05-9413-c2b5412a8840"
	appDir := filepath.Join(root, appID)
	if err := os.MkdirAll(filepath.Join(appDir, "frontend"), 0o755); err != nil {
		t.Fatalf("mkdir frontend dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(appDir, "backend"), 0o755); err != nil {
		t.Fatalf("mkdir backend dir: %v", err)
	}

	files := []string{"manifest.json", "frontend/api.js", "backend/main.go"}
	for _, file := range files {
		source := filepath.Join("..", "..", "..", "..", "generated_apps", appID, file)
		raw, err := os.ReadFile(source)
		if err != nil {
			t.Skipf("read sample file %s: %v", file, err)
		}
		if err := os.WriteFile(filepath.Join(appDir, filepath.FromSlash(file)), raw, 0o644); err != nil {
			t.Fatalf("write sample file %s: %v", file, err)
		}
	}

	item, err := loadManifestFile(filepath.Join(appDir, "manifest.json"))
	if err != nil {
		t.Fatalf("loadManifestFile() error = %v", err)
	}
	want := []string{
		"book_create",
		"book_delete",
		"book_update",
		"borrowing_borrow",
		"borrowing_return",
		"copy_create",
	}
	if strings.Join(item.Actions, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected inferred actions: got %v want %v", item.Actions, want)
	}

	normalizedRaw, err := os.ReadFile(filepath.Join(appDir, "manifest.json"))
	if err != nil {
		t.Fatalf("read normalized manifest: %v", err)
	}
	normalizedText := string(normalizedRaw)
	for _, action := range want {
		if !strings.Contains(normalizedText, "\""+action+"\"") {
			t.Fatalf("normalized manifest missing inferred action %q: %s", action, normalizedText)
		}
	}
	if strings.Contains(normalizedText, "\"book_list\"") {
		t.Fatalf("normalized manifest should not include readonly action: %s", normalizedText)
	}
}
