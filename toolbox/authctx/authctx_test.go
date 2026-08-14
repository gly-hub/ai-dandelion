package authctx

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc/metadata"
)

func TestSignAndVerifyToken(t *testing.T) {
	token, err := SignToken("secret", User{ID: "u1", Username: "admin", RoleIDs: []string{"r1", "r1", "r2"}}, time.Hour)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	user, err := VerifyToken("secret", token)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}
	if user.ID != "u1" || user.Username != "admin" || len(user.RoleIDs) != 2 {
		t.Fatalf("unexpected user: %#v", user)
	}
	if _, err := VerifyToken("other", token); err == nil {
		t.Fatal("expected invalid token with wrong secret")
	}
}

func TestCurrentUserFromIncomingMetadata(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		MetadataUserID, "u1",
		MetadataUsername, "admin",
		MetadataRoleIDs, "r1,r2",
	))
	user, ok := CurrentUser(ctx)
	if !ok {
		t.Fatal("expected user")
	}
	if user.ID != "u1" || user.Username != "admin" || len(user.RoleIDs) != 2 {
		t.Fatalf("unexpected user: %#v", user)
	}
}
