package dao

import (
	"context"
	"testing"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPublicConfigUpdateValueCreatesVersionSnapshot(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.PublicConfig{}, &model.PublicConfigVersion{}); err != nil {
		t.Fatalf("migrate public config tables: %v", err)
	}
	dao := NewPublicConfig(db)
	config := &model.PublicConfig{
		UUID: "config-id", ConfigKey: "country", Name: "城市", ValueJSON: `[{"value":"chengdu","label":"成都"}]`,
		Version: 1, CreatedBy: "user-a", UpdatedBy: "user-a", CreatedAt: 1, UpdatedAt: 1,
	}
	initialVersion := &model.PublicConfigVersion{
		UUID: "version-1", ConfigID: config.UUID, ConfigKey: config.ConfigKey, Version: 1,
		ValueJSON: config.ValueJSON, OperatorID: "user-a", Source: "create", CreatedAt: 1,
	}
	if err := dao.Create(context.Background(), config, initialVersion); err != nil {
		t.Fatalf("create config: %v", err)
	}

	updated, err := dao.UpdateValue(context.Background(), "country", "发货城市", "目的地城市", `[{"value":"shanghai","label":"上海"}]`, "user-b", "update", 2)
	if err != nil {
		t.Fatalf("update config: %v", err)
	}
	if updated.Version != 2 || updated.Name != "发货城市" || updated.UpdatedBy != "user-b" {
		t.Fatalf("updated config = %#v", updated)
	}
	versions, err := dao.ListVersions(context.Background(), "country")
	if err != nil {
		t.Fatalf("list versions: %v", err)
	}
	if len(versions) != 2 || versions[0].Version != 2 || versions[0].ValueJSON != updated.ValueJSON {
		t.Fatalf("version snapshots = %#v", versions)
	}
}
