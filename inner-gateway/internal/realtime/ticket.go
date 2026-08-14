package realtime

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
	"github.com/redis/go-redis/v9"
)

type ticketRecord struct {
	user      authctx.User
	expiresAt time.Time
}

type TicketManager struct {
	mu      sync.Mutex
	records map[string]ticketRecord
	ttl     time.Duration
	store   redis.UniversalClient
}

func NewTicketManager(ttl time.Duration) *TicketManager {
	if ttl <= 0 {
		ttl = time.Minute
	}
	return &TicketManager{records: make(map[string]ticketRecord), ttl: ttl}
}

func NewRedisTicketManager(ttl time.Duration, store redis.UniversalClient) *TicketManager {
	m := NewTicketManager(ttl)
	m.store = store
	return m
}

func (m *TicketManager) Issue(user authctx.User) (string, int64, error) {
	if strings.TrimSpace(user.ID) == "" {
		return "", 0, errors.New("user is required")
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", 0, err
	}
	ticket := base64.RawURLEncoding.EncodeToString(buf)
	expiresAt := time.Now().Add(m.ttl)
	if m.store != nil {
		data, err := json.Marshal(user)
		if err != nil {
			return "", 0, err
		}
		if err := m.store.Set(context.Background(), "realtime:ticket:"+ticket, data, m.ttl).Err(); err != nil {
			return "", 0, err
		}
		return ticket, int64(m.ttl / time.Second), nil
	}
	m.mu.Lock()
	m.records[ticket] = ticketRecord{user: user, expiresAt: expiresAt}
	m.mu.Unlock()
	return ticket, int64(m.ttl / time.Second), nil
}

func (m *TicketManager) Consume(ticket string) (authctx.User, error) {
	ticket = strings.TrimSpace(ticket)
	if ticket == "" {
		return authctx.User{}, errors.New("invalid or expired realtime ticket")
	}
	if m.store != nil {
		data, err := m.store.GetDel(context.Background(), "realtime:ticket:"+ticket).Bytes()
		if err != nil {
			return authctx.User{}, errors.New("invalid or expired realtime ticket")
		}
		var user authctx.User
		if json.Unmarshal(data, &user) != nil {
			return authctx.User{}, errors.New("invalid realtime ticket")
		}
		return user, nil
	}
	m.mu.Lock()
	record, ok := m.records[ticket]
	if ok {
		delete(m.records, ticket)
	}
	m.mu.Unlock()
	if !ok || ticket == "" || time.Now().After(record.expiresAt) {
		return authctx.User{}, errors.New("invalid or expired realtime ticket")
	}
	return record.user, nil
}
