package index

import (
	"context"

	systemproto "github.com/team-dandelion/ai-dandelion/proto/system"
	"github.com/team-dandelion/ai-dandelion/system/boot"
	"github.com/team-dandelion/ai-dandelion/system/global"
	"github.com/team-dandelion/ai-dandelion/system/internal/dao"
	"github.com/team-dandelion/ai-dandelion/system/internal/logic"
	"github.com/team-dandelion/ai-dandelion/system/internal/service"
	"github.com/team-dandelion/ai-dandelion/toolbox/authctx"
	"github.com/team-dandelion/ai-dandelion/toolbox/eventbus"
	uploader_minio "github.com/team-dandelion/ai-dandelion/toolbox/uploader-minio"
	rpc "google.golang.org/grpc"
)

func RegisterHandler(s *rpc.Server) {
	db, err := global.GetApp().GormManager().GetDB("ai-dandelion")
	if err != nil {
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
	operationLogLogic := logic.NewOperationLogLogic(operationLogDao)
	var notificationBus eventbus.Bus
	if redisClient, redisErr := global.GetApp().RedisManager().GetRedisClient("ai-dandelion"); redisErr == nil && redisClient != nil {
		notificationBus, _ = eventbus.NewRedisStreams(redisClient)
	}
	notificationLogic := logic.NewNotificationLogic(notificationDao, userDao, notificationBus)
	userLogic := logic.NewUserLogic(userDao, roleDao, authConfig.TokenSecret, authctx.ParseTTL(authConfig.TokenTTL), operationLogLogic)
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
