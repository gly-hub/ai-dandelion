package index

import (
	"context"
	"sync"

	"github.com/team-dandelion/ai-dandelion/ai-agent/global"
	"github.com/team-dandelion/ai-dandelion/ai-agent/internal/dao"
	"github.com/team-dandelion/ai-dandelion/ai-agent/internal/logic"
	"github.com/team-dandelion/ai-dandelion/ai-agent/internal/service"
	aiagent "github.com/team-dandelion/ai-dandelion/proto/ai-agent"
	rpc "google.golang.org/grpc"
)

var (
	agentBotRuntimeMu sync.Mutex
	agentBotRuntime   *logic.AgentBotRuntime
)

func RegisterHandler(s *rpc.Server) {
	aiDandelionDB, err := global.GetApp().GormManager().GetDB("ai-dandelion")
	if err != nil {
		panic(err)
	}

	sessionDao := dao.NewSession(aiDandelionDB)
	messageDao := dao.NewMessage(aiDandelionDB)
	sessionReferenceDao := dao.NewSessionReference(aiDandelionDB)
	agentModelDao := dao.NewAgentModel(aiDandelionDB)
	agentConfigDao := dao.NewAgentConfig(aiDandelionDB)
	agentSessionConfigDao := dao.NewAgentSessionConfig(aiDandelionDB)
	agentBotDao := dao.NewAgentBot(aiDandelionDB)
	if err := logic.EnsureSeedAgentModels(context.Background(), agentModelDao, global.GetConfig().AgentConfig); err != nil {
		panic(err)
	}

	runnerFactory := logic.NewRunnerFactory(global.GetConfig().AgentConfig, agentConfigDao)
	agentModelLogic := logic.NewAgentModelLogic(agentModelDao)
	skillLogic := logic.NewSkillLogic(global.GetConfig().AgentConfig.SkillStorageDir)
	mcpLogic := logic.NewMCPLogic(global.GetConfig().AgentConfig.SkillStorageDir)
	agentEngine := logic.NewAgentEngine(runnerFactory, agentModelLogic)

	sessionLogic := logic.NewSessionLogic(sessionDao, runnerFactory.DefaultRunner())
	messageLogic := logic.NewMessageLogic(sessionDao, messageDao, sessionReferenceDao, runnerFactory, agentModelLogic, agentSessionConfigDao, skillLogic, mcpLogic)
	runtime := logic.NewAgentBotRuntime(agentBotDao, sessionLogic, messageLogic, agentEngine, skillLogic, mcpLogic)
	agentBotLogic := logic.NewAgentBotLogic(agentBotDao, runtime.Reload)
	if err := runtime.Start(context.Background()); err != nil {
		panic(err)
	}
	agentBotRuntimeMu.Lock()
	agentBotRuntime = runtime
	agentBotRuntimeMu.Unlock()

	aiAgentService := service.NewAiAgentService(sessionLogic, messageLogic, agentModelLogic, skillLogic, mcpLogic, agentBotLogic)
	aiagent.RegisterAiAgentServiceServer(s, aiAgentService)
}

func StopAgentBotRuntime() {
	agentBotRuntimeMu.Lock()
	runtime := agentBotRuntime
	agentBotRuntime = nil
	agentBotRuntimeMu.Unlock()
	if runtime != nil {
		runtime.Stop()
	}
}
