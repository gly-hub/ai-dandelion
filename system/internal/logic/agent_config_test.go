package logic

import (
	"context"
	"testing"

	"github.com/gly-hub/ai-dandelion/system/internal/dao"
	"github.com/gly-hub/ai-dandelion/system/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEnsureSeedAgentConfigMigratesLegacyPrompt(t *testing.T) {
	db, logic := newAgentConfigTestLogic(t)
	legacy := model.AgentSystemConfig{ID: model.AgentSystemConfigID, SystemPrompt: legacyDefaultAgentSystemPrompt}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatalf("create legacy config: %v", err)
	}

	want := "新的通用系统提示词"
	if err := logic.EnsureSeedAgentConfig(context.Background(), want, "bypassPermissions", 20); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	var stored model.AgentSystemConfig
	if err := db.First(&stored, "id = ?", model.AgentSystemConfigID).Error; err != nil {
		t.Fatalf("load config: %v", err)
	}
	if stored.SystemPrompt != want {
		t.Fatalf("legacy prompt was not migrated: %q", stored.SystemPrompt)
	}
}

func TestEnsureSeedAgentConfigPreservesCustomPrompt(t *testing.T) {
	db, logic := newAgentConfigTestLogic(t)
	custom := model.AgentSystemConfig{ID: model.AgentSystemConfigID, SystemPrompt: "用户自定义的系统提示词"}
	if err := db.Create(&custom).Error; err != nil {
		t.Fatalf("create custom config: %v", err)
	}

	if err := logic.EnsureSeedAgentConfig(context.Background(), "新的通用系统提示词", "bypassPermissions", 20); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	var stored model.AgentSystemConfig
	if err := db.First(&stored, "id = ?", model.AgentSystemConfigID).Error; err != nil {
		t.Fatalf("load config: %v", err)
	}
	if stored.SystemPrompt != custom.SystemPrompt {
		t.Fatalf("custom prompt was overwritten: %q", stored.SystemPrompt)
	}
}

func newAgentConfigTestLogic(t *testing.T) (*gorm.DB, *AgentConfigLogic) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.AgentSystemConfig{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	return db, NewAgentConfigLogic(dao.NewAgentConfig(db))
}
