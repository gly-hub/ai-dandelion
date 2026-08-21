package dao

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
	"github.com/redis/go-redis/v9"
)

var ErrAuthTokenNotFound = errors.New("auth token not found")

type AuthTokenSession struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	Username  string `json:"username"`
	ExpiresAt int64  `json:"expiresAt"`
}

type AuthTokenStore struct {
	redis  redis.UniversalClient
	prefix string
}

func NewAuthTokenStore(client redis.UniversalClient) *AuthTokenStore {
	return &AuthTokenStore{redis: client, prefix: "auth:"}
}

func (s *AuthTokenStore) CreateSession(ctx context.Context, session AuthTokenSession, accessJTI, refreshToken string, accessTTL, refreshTTL time.Duration) error {
	if s == nil || s.redis == nil {
		return errors.New("auth token redis store is unavailable")
	}
	if strings.TrimSpace(session.ID) == "" || strings.TrimSpace(session.UserID) == "" {
		return errors.New("auth token session is incomplete")
	}
	if accessTTL <= 0 || refreshTTL <= 0 {
		return errors.New("auth token ttl must be positive")
	}
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	keys := []string{s.sessionKey(session.ID), s.accessKey(accessJTI), s.refreshKey(refreshToken)}
	if err := s.redis.Set(ctx, keys[0], data, refreshTTL).Err(); err != nil {
		return err
	}
	if err := s.redis.Set(ctx, keys[1], session.ID, accessTTL).Err(); err != nil {
		_ = s.redis.Del(ctx, keys[0]).Err()
		return err
	}
	if err := s.redis.Set(ctx, keys[2], session.ID, refreshTTL).Err(); err != nil {
		_ = s.redis.Del(ctx, keys[0], keys[1]).Err()
		return err
	}
	return nil
}

func (s *AuthTokenStore) SaveRotatedTokens(ctx context.Context, sessionID, accessJTI, refreshToken string, accessTTL, refreshTTL time.Duration) error {
	if s == nil || s.redis == nil {
		return errors.New("auth token redis store is unavailable")
	}
	if strings.TrimSpace(sessionID) == "" || accessTTL <= 0 || refreshTTL <= 0 {
		return errors.New("rotated auth token data is incomplete")
	}
	if err := s.redis.Set(ctx, s.accessKey(accessJTI), sessionID, accessTTL).Err(); err != nil {
		return err
	}
	if err := s.redis.Set(ctx, s.refreshKey(refreshToken), sessionID, refreshTTL).Err(); err != nil {
		_ = s.redis.Del(ctx, s.accessKey(accessJTI)).Err()
		return err
	}
	return nil
}

func (s *AuthTokenStore) AccessSessionID(ctx context.Context, accessJTI string) (string, error) {
	if s == nil || s.redis == nil {
		return "", errors.New("auth token redis store is unavailable")
	}
	sessionID, err := s.redis.Get(ctx, s.accessKey(accessJTI)).Result()
	if errors.Is(err, redis.Nil) || strings.TrimSpace(sessionID) == "" {
		return "", ErrAuthTokenNotFound
	}
	if err != nil {
		return "", err
	}
	return sessionID, nil
}

func (s *AuthTokenStore) ConsumeRefreshSessionID(ctx context.Context, refreshToken string) (string, error) {
	if s == nil || s.redis == nil {
		return "", errors.New("auth token redis store is unavailable")
	}
	sessionID, err := s.redis.GetDel(ctx, s.refreshKey(refreshToken)).Result()
	if errors.Is(err, redis.Nil) || strings.TrimSpace(sessionID) == "" {
		return "", ErrAuthTokenNotFound
	}
	if err != nil {
		return "", err
	}
	return sessionID, nil
}

func (s *AuthTokenStore) GetSession(ctx context.Context, sessionID string) (*AuthTokenSession, error) {
	if s == nil || s.redis == nil {
		return nil, errors.New("auth token redis store is unavailable")
	}
	data, err := s.redis.Get(ctx, s.sessionKey(sessionID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrAuthTokenNotFound
	}
	if err != nil {
		return nil, err
	}
	var session AuthTokenSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, errors.New("invalid auth token session")
	}
	if session.ExpiresAt <= time.Now().Unix() {
		return nil, ErrAuthTokenNotFound
	}
	return &session, nil
}

func (s *AuthTokenStore) RevokeSession(ctx context.Context, sessionID string) error {
	if s == nil || s.redis == nil {
		return errors.New("auth token redis store is unavailable")
	}
	return s.redis.Del(ctx, s.sessionKey(sessionID)).Err()
}

func (s *AuthTokenStore) sessionKey(id string) string {
	return s.prefix + "session:" + strings.TrimSpace(id)
}

func (s *AuthTokenStore) accessKey(jti string) string {
	return s.prefix + "access:" + authctx.HashToken(jti)
}

func (s *AuthTokenStore) refreshKey(token string) string {
	return s.prefix + "refresh:" + authctx.HashToken(token)
}
