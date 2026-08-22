package logic

import (
	"context"
	"testing"

	"github.com/gly-hub/ai-dandelion/system/internal/dao"
	"github.com/gly-hub/ai-dandelion/system/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEnsureSeedAgentSessionConfigsUpdatesOnlyLegacyGenerationPrompt(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.AgentSessionConfig{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}

	legacy := model.AgentSessionConfig{
		SessionType:  model.AgentSessionConfigTypeFuncGeneration,
		SystemPrompt: legacyFuncGenerationSystemPrompt,
		Enabled:      true,
	}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatalf("create legacy config: %v", err)
	}

	logic := NewAgentSessionConfigLogic(dao.NewAgentSessionConfig(db))
	if err := logic.EnsureSeedAgentSessionConfigs(context.Background()); err != nil {
		t.Fatalf("seed configs: %v", err)
	}

	var migrated model.AgentSessionConfig
	if err := db.First(&migrated, "session_type = ?", model.AgentSessionConfigTypeFuncGeneration).Error; err != nil {
		t.Fatalf("load migrated config: %v", err)
	}
	if migrated.SystemPrompt != funcGenerationSystemPrompt {
		t.Fatalf("legacy prompt was not migrated: %q", migrated.SystemPrompt)
	}

	migrated.SystemPrompt = "用户自定义的页面生成提示词"
	if err := db.Save(&migrated).Error; err != nil {
		t.Fatalf("save custom prompt: %v", err)
	}
	if err := logic.EnsureSeedAgentSessionConfigs(context.Background()); err != nil {
		t.Fatalf("reseed configs: %v", err)
	}
	if err := db.First(&migrated, "session_type = ?", model.AgentSessionConfigTypeFuncGeneration).Error; err != nil {
		t.Fatalf("reload custom config: %v", err)
	}
	if migrated.SystemPrompt != "用户自定义的页面生成提示词" {
		t.Fatalf("custom prompt was overwritten: %q", migrated.SystemPrompt)
	}
}
