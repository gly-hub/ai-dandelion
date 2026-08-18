package logic

import (
	"context"
	"errors"
	"strings"

	"github.com/gly-hub/ai-dandelion/ai-agent/internal/dao"
	"github.com/gly-hub/ai-dandelion/ai-agent/internal/model"
	aiagent "github.com/gly-hub/ai-dandelion/proto/ai-agent"
	"github.com/gly-hub/ai-dandelion/toolbox/agent"
	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SessionLogic struct {
	sessionDao *dao.Session
	runner     agent.Runner
}

const maxSessionTitleLength = 200

func NewSessionLogic(sessionDao *dao.Session, runners ...agent.Runner) *SessionLogic {
	logic := &SessionLogic{
		sessionDao: sessionDao,
	}
	if len(runners) > 0 {
		logic.runner = runners[0]
	}
	return logic
}

func (s *SessionLogic) ListSessions(ctx context.Context, req *aiagent.SearchMessageReq) (
	[]*aiagent.Session, error) {
	userID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return nil, err
	}
	sessions, err := s.sessionDao.List(ctx, userID, req.GetSessionType())
	if err != nil {
		return nil, err
	}

	out := make([]*aiagent.Session, 0, len(sessions))
	for i := range sessions {
		out = append(out, modelSessionToProto(&sessions[i]))
	}
	return out, nil
}

func (s *SessionLogic) CreateSession(ctx context.Context, req *aiagent.CreateSessionReq) (
	*aiagent.Session, error) {
	userID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return nil, err
	}
	now := nowUnixMicro()
	title := strings.TrimSpace(req.GetTitle())
	if title == "" {
		title = model.DefaultSessionTitle
	}

	sessionType := int(req.GetSessionType())
	if sessionType <= 0 {
		sessionType = model.SessionTypeNormal
	}

	session := &model.Session{
		ID:          uuid.NewString(),
		UserID:      userID,
		Title:       title,
		SessionType: sessionType,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.sessionDao.Create(ctx, session); err != nil {
		return nil, err
	}
	return modelSessionToProto(session), nil
}

func (s *SessionLogic) EnsureSession(ctx context.Context, req *aiagent.EnsureSessionReq) (*aiagent.Session, bool, error) {
	userID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return nil, false, err
	}
	sessionID := strings.TrimSpace(req.GetId())
	if sessionID == "" {
		return nil, false, errors.New("session id is required")
	}

	existing, err := s.sessionDao.Get(ctx, userID, sessionID)
	if err == nil {
		return modelSessionToProto(existing), false, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	now := nowUnixMicro()
	title := strings.TrimSpace(req.GetTitle())
	if title == "" {
		title = model.DefaultSessionTitle
	}
	sessionType := int(req.GetSessionType())
	if sessionType <= 0 {
		sessionType = model.SessionTypeNormal
	}

	session := &model.Session{
		ID:          sessionID,
		UserID:      userID,
		Title:       title,
		SessionType: sessionType,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.sessionDao.Create(ctx, session); err != nil {
		if existing, getErr := s.sessionDao.Get(ctx, userID, sessionID); getErr == nil {
			return modelSessionToProto(existing), false, nil
		}
		return nil, false, err
	}
	return modelSessionToProto(session), true, nil
}

func (s *SessionLogic) UpdateSession(ctx context.Context, req *aiagent.UpdateSessionReq) (*aiagent.Session, error) {
	userID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return nil, err
	}
	sessionID := strings.TrimSpace(req.GetId())
	if sessionID == "" {
		return nil, errors.New("session id is required")
	}
	title := strings.TrimSpace(req.GetTitle())
	if title == "" {
		return nil, errors.New("session title is required")
	}
	if len([]rune(title)) > maxSessionTitleLength {
		return nil, errors.New("session title must not exceed 200 characters")
	}

	session, err := s.sessionDao.Get(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	now := nowUnixMicro()
	if err := s.sessionDao.UpdateTitle(ctx, userID, sessionID, title, now); err != nil {
		return nil, err
	}
	session.Title = title
	session.UpdatedAt = now
	return modelSessionToProto(session), nil
}

func (s *SessionLogic) DeleteSession(ctx context.Context, req *aiagent.DeleteSessionReq) error {
	userID, err := authctx.RequireUserID(ctx)
	if err != nil {
		return err
	}
	sessionID := strings.TrimSpace(req.GetId())
	if sessionID == "" {
		return errors.New("session id is required")
	}
	session, err := s.sessionDao.Get(ctx, userID, sessionID)
	if err != nil {
		return err
	}
	if err := s.sessionDao.Delete(ctx, userID, sessionID); err != nil {
		return err
	}
	if s.runner == nil || strings.TrimSpace(session.AgentSessionId) == "" {
		return nil
	}
	return s.runner.DeleteSession(ctx, session.AgentSessionId)
}

func modelSessionToProto(session *model.Session) *aiagent.Session {
	if session == nil {
		return nil
	}
	return &aiagent.Session{
		Id:          session.ID,
		Title:       session.Title,
		CreatedAt:   session.CreatedAt,
		UpdatedAt:   session.UpdatedAt,
		SessionType: int32(session.SessionType),
	}
}
