package logic

import (
	"context"
	"testing"

	"github.com/team-dandelion/ai-dandelion/func-operation/internal/dao"
	"github.com/team-dandelion/ai-dandelion/func-operation/internal/model"
	funcoperation "github.com/team-dandelion/ai-dandelion/proto/func-operation"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUploadPublicConfigRequiresBoundKeyAndCreatesVersion(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.PublicConfig{}, &model.PublicConfigVersion{}, &model.PublicConfigImportKey{}); err != nil {
		t.Fatal(err)
	}
	key, keyHash, err := newSwaggerImportKey()
	if err != nil {
		t.Fatal(err)
	}
	store := dao.NewPublicConfig(db)
	logic := NewPublicConfigLogic(store, nil)
	if _, err := logic.Import(context.Background(), &funcoperation.ImportPublicConfigsReq{ApiKey: key, ConfigsJson: `{"country":[{"value":"us","label":"美国"}]}`}); err == nil {
		t.Fatal("upload accepted before a global import key was created")
	}
	config := &model.PublicConfig{UUID: "config-id", ConfigKey: "country", Name: "国家", ValueJSON: `[{"value":"cn","label":"中国"}]`, Version: 1}
	version := &model.PublicConfigVersion{UUID: "version-id", ConfigID: config.UUID, ConfigKey: config.ConfigKey, Version: 1, ValueJSON: config.ValueJSON, Source: publicConfigSourceCreate}
	if err := store.Create(context.Background(), config, version); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertImportKey(context.Background(), keyHash, "user", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := logic.Import(context.Background(), &funcoperation.ImportPublicConfigsReq{ApiKey: "wrong", ConfigsJson: `{"country":[{"value":"us","label":"美国"}]}`}); err == nil {
		t.Fatal("upload accepted wrong key")
	}
	updated, err := logic.Import(context.Background(), &funcoperation.ImportPublicConfigsReq{ApiKey: key, ConfigsJson: `{"country":[{"value":"us","label":"美国"}],"city":[{"value":"chengdu","label":"成都"}]}`})
	if err != nil || len(updated) != 2 {
		t.Fatalf("upload result = %#v, %v", updated, err)
	}
	country, err := store.Get(context.Background(), "country")
	if err != nil || country.Version != 2 || country.ValueJSON != `[{"label":"美国","value":"us"}]` {
		t.Fatalf("country import = %#v, %v", country, err)
	}
	if _, err := store.Get(context.Background(), "city"); err != nil {
		t.Fatalf("city was not created: %v", err)
	}
}
