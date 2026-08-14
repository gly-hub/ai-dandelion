package model

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFunctionAutoAllocatesIDOnSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Function{}); err != nil {
		t.Fatalf("migrate function: %v", err)
	}

	function := &Function{UUID: "test-function", Name: "Test function"}
	if err := db.Create(function).Error; err != nil {
		t.Fatalf("create function: %v", err)
	}
	if function.ID == 0 {
		t.Fatal("function ID was not allocated")
	}
}
