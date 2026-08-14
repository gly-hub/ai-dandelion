package logic

import (
	"context"
	"testing"
)

func TestFunctionAuthorizerRejectsMissingDependencies(t *testing.T) {
	t.Parallel()
	if err := (&FunctionAuthorizer{}).Require(context.Background(), functionPermissionEdit); err == nil {
		t.Fatal("Require() should fail closed when authorization is not configured")
	}
}

func TestFunctionMenuSyncRejectsMissingDependencies(t *testing.T) {
	t.Parallel()
	allowed, err := (&FunctionMenuSync{}).CheckAccess(context.Background(), "user-1", "function-1", "")
	if err == nil || allowed {
		t.Fatal("CheckAccess() must fail closed when menu authorization is not configured")
	}
}
