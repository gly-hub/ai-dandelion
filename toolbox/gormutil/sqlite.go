package gormutil

import (
	"os"
	"path/filepath"
	"strings"

	qggorm "github.com/team-dandelion/quickgo/db/gorm"
)

func EnsureSQLiteDirs(config *qggorm.GormManagerConfig) error {
	if config == nil {
		return nil
	}
	for _, database := range config.Databases {
		if database.Master.Type != qggorm.DatabaseTypeSQLite {
			continue
		}
		path := database.Master.DSN
		if path == "" {
			path = database.Master.Database
		}
		if path == "" || path == ":memory:" {
			continue
		}
		path = sqliteFilePath(path)
		if path == "" || path == ":memory:" {
			continue
		}
		dir := filepath.Dir(path)
		if dir == "." || dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func sqliteFilePath(dsn string) string {
	if index := strings.Index(dsn, "?"); index >= 0 {
		return dsn[:index]
	}
	return dsn
}
