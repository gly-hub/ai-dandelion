package dao

import (
	"regexp"
	"strings"
)

var (
	sqliteIntegerPrimaryKeyAutoIncrement = regexp.MustCompile(`(?i)\bINTEGER\s+PRIMARY\s+KEY\s+AUTOINCREMENT\b`)
	sqliteAutoIncrement                  = regexp.MustCompile(`(?i)\bAUTOINCREMENT\b`)
	sqliteInteger                        = regexp.MustCompile(`(?i)\bINTEGER\b`)
	sqliteReal                           = regexp.MustCompile(`(?i)\bREAL\b`)
	sqliteText                           = regexp.MustCompile(`(?i)\bTEXT\b`)
)

func normalizeDDLForMySQL(ddl string) string {
	return normalizeDDLForDialect(ddl, "mysql")
}

func normalizeDDLForDialect(ddl string, dialect string) string {
	ddl = strings.TrimSpace(ddl)
	if ddl == "" || dialect != "mysql" {
		return ddl
	}
	ddl = sqliteIntegerPrimaryKeyAutoIncrement.ReplaceAllString(ddl, "BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY")
	ddl = sqliteAutoIncrement.ReplaceAllString(ddl, "AUTO_INCREMENT")
	ddl = sqliteInteger.ReplaceAllString(ddl, "BIGINT")
	ddl = sqliteReal.ReplaceAllString(ddl, "DOUBLE")
	// MySQL rejects DEFAULT on TEXT/BLOB columns in common sql_mode settings.
	ddl = sqliteText.ReplaceAllString(ddl, "VARCHAR(255)")
	return ddl
}
