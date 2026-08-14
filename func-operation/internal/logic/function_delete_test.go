package logic

import (
	"context"
	"testing"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/model"
)

func TestDeleteFunctionMenuSkipsWhenMenuSyncMissing(t *testing.T) {
	t.Parallel()

	logic := &FunctionLogic{}
	if err := logic.deleteFunctionMenu(context.Background(), &model.Function{UUID: "func-1"}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestDeleteFunctionMenuSkipsNilFunction(t *testing.T) {
	t.Parallel()

	logic := &FunctionLogic{}
	if err := logic.deleteFunctionMenu(context.Background(), nil); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
