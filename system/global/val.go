package global

import (
	"github.com/team-dandelion/ai-dandelion/system/config"
	"github.com/team-dandelion/quickgo"
)

var (
	framework  *quickgo.Framework
	conf       *config.Config
	ErrChan    = make(chan error, 10)
	ConfigPath string
)

func GetConfig() *config.Config {
	if conf == nil {
		panic("config not init")
	}
	return conf
}

func SetConfig(c *config.Config) {
	conf = c
}

func GetApp() *quickgo.Framework {
	if framework == nil {
		panic("app not init")
	}
	return framework
}

func SetApp(app *quickgo.Framework) {
	if app == nil {
		panic("app not init")
	}
	framework = app
}
