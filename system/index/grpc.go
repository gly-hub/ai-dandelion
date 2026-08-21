package index

import (
	"context"
	"time"

	systemproto "github.com/gly-hub/ai-dandelion/proto/system"
	"github.com/gly-hub/ai-dandelion/system/boot"
	"github.com/gly-hub/ai-dandelion/system/global"
	"github.com/gly-hub/ai-dandelion/system/internal/dao"
	"github.com/gly-hub/ai-dandelion/system/internal/logic"
	"github.com/gly-hub/ai-dandelion/system/internal/migration"
	"github.com/gly-hub/ai-dandelion/system/internal/service"
	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
	"github.com/gly-hub/ai-dandelion/toolbox/eventbus"
	uploader_minio "github.com/gly-hub/ai-dandelion/toolbox/uploader-minio"
	rpc "google.golang.org/grpc"
)

func RegisterHandler(s *rpc.Server) {
	db, err := global.GetApp().GormManager().GetDB("ai-dandelion")
	if err != nil {
		panic(err)
	}
	if err := migration.EnsureUploadOwnership(db); err != nil {
		panic(err)
	}

	userDao := dao.NewUser(db)
	menuDao := dao.NewMenu(db)
	roleDao := dao.NewRole(db)
	operationLogDao := dao.NewOperationLog(db)
	notificationDao := dao.NewNotification(db)
	agentModelDao := dao.NewAgentModel(db)
	agentConfigDao := dao.NewAgentConfig(db)
	agentSessionConfigDao := dao.NewAgentSessionConfig(db)
	uploadDao := dao.NewUpload(db)
	var uploader *uploader_minio.MinioUploader
	if cfg := global.GetConfig().UploaderConfig; cfg.Address != "" && cfg.Bucket != "" {
		uploader, err = uploader_minio.NewUploader(cfg)
		if err != nil {
			panic(err)
		}
	}
	authConfig := global.GetConfig().AuthConfig
	redisClient, redisErr := global.GetApp().RedisManager().GetRedisClient("ai-dandelion")
	if redisErr != nil || redisClient == nil {
		panic("redis is required for auth token storage")
	}
	authTokenStore := dao.NewAuthTokenStore(redisClient)
	operationLogLogic := logic.NewOperationLogLogic(operationLogDao)
	var notificationBus eventbus.Bus
	if redisClient, redisErr := global.GetApp().RedisManager().GetRedisClient("ai-dandelion"); redisErr == nil && redisClient != nil {
		notificationBus, _ = eventbus.NewRedisStreams(redisClient)
	}
	notificationLogic := logic.NewNotificationLogic(notificationDao, userDao, notificationBus)
	userLogic := logic.NewUserLogicWithTokenStore(
		userDao,
		roleDao,
		authTokenStore,
		authConfig.AccessTokenSecret,
		authctx.ParseTTLWithDefault(authConfig.AccessTokenTTL, 4*time.Hour),
		authctx.ParseTTLWithDefault(authConfig.RefreshTokenTTL, 7*24*time.Hour),
		operationLogLogic,
	)
	menuLogic := logic.NewMenuLogic(menuDao, roleDao, operationLogLogic)
	roleLogic := logic.NewRoleLogic(roleDao, menuDao, operationLogLogic)
	agentModelLogic := logic.NewAgentModelLogic(agentModelDao)
	agentConfigLogic := logic.NewAgentConfigLogic(agentConfigDao)
	agentSessionConfigLogic := logic.NewAgentSessionConfigLogic(agentSessionConfigDao)
	uploadLogic := logic.NewUploadLogic(uploadDao, uploader)
	if err := boot.EnsureDatabaseReady(global.GetConfig().AppConfig.Env); err != nil {
		panic(err)
	}
	if err := menuLogic.EnsureSeedMenus(context.Background()); err != nil {
		panic(err)
	}
	if err := menuLogic.EnsureSeedButtonMenus(context.Background()); err != nil {
		panic(err)
	}
	if err := menuLogic.EnsureSeedRoles(context.Background()); err != nil {
		panic(err)
	}
	if err := userLogic.EnsureSeedAdminUser(context.Background()); err != nil {
		panic(err)
	}
	if err := agentConfigLogic.EnsureSeedAgentConfig(
		context.Background(),
		"You are a helpful agent assistant. Keep answers clear and practical.",
		"bypassPermissions",
		20,
	); err != nil {
		panic(err)
	}
	if err := agentSessionConfigLogic.EnsureSeedAgentSessionConfigs(context.Background()); err != nil {
		panic(err)
	}
	systemService := service.NewSystemService(userLogic, menuLogic, roleLogic, agentModelLogic, agentConfigLogic, agentSessionConfigLogic, operationLogLogic, notificationLogic, uploadLogic)
	systemproto.RegisterSystemServiceServer(s, systemService)
}
