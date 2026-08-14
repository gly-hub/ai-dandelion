package logic

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/team-dandelion/ai-dandelion/ai-agent/internal/dao"
	"github.com/team-dandelion/ai-dandelion/ai-agent/internal/model"
	aiagent "github.com/team-dandelion/ai-dandelion/proto/ai-agent"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSessionLogicUpdateSession(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Session{}); err != nil {
		t.Fatalf("migrate session table: %v", err)
	}

	ctx := context.Background()
	sessionDao := dao.NewSession(db)
	logic := NewSessionLogic(sessionDao)
	session, err := logic.CreateSession(ctx, &aiagent.CreateSessionReq{Title: "Original title"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	updated, err := logic.UpdateSession(ctx, &aiagent.UpdateSessionReq{
		Id:    session.GetId(),
		Title: "  Renamed session  ",
	})
	if err != nil {
		t.Fatalf("update session: %v", err)
	}
	if updated.GetTitle() != "Renamed session" {
		t.Fatalf("unexpected updated title: %q", updated.GetTitle())
	}
	stored, err := sessionDao.Get(ctx, session.GetId())
	if err != nil {
		t.Fatalf("get updated session: %v", err)
	}
	if stored.Title != "Renamed session" || stored.UpdatedAt != updated.GetUpdatedAt() {
		t.Fatalf("updated session was not persisted: %#v", stored)
	}

	if _, err := logic.UpdateSession(ctx, &aiagent.UpdateSessionReq{Id: session.GetId(), Title: "  "}); err == nil {
		t.Fatal("expected blank title to be rejected")
	}
	if _, err := logic.UpdateSession(ctx, &aiagent.UpdateSessionReq{Id: session.GetId(), Title: strings.Repeat("a", 201)}); err == nil {
		t.Fatal("expected overlong title to be rejected")
	}
}

func TestSessionLogicCreateListDelete(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Session{}, &model.Message{}, &model.SessionReference{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}

	logic := NewSessionLogic(dao.NewSession(db))
	ctx := context.Background()

	first, err := logic.CreateSession(ctx, &aiagent.CreateSessionReq{Title: "  "})
	if err != nil {
		t.Fatalf("create default session: %v", err)
	}
	if first.GetTitle() != model.DefaultSessionTitle {
		t.Fatalf("unexpected default title: %q", first.GetTitle())
	}

	time.Sleep(time.Second)
	second, err := logic.CreateSession(ctx, &aiagent.CreateSessionReq{Title: "Postman test"})
	if err != nil {
		t.Fatalf("create named session: %v", err)
	}

	sessions, err := logic.ListSessions(ctx, &aiagent.SearchMessageReq{})
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	if sessions[0].GetId() != second.GetId() || sessions[1].GetId() != first.GetId() {
		t.Fatalf("sessions were not returned updated-first: %#v", sessions)
	}

	functionSession, err := logic.CreateSession(ctx, &aiagent.CreateSessionReq{
		Title:       "Function draft",
		SessionType: model.SessionTypeFunction,
	})
	if err != nil {
		t.Fatalf("create function session: %v", err)
	}
	filtered, err := logic.ListSessions(ctx, &aiagent.SearchMessageReq{SessionType: model.SessionTypeFunction})
	if err != nil {
		t.Fatalf("list function sessions: %v", err)
	}
	if len(filtered) != 1 || filtered[0].GetId() != functionSession.GetId() {
		t.Fatalf("unexpected function sessions: %#v", filtered)
	}

	if err := logic.DeleteSession(ctx, &aiagent.DeleteSessionReq{Id: first.GetId()}); err != nil {
		t.Fatalf("delete session: %v", err)
	}

	ensured, created, err := logic.EnsureSession(ctx, &aiagent.EnsureSessionReq{
		Id:          "42:product",
		Title:       "Function product",
		SessionType: model.SessionTypeFunction,
	})
	if err != nil {
		t.Fatalf("ensure new session: %v", err)
	}
	if created != true || ensured.GetId() != "42:product" {
		t.Fatalf("unexpected ensure create result: created=%v id=%q", created, ensured.GetId())
	}

	again, createdAgain, err := logic.EnsureSession(ctx, &aiagent.EnsureSessionReq{
		Id:          "42:product",
		Title:       "Function product",
		SessionType: model.SessionTypeFunction,
	})
	if err != nil {
		t.Fatalf("ensure existing session: %v", err)
	}
	if createdAgain != false || again.GetId() != "42:product" {
		t.Fatalf("unexpected ensure existing result: created=%v id=%q", createdAgain, again.GetId())
	}

	filtered, err = logic.ListSessions(ctx, &aiagent.SearchMessageReq{SessionType: model.SessionTypeFunction})
	if err != nil {
		t.Fatalf("list function sessions after ensure: %v", err)
	}
	if len(filtered) != 2 {
		t.Fatalf("expected 2 function sessions, got %d", len(filtered))
	}

	sessions, err = logic.ListSessions(ctx, &aiagent.SearchMessageReq{SessionType: model.SessionTypeNormal})
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(sessions) != 1 || sessions[0].GetId() != second.GetId() {
		t.Fatalf("unexpected sessions after delete: %#v", sessions)
	}
}
