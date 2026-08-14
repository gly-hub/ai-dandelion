package logic

import (
	"context"
	"errors"

	"github.com/gly-hub/ai-dandelion/toolbox/agent"
)

type AgentEngine struct {
	runnerFactory   runnerFactory
	agentModelLogic *AgentModelLogic
}

type AgentEngineRunConfig struct {
	ModelID         string
	SystemPrompt    string
	PermissionMode  string
	MaxTurns        int
	Skills          []string
	AddDirs         []string
	MCPServers      map[string]agent.MCPServerConfig
	AskUserQuestion agent.AskUserQuestionHandler
	ToolPermission  agent.ToolPermissionHandler
	UserContent     any
}

func NewAgentEngine(runnerFactory runnerFactory, agentModelLogic *AgentModelLogic) *AgentEngine {
	return &AgentEngine{
		runnerFactory:   runnerFactory,
		agentModelLogic: agentModelLogic,
	}
}

func (e *AgentEngine) Stream(
	ctx context.Context,
	agentSessionID string,
	prompt string,
	resume bool,
	config AgentEngineRunConfig,
) (<-chan agent.Event, <-chan error, error) {
	if e == nil || e.runnerFactory == nil || e.runnerFactory.DefaultRunner() == nil {
		return nil, nil, errAgentRunnerNotConfigured
	}
	runner, err := e.resolveRunner(ctx, config)
	if err != nil {
		return nil, nil, err
	}
	if runner == nil {
		return nil, nil, errAgentRunnerNotConfigured
	}
	events, errs := runner.Stream(ctx, agentSessionID, prompt, resume, agent.StreamOptions{
		Skills:          config.Skills,
		AddDirs:         config.AddDirs,
		MCPServers:      config.MCPServers,
		AskUserQuestion: config.AskUserQuestion,
		ToolPermission:  config.ToolPermission,
		UserContent:     config.UserContent,
	})
	return events, errs, nil
}

func (e *AgentEngine) resolveRunner(ctx context.Context, config AgentEngineRunConfig) (agent.Runner, error) {
	override := AgentRuntimeOverride{
		SystemPrompt:   config.SystemPrompt,
		PermissionMode: config.PermissionMode,
		MaxTurns:       config.MaxTurns,
		Skills:         config.Skills,
	}
	if e.agentModelLogic == nil {
		return e.runnerFactory.RunnerForConfig(ctx, nil, override), nil
	}
	runner, err := e.agentModelLogic.ResolveRunner(ctx, e.runnerFactory, config.ModelID, override)
	if err != nil {
		return nil, err
	}
	if runner == nil {
		return nil, errors.New("agent runner is not configured")
	}
	return runner, nil
}
