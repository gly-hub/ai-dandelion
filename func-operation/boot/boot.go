package boot

import (
	"github.com/team-dandelion/ai-dandelion/func-operation/config"
	"github.com/team-dandelion/ai-dandelion/func-operation/global"
	"github.com/team-dandelion/ai-dandelion/toolbox/gormutil"
	"github.com/team-dandelion/quickgo"
)

func bootConfig(confPath string) (cfg *config.Config, err error) {
	quickgo.InitConfig("local", confPath)
	cfg = new(config.Config)
	quickgo.LoadCustomConfig(cfg)
	global.SetConfig(cfg)
	return cfg, nil
}

func Boot(confPath string) (err error) {
	cfg, err := bootConfig(confPath)
	if err != nil {
		return err
	}
	if err := gormutil.EnsureSQLiteDirs(&cfg.GormConfig); err != nil {
		return err
	}

	app, err := quickgo.NewFramework(
		quickgo.ConfigOptionWithApp(cfg.AppConfig),
		quickgo.ConfigOptionWithLogger(cfg.LoggerConfig),
		quickgo.ConfigOptionWithGrpcServer(&cfg.GrpcServerConfig),
		quickgo.ConfigOptionWithGrpcClient(&cfg.GrpcClientConfig),
		quickgo.ConfigOptionWithGorm(&cfg.GormConfig),
		quickgo.ConfigOptionWithRedis(&cfg.RedisConfig),
		quickgo.ConfigOptionWithTracing(&cfg.TracingConfig),
	)
	if err != nil {
		return err
	}
	if err = app.Init(); err != nil {
		panic(err)
	}

	global.SetApp(app)
	if err := EnsureDatabaseReady(global.GetConfig().AppConfig.Env); err != nil {
		return err
	}
	return nil
}
