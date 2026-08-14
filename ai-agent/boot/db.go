package boot

import (
	"context"
	"fmt"

	"github.com/gly-hub/ai-dandelion/ai-agent/global"
	"github.com/gly-hub/ai-dandelion/toolbox/gormutil"
	"github.com/gly-hub/ai-dandelion/toolbox/redislock"
	"github.com/gly-hub/quickgo/logger"
)

type DatabaseModel interface {
	TableName() string
	TableComment() string
}

var models = make([]DatabaseModel, 0)

func Register(model ...DatabaseModel) {
	if len(model) == 0 {
		return
	}
	models = append(models, model...)
}

func GameDbAutoMigrate(stage string) {
	_ = migrate(stage)
}

func EnsureDatabaseReady(stage string) error {
	if !migrate(stage) {
		return fmt.Errorf("database migrate failed")
	}
	return nil
}

func migrate(stage string) bool {
	ctx := context.Background()
	dbIns, _ := global.GetApp().GormManager().GetDB("ai-dandelion")
	if dbIns == nil {
		return false
	}
	dbIns = gormutil.WithMySQLTableOptions(dbIns)

	redisClient, _ := global.GetApp().RedisManager().GetRedisClient("ai-dandelion")
	if redisClient != nil && gormutil.IsMySQL(dbIns) {
		redisLock, err := redislock.NewDistributeLockRedis(redisClient, fmt.Sprintf("business_console_db_migrate:%s", stage), 600, "1")
		if err != nil {
			logger.Error(ctx, "Migrate Model Error: %v", err)
			return false
		}
		defer func() {
			_ = redisLock.Unlock()
		}()
	} else if gormutil.IsSQLite(dbIns) {
		logger.Info(ctx, "SQLite detected, running AutoMigrate without Redis lock")
	} else {
		logger.Error(ctx, "Redis unavailable for migrate lock, running AutoMigrate without lock")
	}

	for _, model := range models {
		if dbIns.Migrator().HasTable(model) && (stage == "production") {
			continue
		}
		var err error
		if stage == "production" {
			err = dbIns.Migrator().AutoMigrate(model)
			gormutil.ApplyTableComment(dbIns, model.TableName(), model.TableComment())
		} else {
			err = dbIns.AutoMigrate(model)
			gormutil.ApplyTableComment(dbIns, model.TableName(), model.TableComment())
		}
		if err != nil {
			logger.Error(context.Background(), "Migrate Model Error:%v", err)
		}
	}
	return true
}
