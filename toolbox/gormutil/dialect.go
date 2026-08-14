package gormutil

import (
	"fmt"

	"gorm.io/gorm"
)

const (
	DialectMySQL  = "mysql"
	DialectSQLite = "sqlite"
)

func DialectName(db *gorm.DB) string {
	if db == nil || db.Dialector == nil {
		return ""
	}
	return db.Dialector.Name()
}

func IsMySQL(db *gorm.DB) bool {
	return DialectName(db) == DialectMySQL
}

func IsSQLite(db *gorm.DB) bool {
	return DialectName(db) == DialectSQLite
}

func WithMySQLTableOptions(db *gorm.DB) *gorm.DB {
	if !IsMySQL(db) {
		return db
	}
	return db.Set("gorm:table_options", "ENGINE=InnoDB CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci")
}

func ApplyTableComment(db *gorm.DB, tableName string, comment string) {
	if !IsMySQL(db) {
		return
	}
	db.Exec(fmt.Sprintf("ALTER TABLE %s COMMENT '%s'", tableName, comment))
}
