package logic

import (
	"context"
	"testing"

	"github.com/gly-hub/ai-dandelion/system/internal/dao"
	"github.com/gly-hub/ai-dandelion/system/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEnsureSeedAgentSessionConfigsMigratesLegacyPrompts(t *testing.T) {
	tests := []struct {
		name        string
		sessionType string
		legacy      string
		want        string
	}{
		{name: "product", sessionType: model.AgentSessionConfigTypeFuncProduct, legacy: legacyProductSystemPrompt, want: productSystemPrompt},
		{name: "technical", sessionType: model.AgentSessionConfigTypeFuncTechnical, legacy: legacyTechnicalSystemPrompt, want: technicalSystemPrompt},
		{name: "generation legacy", sessionType: model.AgentSessionConfigTypeFuncGeneration, legacy: legacyFuncGenerationSystemPrompt, want: funcGenerationSystemPrompt},
		{name: "generation responsive legacy", sessionType: model.AgentSessionConfigTypeFuncGeneration, legacy: legacyFuncGenerationSystemPromptV2, want: funcGenerationSystemPrompt},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, logic := newAgentSessionConfigTestLogic(t)
			legacy := model.AgentSessionConfig{SessionType: tt.sessionType, SystemPrompt: tt.legacy, Enabled: true}
			if err := db.Create(&legacy).Error; err != nil {
				t.Fatalf("create legacy config: %v", err)
			}

			if err := logic.EnsureSeedAgentSessionConfigs(context.Background()); err != nil {
				t.Fatalf("seed configs: %v", err)
			}
			var migrated model.AgentSessionConfig
			if err := db.First(&migrated, "session_type = ?", tt.sessionType).Error; err != nil {
				t.Fatalf("load migrated config: %v", err)
			}
			if migrated.SystemPrompt != tt.want {
				t.Fatalf("legacy prompt was not migrated: %q", migrated.SystemPrompt)
			}
		})
	}
}

func TestEnsureSeedAgentSessionConfigsPreservesCustomPrompt(t *testing.T) {
	db, logic := newAgentSessionConfigTestLogic(t)
	custom := model.AgentSessionConfig{
		SessionType:  model.AgentSessionConfigTypeFuncProduct,
		SystemPrompt: "用户自定义的产品提示词",
		Enabled:      true,
	}
	if err := db.Create(&custom).Error; err != nil {
		t.Fatalf("create custom config: %v", err)
	}
	if err := logic.EnsureSeedAgentSessionConfigs(context.Background()); err != nil {
		t.Fatalf("seed configs: %v", err)
	}
	var stored model.AgentSessionConfig
	if err := db.First(&stored, "session_type = ?", custom.SessionType).Error; err != nil {
		t.Fatalf("load custom config: %v", err)
	}
	if stored.SystemPrompt != custom.SystemPrompt {
		t.Fatalf("custom prompt was overwritten: %q", stored.SystemPrompt)
	}
}

func newAgentSessionConfigTestLogic(t *testing.T) (*gorm.DB, *AgentSessionConfigLogic) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.AgentSessionConfig{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	return db, NewAgentSessionConfigLogic(dao.NewAgentSessionConfig(db))
}
