package boot

import (
	"github.com/team-dandelion/ai-dandelion/inner-gateway/config"
	"github.com/team-dandelion/ai-dandelion/inner-gateway/global"
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

	app, err := quickgo.NewFramework(
		quickgo.ConfigOptionWithApp(cfg.AppConfig),
		quickgo.ConfigOptionWithLogger(cfg.LoggerConfig),
		quickgo.ConfigOptionWithGrpcClient(&cfg.GrpcClientConfig),
		quickgo.ConfigOptionWithHTTPServer(&cfg.HttpServerConfig),
		quickgo.ConfigOptionWithRedis(&cfg.RedisConfig),
		quickgo.ConfigOptionWithMongoDB(&cfg.MongoDBConfig),
		quickgo.ConfigOptionWithTracing(&cfg.TracingConfig),
	)

	if err != nil {
		return err
	}
	// 初始化所有组件
	if err = app.Init(); err != nil {
		panic(err)
	}

	global.SetApp(app)
	return nil
}
