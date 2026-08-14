package generatedapp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/dao"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPrepareDraftDataModelsUsesDebugStore(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	appID := "3658889f-3b80-45a9-88e5-8de80fa287b5"
	appDir := filepath.Join(root, appID)
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("mkdir app directory: %v", err)
	}
	manifest := `{"schemaVersion":"v2","id":"3658889f-3b80-45a9-88e5-8de80fa287b5","name":"Teacher","version":"v0.1.0","export":"handle","dataModels":[{"name":"Teacher","fields":[{"name":"name","type":"string"}]}]}`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	productionDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open production database: %v", err)
	}
	debugDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open debug database: %v", err)
	}
	service, err := NewService(ctx, root, dao.NewGeneratedApp(productionDB), WithDraftRuntime(), WithDataStore(dao.NewGeneratedApp(debugDB)))
	if err != nil {
		t.Fatalf("new draft runtime: %v", err)
	}
	defer func() { _ = service.Close(ctx) }()

	if err := service.PrepareDraftDataModels(ctx, appID); err != nil {
		t.Fatalf("prepare draft data models: %v", err)
	}
	forms, err := service.ListDataForms(ctx, appID)
	if err != nil {
		t.Fatalf("list draft data forms: %v", err)
	}
	if len(forms) != 1 || forms[0].TableName == "" {
		t.Fatalf("unexpected draft forms: %#v", forms)
	}
	if !debugDB.Migrator().HasTable(forms[0].TableName) {
		t.Fatalf("debug database is missing table %q", forms[0].TableName)
	}
	if productionDB.Migrator().HasTable(forms[0].TableName) {
		t.Fatalf("production database unexpectedly contains debug table %q", forms[0].TableName)
	}
}
