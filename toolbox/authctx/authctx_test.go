package authctx

import (
	"context"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/metadata"
)

func TestSignAndVerifyToken(t *testing.T) {
	token, err := SignToken("secret", User{ID: "u1", Username: "admin", RoleIDs: []string{"r1", "r1", "r2"}}, time.Hour)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	if got := len(strings.Split(token, ".")); got != 3 {
		t.Fatalf("JWT segment count = %d, want 3", got)
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

func TestSignAndVerifyAccessTokenIncludesSessionAndJTI(t *testing.T) {
	token, tokenID, err := SignAccessToken("secret", User{ID: "u1", Username: "admin"}, "session-1", time.Hour)
	if err != nil {
		t.Fatalf("sign access token: %v", err)
	}
	claims, err := VerifyAccessToken("secret", token)
	if err != nil {
		t.Fatalf("verify access token: %v", err)
	}
	if claims.ID != tokenID || claims.SessionID != "session-1" || claims.TokenType != "access" {
		t.Fatalf("unexpected claims: %#v", claims)
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
