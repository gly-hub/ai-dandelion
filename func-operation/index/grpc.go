package index

import (
	"context"
	"fmt"
	"time"

	"github.com/team-dandelion/ai-dandelion/func-operation/global"
	"github.com/team-dandelion/ai-dandelion/func-operation/internal/dao"
	"github.com/team-dandelion/ai-dandelion/func-operation/internal/logic"
	"github.com/team-dandelion/ai-dandelion/func-operation/internal/runtime/generatedapp"
	"github.com/team-dandelion/ai-dandelion/func-operation/internal/service"
	aiagent "github.com/team-dandelion/ai-dandelion/proto/ai-agent"
	funcoperation "github.com/team-dandelion/ai-dandelion/proto/func-operation"
	systemproto "github.com/team-dandelion/ai-dandelion/proto/system"
	"github.com/team-dandelion/ai-dandelion/toolbox/eventbus"
	rpc "google.golang.org/grpc"
)

func RegisterHandler(s *rpc.Server) {
	db, err := global.GetApp().GormManager().GetDB("ai-dandelion")
	if err != nil {
		panic(err)
	}
	debugDB, err := global.GetApp().GormManager().GetDB("ai-dandelion-debug")
	if err != nil {
		panic(err)
	}
	functionDao := dao.NewFunction(db)
	messageStore := dao.NewAiAgentMessageStore(db)
	generatedAppDao := dao.NewGeneratedApp(db)
	publicConfigDao := dao.NewPublicConfig(db)
	externalAPIDao := dao.NewExternalAPI(db)
	debugGeneratedAppDao := dao.NewGeneratedApp(debugDB)
	releaseDao := dao.NewFunctionRelease(db)
	outboxDao := dao.NewFunctionOutbox(db)
	redisClient, redisErr := global.GetApp().RedisManager().GetRedisClient("ai-dandelion")
	if redisErr != nil || redisClient == nil {
		panic(fmt.Errorf("initialize realtime event bus redis client: %w", redisErr))
	}
	realtimeBus, realtimeBusErr := eventbus.NewRedisStreams(redisClient)
	if realtimeBusErr != nil {
		panic(fmt.Errorf("initialize realtime event bus: %w", realtimeBusErr))
	}
	generatedFunctionMenuDao := dao.NewGeneratedFunctionMenu(db)
	aiAgentClientProvider := func(ctx context.Context) (aiagent.AiAgentServiceClient, error) {
		conn, err := global.GetApp().GrpcClientManager().GetConn(ctx, "ai-agent")
		if err != nil {
			return nil, err
		}
		return aiagent.NewAiAgentServiceClient(conn), nil
	}
	systemConn, err := global.GetApp().GrpcClientManager().GetConn(context.Background(), "system")
	if err != nil {
		panic(err)
	}
	systemClient := systemproto.NewSystemServiceClient(systemConn)
	menuSync := logic.NewFunctionMenuSync(systemClient)
	authorizer := logic.NewFunctionAuthorizer(systemClient)
	publicConfigLogic := logic.NewPublicConfigLogic(publicConfigDao, authorizer)
	externalAPILogic := logic.NewExternalAPILogic(externalAPIDao, authorizer)
	generatedAppConfig := global.GetConfig().GeneratedApp
	artifactStore, err := generatedapp.NewArtifactStore(generatedAppConfig.RootDir, generatedapp.ArtifactStoreConfig{
		Driver: generatedAppConfig.ArtifactStore.Driver,
		S3: generatedapp.S3ArtifactStoreConfig{
			Endpoint:  generatedAppConfig.ArtifactStore.S3.Endpoint,
			AccessKey: generatedAppConfig.ArtifactStore.S3.AccessKey,
			SecretKey: generatedAppConfig.ArtifactStore.S3.SecretKey,
			Bucket:    generatedAppConfig.ArtifactStore.S3.Bucket,
			Prefix:    generatedAppConfig.ArtifactStore.S3.Prefix,
			Region:    generatedAppConfig.ArtifactStore.S3.Region,
			UseSSL:    generatedAppConfig.ArtifactStore.S3.UseSSL,
		},
	})
	if err != nil {
		panic(err)
	}
	appRuntime, err := generatedapp.NewService(
		context.Background(),
		generatedAppConfig.RootDir,
		generatedAppDao,
		generatedapp.WithInvokeTimeout(time.Duration(generatedAppConfig.DataCapability.InvokeTimeoutMs)*time.Millisecond),
		generatedapp.WithMaxResultBytes(generatedAppConfig.DataCapability.MaxResponseBytes),
		generatedapp.WithArtifactStore(artifactStore),
	)
	if err != nil {
		panic(err)
	}
	previewRuntime, err := generatedapp.NewService(
		context.Background(),
		generatedAppConfig.RootDir,
		generatedAppDao,
		generatedapp.WithInvokeTimeout(time.Duration(generatedAppConfig.DataCapability.InvokeTimeoutMs)*time.Millisecond),
		generatedapp.WithMaxResultBytes(generatedAppConfig.DataCapability.MaxResponseBytes),
		generatedapp.WithDraftRuntime(),
		generatedapp.WithDataStore(debugGeneratedAppDao),
	)
	if err != nil {
		panic(err)
	}
	releaseLogic := logic.NewReleaseLogic(releaseDao, outboxDao, appRuntime, functionDao, realtimeBus)
	if err := releaseLogic.BackfillLegacyPublished(context.Background()); err != nil {
		panic(err)
	}
	artifactMaintenance := generatedAppConfig.ArtifactMaintenance
	staleStagingAfter := time.Duration(artifactMaintenance.StaleStagingSeconds) * time.Second
	if staleStagingAfter <= 0 {
		staleStagingAfter = time.Hour
	}
	if _, err := releaseLogic.ReconcileArtifactStore(context.Background(), staleStagingAfter); err != nil {
		panic(err)
	}
	if err := releaseLogic.RevokeOrphanedPublished(context.Background()); err != nil {
		panic(err)
	}
	if err := releaseLogic.RestorePublished(context.Background()); err != nil {
		panic(err)
	}
	outboxProcessor := logic.NewOutboxProcessor(outboxDao, functionDao, menuSync, realtimeBus)
	if _, err := outboxProcessor.ProcessReady(context.Background(), 50); err != nil {
		panic(err)
	}
	StartOutboxRuntime(outboxProcessor)
	artifactReconcileInterval := time.Duration(artifactMaintenance.ReconcileIntervalSeconds) * time.Second
	StartArtifactRuntime(releaseLogic, artifactReconcileInterval, staleStagingAfter)
	functionLogic := logic.NewFunctionLogic(functionDao, messageStore, generatedAppDao, appRuntime, previewRuntime, aiAgentClientProvider, menuSync, authorizer, releaseLogic)
	appLogic := logic.NewGeneratedAppLogic(appRuntime, previewRuntime, functionDao, menuSync, generatedFunctionMenuDao, releaseLogic, authorizer, publicConfigLogic)
	if err := appLogic.SyncPublishedFunctionActions(context.Background()); err != nil {
		panic(err)
	}
	appRuntime.SetCapabilityBroker(appLogic)
	previewRuntime.SetCapabilityBroker(appLogic)
	appRuntime.SetExternalAPIExecutor(externalAPILogic)
	previewRuntime.SetExternalAPIExecutor(externalAPILogic)
	outboxLogic := logic.NewOutboxManagementLogic(outboxDao, outboxProcessor, authorizer)
	funcOperationService := service.NewFuncOperationService(functionLogic, appLogic, outboxLogic, publicConfigLogic, externalAPILogic)

	funcoperation.RegisterFuncOperationServiceServer(s, funcOperationService)
}
