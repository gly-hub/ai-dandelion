package config

import (
	"github.com/gly-hub/quickgo"
	"github.com/gly-hub/quickgo/db/gorm"
	"github.com/gly-hub/quickgo/db/redis"
	"github.com/gly-hub/quickgo/tracing"
)

type Config struct {
	AppConfig        quickgo.AppConfig        `json:"app" yaml:"app"`
	LoggerConfig     quickgo.LoggerConfig     `json:"logger" yaml:"logger"`
	GrpcServerConfig quickgo.GrpcServerConfig `json:"grpcServer" yaml:"grpcServer"`
	GrpcClientConfig quickgo.GrpcClientConfig `json:"grpcClient" yaml:"grpcClient"`
	GormConfig       gorm.GormManagerConfig   `json:"gorm" yaml:"gorm"`
	RedisConfig      redis.RedisManagerConfig `json:"redis" yaml:"redis"`
	TracingConfig    tracing.Config           `json:"tracing" yaml:"tracing"`
	AgentConfig      AgentConfig              `json:"agent" yaml:"agent"`
}

type AgentConfig struct {
	CWD             string           `json:"cwd" yaml:"cwd"`
	SystemPrompt    string           `json:"system_prompt" yaml:"system_prompt"`
	Model           string           `json:"model" yaml:"model"`
	CLIPath         string           `json:"cli_path" yaml:"cli_path"`
	PermissionMode  string           `json:"permission_mode" yaml:"permission_mode"`
	MaxTurns        int              `json:"max_turns" yaml:"max_turns"`
	AuthToken       string           `json:"auth_token" yaml:"auth_token"`
	BaseURL         string           `json:"base_url" yaml:"base_url"`
	SkillStorageDir string           `json:"skill_storage_dir" yaml:"skill_storage_dir"`
	ThinkConfig     AgentThinkConfig `json:"think_config" yaml:"think_config"`
}

type AgentThinkConfig struct {
	Mode              string `json:"mode" yaml:"mode"`
	BudgetTokens      int    `json:"budget_tokens" yaml:"budget_tokens"`
	Display           string `json:"display" yaml:"display"`
	MaxThinkingTokens int    `json:"max_thinking_tokens" yaml:"max_thinking_tokens"`
}
