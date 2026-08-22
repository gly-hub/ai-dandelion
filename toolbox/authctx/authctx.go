package authctx

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc/metadata"
)

const (
	MetadataUserID   = "x-user-id"
	MetadataUsername = "x-username"
	MetadataRoleIDs  = "x-role-ids"
)

const defaultTokenTTL = 4 * time.Hour

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
	SessionID string   `json:"sid,omitempty"`
	TokenType string   `json:"typ,omitempty"`
	jwt.RegisteredClaims
}

func SignToken(secret string, user User, ttl time.Duration) (string, error) {
	token, _, err := SignAccessToken(secret, user, "", ttl)
	return token, err
}

func SignAccessToken(secret string, user User, sessionID string, ttl time.Duration) (string, string, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "", "", errors.New("auth token secret is required")
	}
	user.ID = strings.TrimSpace(user.ID)
	user.Username = strings.TrimSpace(user.Username)
	if user.ID == "" {
		return "", "", errors.New("user id is required")
	}
	if ttl <= 0 {
		ttl = defaultTokenTTL
	}
	tokenID, err := NewOpaqueToken(18)
	if err != nil {
		return "", "", err
	}
	now := time.Now()
	claims := Claims{
		UserID:    user.ID,
		Username:  user.Username,
		RoleIDs:   normalizeRoleIDs(user.RoleIDs),
		SessionID: strings.TrimSpace(sessionID),
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", "", err
	}
	return signed, tokenID, nil
}

func VerifyToken(secret string, token string) (User, error) {
	claims, err := VerifyAccessToken(secret, token)
	if err != nil {
		return User{}, err
	}
	return userFromClaims(claims)
}

func VerifyAccessToken(secret string, token string) (*Claims, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, errors.New("auth token secret is required")
	}
	if strings.TrimSpace(token) == "" {
		return nil, ErrInvalidToken
	}
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(parsed *jwt.Token) (interface{}, error) {
		if parsed.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}
	if !parsed.Valid || claims.ExpiresAt == nil || claims.ExpiresAt.Time.IsZero() {
		return nil, ErrInvalidToken
	}
	if claims.TokenType != "" && claims.TokenType != "access" {
		return nil, ErrInvalidToken
	}
	if claims.UserID == "" || claims.ID == "" || (claims.Subject != "" && claims.Subject != claims.UserID) {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func userFromClaims(claims *Claims) (User, error) {
	if claims == nil || strings.TrimSpace(claims.UserID) == "" {
		return User{}, ErrInvalidToken
	}
	return User{
		ID:       strings.TrimSpace(claims.UserID),
		Username: strings.TrimSpace(claims.Username),
		RoleIDs:  normalizeRoleIDs(claims.RoleIDs),
	}, nil
}

func NewOpaqueToken(size int) (string, error) {
	if size < 32 {
		size = 32
	}
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return fmt.Sprintf("%x", sum[:])
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
	return ParseTTLWithDefault(raw, defaultTokenTTL)
}

func ParseTTLWithDefault(raw string, fallback time.Duration) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	if seconds, err := strconv.ParseInt(raw, 10, 64); err == nil {
		if seconds <= 0 {
			return fallback
		}
		return time.Duration(seconds) * time.Second
	}
	ttl, err := time.ParseDuration(raw)
	if err != nil || ttl <= 0 {
		return fallback
	}
	return ttl
}
