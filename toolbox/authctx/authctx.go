package authctx

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"
)

const (
	MetadataUserID   = "x-user-id"
	MetadataUsername = "x-username"
	MetadataRoleIDs  = "x-role-ids"
)

const defaultTokenTTL = 24 * time.Hour

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

type User struct {
	ID       string
	Username string
	RoleIDs  []string
}

type Claims struct {
	UserID    string   `json:"uid"`
	Username  string   `json:"un"`
	RoleIDs   []string `json:"rids,omitempty"`
	ExpiresAt int64    `json:"exp"`
}

func SignToken(secret string, user User, ttl time.Duration) (string, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "", errors.New("auth token secret is required")
	}
	user.ID = strings.TrimSpace(user.ID)
	user.Username = strings.TrimSpace(user.Username)
	if user.ID == "" {
		return "", errors.New("user id is required")
	}
	if ttl <= 0 {
		ttl = defaultTokenTTL
	}
	claims := Claims{
		UserID:    user.ID,
		Username:  user.Username,
		RoleIDs:   normalizeRoleIDs(user.RoleIDs),
		ExpiresAt: time.Now().Add(ttl).Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payloadPart := base64.RawURLEncoding.EncodeToString(payload)
	return payloadPart + "." + sign(payloadPart, secret), nil
}

func VerifyToken(secret string, token string) (User, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return User{}, errors.New("auth token secret is required")
	}
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return User{}, ErrInvalidToken
	}
	expected := sign(parts[0], secret)
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return User{}, ErrInvalidToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return User{}, ErrInvalidToken
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return User{}, ErrInvalidToken
	}
	if claims.ExpiresAt <= 0 || time.Now().Unix() > claims.ExpiresAt {
		return User{}, ErrExpiredToken
	}
	claims.UserID = strings.TrimSpace(claims.UserID)
	if claims.UserID == "" {
		return User{}, ErrInvalidToken
	}
	return User{
		ID:       claims.UserID,
		Username: strings.TrimSpace(claims.Username),
		RoleIDs:  normalizeRoleIDs(claims.RoleIDs),
	}, nil
}

func ContextWithUser(ctx context.Context, user User) context.Context {
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	md.Set(MetadataUserID, strings.TrimSpace(user.ID))
	md.Set(MetadataUsername, strings.TrimSpace(user.Username))
	md.Set(MetadataRoleIDs, strings.Join(normalizeRoleIDs(user.RoleIDs), ","))
	return metadata.NewOutgoingContext(ctx, md)
}

func ForwardUserContext(ctx context.Context) context.Context {
	user, ok := CurrentUser(ctx)
	if !ok {
		return ctx
	}
	return ContextWithUser(ctx, user)
}

func CurrentUser(ctx context.Context) (User, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md, ok = metadata.FromOutgoingContext(ctx)
	}
	if !ok {
		return User{}, false
	}
	userID := firstMetadataValue(md, MetadataUserID)
	if strings.TrimSpace(userID) == "" {
		return User{}, false
	}
	return User{
		ID:       strings.TrimSpace(userID),
		Username: strings.TrimSpace(firstMetadataValue(md, MetadataUsername)),
		RoleIDs:  splitRoleIDs(firstMetadataValue(md, MetadataRoleIDs)),
	}, true
}

func RequireUserID(ctx context.Context) (string, error) {
	user, ok := CurrentUser(ctx)
	if !ok || strings.TrimSpace(user.ID) == "" {
		return "", errors.New("user is required")
	}
	return strings.TrimSpace(user.ID), nil
}

func TokenTTLSeconds(ttl time.Duration) int64 {
	if ttl <= 0 {
		ttl = defaultTokenTTL
	}
	return int64(ttl / time.Second)
}

func sign(payload string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func normalizeRoleIDs(roleIDs []string) []string {
	out := make([]string, 0, len(roleIDs))
	seen := make(map[string]struct{}, len(roleIDs))
	for _, roleID := range roleIDs {
		roleID = strings.TrimSpace(roleID)
		if roleID == "" {
			continue
		}
		if _, ok := seen[roleID]; ok {
			continue
		}
		seen[roleID] = struct{}{}
		out = append(out, roleID)
	}
	return out
}

func splitRoleIDs(raw string) []string {
	if raw == "" {
		return nil
	}
	return normalizeRoleIDs(strings.Split(raw, ","))
}

func firstMetadataValue(md metadata.MD, key string) string {
	values := md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func FormatTTL(ttl time.Duration) string {
	if ttl <= 0 {
		ttl = defaultTokenTTL
	}
	return fmt.Sprintf("%ds", int64(ttl/time.Second))
}

func ParseTTLSeconds(raw int64) time.Duration {
	if raw <= 0 {
		return defaultTokenTTL
	}
	return time.Duration(raw) * time.Second
}

func ParseTTL(raw string) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultTokenTTL
	}
	if seconds, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return ParseTTLSeconds(seconds)
	}
	ttl, err := time.ParseDuration(raw)
	if err != nil || ttl <= 0 {
		return defaultTokenTTL
	}
	return ttl
}
