package logic

import (
	"context"
	"os"
	"strings"

	"github.com/team-dandelion/ai-dandelion/ai-agent/config"
	"github.com/team-dandelion/ai-dandelion/ai-agent/internal/dao"
	"github.com/team-dandelion/ai-dandelion/ai-agent/internal/model"
	"github.com/team-dandelion/ai-dandelion/toolbox/agent"
	"gorm.io/gorm"
)

type RunnerFactory struct {
	base           config.AgentConfig
	agentConfigDao *dao.AgentConfig
}

func NewRunnerFactory(base config.AgentConfig, agentConfigDao *dao.AgentConfig) *RunnerFactory {
	return &RunnerFactory{base: base, agentConfigDao: agentConfigDao}
}

func (f *RunnerFactory) DefaultRunner() agent.Runner {
	return f.build(context.Background(), nil, AgentRuntimeOverride{})
}

func (f *RunnerFactory) RunnerFor(ctx context.Context, record *model.AgentModel) agent.Runner {
	return f.RunnerForConfig(ctx, record, AgentRuntimeOverride{})
}

func (f *RunnerFactory) RunnerForConfig(ctx context.Context, record *model.AgentModel, override AgentRuntimeOverride) agent.Runner {
	return f.build(ctx, record, override)
}

func (f *RunnerFactory) build(ctx context.Context, record *model.AgentModel, override AgentRuntimeOverride) agent.Runner {
	cfg := f.base
	if f.agentConfigDao != nil {
		if sysCfg, err := f.agentConfigDao.Get(ctx); err == nil && sysCfg != nil {
			if sysCfg.SystemPrompt != "" {
				cfg.SystemPrompt = sysCfg.SystemPrompt
			}
			if sysCfg.PermissionMode != "" {
				cfg.PermissionMode = sysCfg.PermissionMode
			}
			if sysCfg.MaxTurns > 0 {
				cfg.MaxTurns = sysCfg.MaxTurns
			}
		} else if err != nil && err != gorm.ErrRecordNotFound {
			_ = err
		}
	}
	if override.SystemPrompt != "" {
		cfg.SystemPrompt = override.SystemPrompt
	}
	if override.PermissionMode != "" {
		cfg.PermissionMode = override.PermissionMode
	}
	if override.MaxTurns > 0 {
		cfg.MaxTurns = override.MaxTurns
	}
	if record != nil {
		if record.Model != "" {
			cfg.Model = record.Model
		}
		if record.BaseURL != "" {
			cfg.BaseURL = record.BaseURL
		}
		if record.AuthToken != "" {
			cfg.AuthToken = record.AuthToken
		}
		cfg.ThinkConfig = config.AgentThinkConfig{
			Mode:              record.ThinkMode,
			BudgetTokens:      record.ThinkBudgetTokens,
			Display:           record.ThinkDisplay,
			MaxThinkingTokens: record.MaxThinkingTokens,
		}
	}
	return newAgentRunner(cfg, record != nil)
}

type AgentRuntimeOverride struct {
	SystemPrompt   string
	PermissionMode string
	MaxTurns       int
	Skills         []string
}

func newAgentRunner(cfg config.AgentConfig, preferConfiguredCredentials bool) agent.Runner {
	if strings.TrimSpace(cfg.CWD) == "" {
		return nil
	}

	env := make(map[string]string)
	// A selected model carries its own provider settings. It must not inherit a
	// process-wide endpoint or token from a different provider channel.
	authToken := resolveProviderValue(cfg.AuthToken, "ANTHROPIC_AUTH_TOKEN", preferConfiguredCredentials)
	if authToken != "" {
		env["ANTHROPIC_AUTH_TOKEN"] = authToken
	}
	baseURL := resolveProviderValue(cfg.BaseURL, "ANTHROPIC_BASE_URL", preferConfiguredCredentials)
	if baseURL != "" {
		env["ANTHROPIC_BASE_URL"] = baseURL
	}

	return agent.NewClaudeRunner(agent.Config{
		CWD:               cfg.CWD,
		SystemPrompt:      cfg.SystemPrompt,
		Model:             cfg.Model,
		CLIPath:           cfg.CLIPath,
		PermissionMode:    cfg.PermissionMode,
		Env:               env,
		MaxTurns:          cfg.MaxTurns,
		Thinking:          newThinkingConfig(cfg.ThinkConfig),
		MaxThinkingTokens: cfg.ThinkConfig.MaxThinkingTokens,
	})
}

func resolveProviderValue(configured, environmentKey string, preferConfigured bool) string {
	configured = strings.TrimSpace(configured)
	if preferConfigured && configured != "" {
		return configured
	}
	if value := strings.TrimSpace(os.Getenv(environmentKey)); value != "" {
		return value
	}
	return configured
}

func newThinkingConfig(cfg config.AgentThinkConfig) *agent.ThinkingConfig {
	if cfg.Mode == "" && cfg.Display == "" && cfg.BudgetTokens == 0 {
		return nil
	}
	return &agent.ThinkingConfig{
		Mode:         cfg.Mode,
		BudgetTokens: cfg.BudgetTokens,
		Display:      cfg.Display,
	}
}
