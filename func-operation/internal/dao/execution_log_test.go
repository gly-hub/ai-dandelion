package dao

import (
	"context"
	"fmt"
	"testing"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFunctionExecutionLogListByFunctionIDPaginatesFilteredRows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.FunctionExecutionLog{}); err != nil {
		t.Fatalf("migrate execution logs: %v", err)
	}
	store := NewFunctionExecutionLog(db)
	for index := 1; index <= 5; index++ {
		item := &model.FunctionExecutionLog{
			UUID: fmt.Sprintf("log-%d", index), FunctionID: "function-1", AppID: "app-1", InvocationType: "published", Status: "succeeded", LogsJSON: "[]", CreatedAt: int64(index),
		}
		if index == 3 {
			item.Status = "failed"
			item.ErrorCode = "DATA_CREATE_FAILED"
		}
		if err := store.Create(context.Background(), item); err != nil {
			t.Fatalf("create log %d: %v", index, err)
		}
	}

	items, total, err := store.ListByFunctionID(context.Background(), "function-1", ExecutionLogFilter{Page: 2, Limit: 2})
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if total != 5 || len(items) != 2 {
		t.Fatalf("page result total=%d items=%#v", total, items)
	}
	if items[0].UUID != "log-3" || items[1].UUID != "log-2" {
		t.Fatalf("expected descending second page, got %#v", items)
	}

	items, total, err = store.ListByFunctionID(context.Background(), "function-1", ExecutionLogFilter{Page: 1, Limit: 20, Status: "failed"})
	if err != nil {
		t.Fatalf("list filtered page: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].UUID != "log-3" {
		t.Fatalf("filtered result total=%d items=%#v", total, items)
	}
}
