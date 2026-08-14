package logic

import (
	"errors"
	"testing"

	"github.com/gly-hub/ai-dandelion/func-operation/internal/runtime/generatedapp"
)

func TestClassifyInvokeErrorAppNotFound(t *testing.T) {
	detail := classifyInvokeError(generatedapp.ErrAppNotFound)
	if detail.ErrorCode != invokeErrorCodeFrontendLoad || detail.Hint == "" {
		t.Fatalf("unexpected detail: %#v", detail)
	}
}

func TestClassifyInvokeErrorWasmCompile(t *testing.T) {
	detail := classifyInvokeError(errors.New("compile generated app \"x\": invalid wasm"))
	if detail.ErrorCode != invokeErrorCodeWasmCompile || detail.Stage != "wasm_compile" {
		t.Fatalf("unexpected detail: %#v", detail)
	}
}
