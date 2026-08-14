package logic

import (
	"context"
	"sync"

	"github.com/team-dandelion/ai-dandelion/ai-agent/internal/dao"
	"github.com/team-dandelion/ai-dandelion/ai-agent/internal/model"
	"github.com/team-dandelion/quickgo/logger"
)

type AgentBotRuntime struct {
	agentBotDao  *dao.AgentBot
	sessionLogic *SessionLogic
	messageLogic *MessageLogic
	agentEngine  *AgentEngine
	skillLogic   *SkillLogic
	mcpLogic     *MCPLogic

	mu      sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
	workers map[string]*agentBotWorkerEntry
	wg      sync.WaitGroup
}

func NewAgentBotRuntime(agentBotDao *dao.AgentBot, sessionLogic *SessionLogic, messageLogic *MessageLogic, agentEngine *AgentEngine, skillLogic *SkillLogic, mcpLogic *MCPLogic) *AgentBotRuntime {
	return &AgentBotRuntime{
		agentBotDao:  agentBotDao,
		sessionLogic: sessionLogic,
		messageLogic: messageLogic,
		agentEngine:  agentEngine,
		skillLogic:   skillLogic,
		mcpLogic:     mcpLogic,
	}
}

func (r *AgentBotRuntime) Start(ctx context.Context) error {
	if r == nil || r.agentBotDao == nil || r.sessionLogic == nil || r.messageLogic == nil || r.agentEngine == nil {
		return nil
	}
	r.mu.Lock()
	if r.cancel != nil {
		r.mu.Unlock()
		return nil
	}
	r.ctx, r.cancel = context.WithCancel(ctx)
	r.workers = make(map[string]*agentBotWorkerEntry)
	r.mu.Unlock()
	return r.reload(ctx)
}

func (r *AgentBotRuntime) Reload(ctx context.Context) error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	started := r.cancel != nil
	r.mu.Unlock()
	if !started {
		return nil
	}
	return r.reload(ctx)
}

func (r *AgentBotRuntime) reload(ctx context.Context) error {
	r.stopWorkers()
	items, err := r.agentBotDao.List(ctx)
	if err != nil {
		return err
	}
	for i := range items {
		item := items[i]
		if item.Bot.Status != model.AgentBotStatusEnabled {
			continue
		}
		for j := range item.Channels {
			channel := item.Channels[j]
			if channel.Status != model.AgentBotChannelStatusEnabled {
				continue
			}
			r.startChannelWorker(ctx, item, channel)
		}
	}
	return nil
}

func (r *AgentBotRuntime) Stop() {
	if r == nil {
		return
	}
	r.stopWorkers()
	r.mu.Lock()
	cancel := r.cancel
	r.cancel = nil
	r.ctx = nil
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	r.wg.Wait()
}

func (r *AgentBotRuntime) stopWorkers() {
	r.mu.Lock()
	workers := r.workers
	r.workers = make(map[string]*agentBotWorkerEntry)
	r.mu.Unlock()
	for _, worker := range workers {
		worker.stop()
	}
}

type agentBotChannelWorker interface {
	Start(context.Context) error
	Stop()
}

type agentBotWorkerEntry struct {
	cancel context.CancelFunc
	worker agentBotChannelWorker
}

func (w *agentBotWorkerEntry) stop() {
	if w == nil {
		return
	}
	if w.cancel != nil {
		w.cancel()
	}
	if w.worker != nil {
		w.worker.Stop()
	}
}

func (r *AgentBotRuntime) runtimeContext() (context.Context, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ctx == nil || r.cancel == nil {
		return nil, false
	}
	return r.ctx, true
}

func (r *AgentBotRuntime) setWorker(key string, worker *agentBotWorkerEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.workers == nil {
		r.workers = make(map[string]*agentBotWorkerEntry)
	}
	r.workers[key] = worker
}

func (r *AgentBotRuntime) deleteWorker(key string, worker *agentBotWorkerEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.workers != nil && r.workers[key] == worker {
		delete(r.workers, key)
	}
}

func (r *AgentBotRuntime) startChannelWorker(ctx context.Context, item dao.AgentBotAggregate, channel model.AgentBotChannel) {
	runCtx, ok := r.runtimeContext()
	if !ok {
		return
	}
	engineConfig, err := r.agentBotEngineConfig(item)
	if err != nil {
		logger.Error(ctx, "skip agent bot %s channel %s: invalid agent config: %v", item.Bot.Name, channel.Name, err)
		return
	}
	workerImpl, key, err := r.newChannelWorker(item, channel, engineConfig)
	if err != nil {
		logger.Error(ctx, "skip agent bot %s channel %s: %v", item.Bot.Name, channel.Name, err)
		return
	}
	if workerImpl == nil {
		return
	}
	workerCtx, cancel := context.WithCancel(runCtx)
	worker := &agentBotWorkerEntry{cancel: cancel, worker: workerImpl}
	r.setWorker(key, worker)

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer r.deleteWorker(key, worker)
		if err := workerImpl.Start(workerCtx); err != nil {
			logger.Error(workerCtx, "start agent bot %s channel %s failed: %v", item.Bot.Name, channel.Name, err)
			return
		}
		<-workerCtx.Done()
		workerImpl.Stop()
	}()
	logger.Info(runCtx, "started agent bot runtime: %s / %s", item.Bot.Name, channel.Name)
}

func (r *AgentBotRuntime) newChannelWorker(
	item dao.AgentBotAggregate,
	channel model.AgentBotChannel,
	engineConfig AgentEngineRunConfig,
) (agentBotChannelWorker, string, error) {
	switch channel.Channel {
	case "wecom":
		return newWeComAgentBotWorker(r, item, channel, engineConfig)
	default:
		return nil, "", nil
	}
}

func (r *AgentBotRuntime) agentBotEngineConfig(item dao.AgentBotAggregate) (AgentEngineRunConfig, error) {
	config := AgentEngineRunConfig{
		ModelID:        item.Bot.ModelID,
		SystemPrompt:   item.Bot.SystemPrompt,
		PermissionMode: item.Bot.PermissionMode,
		MaxTurns:       item.Bot.MaxTurns,
	}
	mcpIDs := make([]string, 0)
	skillIDs := make([]string, 0)
	for _, capability := range item.Capabilities {
		if capability.Enabled != model.AgentBotCapabilityEnabled {
			continue
		}
		switch capability.CapabilityType {
		case model.AgentBotCapabilityTypeSkill:
			config.Skills = append(config.Skills, capability.CapabilityID)
			skillIDs = append(skillIDs, capability.CapabilityID)
		case model.AgentBotCapabilityTypeMCP:
			mcpIDs = append(mcpIDs, capability.CapabilityID)
		}
	}
	addDirs, err := r.skillLogic.ResolveLinkedSkillDirs(skillIDs)
	if err != nil {
		return config, err
	}
	mcpServers, err := r.mcpLogic.ResolveLinkedMCPServers(mcpIDs)
	if err != nil {
		return config, err
	}
	config.AddDirs = addDirs
	config.MCPServers = mcpServers
	return config, nil
}
