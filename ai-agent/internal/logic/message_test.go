package logic

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gly-hub/ai-dandelion/ai-agent/internal/dao"
	"github.com/gly-hub/ai-dandelion/ai-agent/internal/model"
	aiagent "github.com/gly-hub/ai-dandelion/proto/ai-agent"
	"github.com/gly-hub/ai-dandelion/toolbox/agent"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeRunner struct {
	calls  []runnerCall
	events []agent.Event
}

type cancelableRunner struct{}

func (cancelableRunner) Stream(ctx context.Context, sessionID string, prompt string, resume bool, options agent.StreamOptions) (<-chan agent.Event, <-chan error) {
	events := make(chan agent.Event)
	errs := make(chan error, 1)
	go func() {
		defer close(events)
		defer close(errs)
		events <- agent.Event{Type: "text_delta", Text: "partial"}
		<-ctx.Done()
		errs <- ctx.Err()
	}()
	return events, errs
}

func (cancelableRunner) DeleteSession(context.Context, string) error { return nil }

type runnerCall struct {
	sessionID string
	prompt    string
	resume    bool
	skills    []string
	addDirs   []string
	mcps      int
}

func (r *fakeRunner) Stream(_ context.Context, sessionID string, prompt string, resume bool, options agent.StreamOptions) (<-chan agent.Event, <-chan error) {
	r.calls = append(r.calls, runnerCall{sessionID: sessionID, prompt: prompt, resume: resume, skills: options.Skills, addDirs: options.AddDirs, mcps: len(options.MCPServers)})
	streamEvents := r.events
	if len(streamEvents) == 0 {
		streamEvents = []agent.Event{
			{Type: "text_delta", Text: "hello"},
			{Type: "done", AgentSessionID: "sdk-session-1", Done: true},
		}
	}
	events := make(chan agent.Event, len(streamEvents))
	errs := make(chan error)
	go func() {
		defer close(events)
		defer close(errs)
		for _, event := range streamEvents {
			events <- event
		}
	}()
	return events, errs
}

func (*fakeRunner) DeleteSession(context.Context, string) error {
	return nil
}

type stubRunnerFactory struct {
	runner agent.Runner
}

func (f *stubRunnerFactory) DefaultRunner() agent.Runner {
	return f.runner
}

func (f *stubRunnerFactory) RunnerFor(context.Context, *model.AgentModel) agent.Runner {
	return f.runner
}

func (f *stubRunnerFactory) RunnerForConfig(context.Context, *model.AgentModel, AgentRuntimeOverride) agent.Runner {
	return f.runner
}

func newTestMessageLogic(db *gorm.DB, sessionDao *dao.Session, messageDao *dao.Message, runner agent.Runner) *MessageLogic {
	return NewMessageLogic(
		sessionDao,
		messageDao,
		nil,
		&stubRunnerFactory{runner: runner},
		NewAgentModelLogic(dao.NewAgentModel(db)),
		nil,
		nil,
		nil,
	)
}

func TestMessageLogicStreamPersistsConversation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Session{}, &model.Message{}, &model.SessionReference{}, &model.AgentModel{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}

	sessionDao := dao.NewSession(db)
	messageDao := dao.NewMessage(db)
	sessionLogic := NewSessionLogic(sessionDao)
	runner := &fakeRunner{}
	messageLogic := NewMessageLogic(
		sessionDao,
		messageDao,
		dao.NewSessionReference(db),
		&stubRunnerFactory{runner: runner},
		NewAgentModelLogic(dao.NewAgentModel(db)),
		nil,
		nil,
		nil,
	)

	ctx := testUserContext()
	session, err := sessionLogic.CreateSession(ctx, &aiagent.CreateSessionReq{})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	sent := make([]*aiagent.StreamMessageResp, 0)
	err = messageLogic.StreamMessage(ctx, &aiagent.StreamMessageReq{
		SessionId: session.GetId(),
		Content:   "ping",
	}, func(resp *aiagent.StreamMessageResp) error {
		sent = append(sent, resp)
		return nil
	})
	if err != nil {
		t.Fatalf("stream message: %v", err)
	}
	if len(sent) != 2 {
		t.Fatalf("expected text and done events, got %d", len(sent))
	}
	if !sent[1].GetDone() || sent[1].GetMessage().GetContent() != "hello" {
		t.Fatalf("unexpected done event: %#v", sent[1])
	}
	if sent[1].GetAgentSessionId() != "sdk-session-1" {
		t.Fatalf("unexpected agent session id: %q", sent[1].GetAgentSessionId())
	}

	messages, _, _, err := messageLogic.ListMessages(ctx, &aiagent.GetMessageReq{SessionId: session.GetId()})
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 persisted messages, got %d", len(messages))
	}
	roles := map[string]bool{}
	for _, message := range messages {
		roles[message.GetRole()] = true
	}
	if !roles[model.RoleUser] || !roles[model.RoleAssistant] {
		t.Fatalf("missing persisted roles: %#v", messages)
	}

	storedSession, err := sessionDao.Get(ctx, "user-a", session.GetId())
	if err != nil {
		t.Fatalf("get stored session: %v", err)
	}
	if storedSession.AgentSessionId != "sdk-session-1" {
		t.Fatalf("agent session id was not persisted: %q", storedSession.AgentSessionId)
	}

	err = messageLogic.StreamMessage(ctx, &aiagent.StreamMessageReq{
		SessionId: session.GetId(),
		Content:   "again",
	}, func(resp *aiagent.StreamMessageResp) error {
		return nil
	})
	if err != nil {
		t.Fatalf("stream second message: %v", err)
	}
	if len(runner.calls) != 2 {
		t.Fatalf("expected 2 runner calls, got %d", len(runner.calls))
	}
	if runner.calls[1].sessionID != "sdk-session-1" || !runner.calls[1].resume {
		t.Fatalf("second call did not resume with agent session id: %#v", runner.calls[1])
	}
}

func TestMessageLogicStreamPersistsPartialOnCancel(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Session{}, &model.Message{}, &model.SessionReference{}, &model.AgentModel{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	sessionDao := dao.NewSession(db)
	messageDao := dao.NewMessage(db)
	session, err := NewSessionLogic(sessionDao).CreateSession(testUserContext(), &aiagent.CreateSessionReq{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(testUserContext())
	done := make(chan error, 1)
	logic := newTestMessageLogic(db, sessionDao, messageDao, cancelableRunner{})
	go func() {
		done <- logic.StreamMessage(ctx, &aiagent.StreamMessageReq{SessionId: session.GetId(), Content: "ping"}, func(*aiagent.StreamMessageResp) error { return nil })
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("stream error = %v", err)
	}
	messages, _, _, err := logic.ListMessages(testUserContext(), &aiagent.GetMessageReq{SessionId: session.GetId()})
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 || messages[0].GetContent() != "partial" {
		t.Fatalf("partial messages = %#v", messages)
	}
}

func TestMessageLogicStreamPersistsStructuredParts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Session{}, &model.Message{}, &model.SessionReference{}, &model.AgentModel{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}

	sessionDao := dao.NewSession(db)
	messageDao := dao.NewMessage(db)
	sessionLogic := NewSessionLogic(sessionDao)
	runner := &fakeRunner{
		events: []agent.Event{
			{Type: "thinking_start"},
			{Type: "thinking_delta", Text: "plan"},
			{Type: "thinking_stop"},
			{Type: "tool_start", ToolID: "tool-1", ToolName: "search"},
			{Type: "tool_delta", ToolID: "tool-1", ToolInput: "{\"q\":\"go\"}"},
			{Type: "tool_stop", ToolID: "tool-1", ToolInput: "{\"q\":\"go\"}"},
			{Type: "tool_result", ToolID: "tool-1", ResultText: "ok"},
			{Type: "text_delta", Text: "answer"},
			{Type: "done", AgentSessionID: "sdk-session-2", Done: true},
		},
	}
	messageLogic := newTestMessageLogic(db, sessionDao, messageDao, runner)

	ctx := testUserContext()
	session, err := sessionLogic.CreateSession(ctx, &aiagent.CreateSessionReq{})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	var done *aiagent.StreamMessageResp
	err = messageLogic.StreamMessage(ctx, &aiagent.StreamMessageReq{
		SessionId: session.GetId(),
		Content:   "ping",
	}, func(resp *aiagent.StreamMessageResp) error {
		if resp.GetDone() {
			done = resp
		}
		return nil
	})
	if err != nil {
		t.Fatalf("stream message: %v", err)
	}
	if done == nil {
		t.Fatalf("expected done event")
	}
	if done.GetMessage().GetContent() != "answer" {
		t.Fatalf("unexpected answer: %q", done.GetMessage().GetContent())
	}

	parts := done.GetMessage().GetParts()
	if len(parts) != 3 {
		t.Fatalf("expected thinking, tool, text parts, got %#v", parts)
	}
	if parts[0].GetType() != "thinking" || parts[0].GetText() != "plan" || parts[0].GetStatus() != "finished" {
		t.Fatalf("unexpected thinking part: %#v", parts[0])
	}
	if parts[1].GetType() != "tool" || parts[1].GetToolId() != "tool-1" || parts[1].GetResult() != "ok" {
		t.Fatalf("unexpected tool part: %#v", parts[1])
	}
	if parts[2].GetType() != "text" || parts[2].GetText() != "answer" {
		t.Fatalf("unexpected text part: %#v", parts[2])
	}

	var contentPayload struct {
		Type    string           `json:"type"`
		Done    bool             `json:"done"`
		Message *aiagent.Message `json:"message"`
	}
	if err := json.Unmarshal([]byte(done.GetContent()), &contentPayload); err != nil {
		t.Fatalf("stream content is not JSON: %v content=%s", err, done.GetContent())
	}
	if contentPayload.Type != "done" || !contentPayload.Done || contentPayload.Message.GetContent() != "answer" {
		t.Fatalf("unexpected stream content payload: %#v", contentPayload)
	}
}

func TestMessageLogicStreamInjectsDefaultFunctionSkill(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Session{}, &model.Message{}, &model.SessionReference{}, &model.AgentModel{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	storageRoot := t.TempDir()
	for _, skillID := range []string{"generated-app-builder", "technical-doc-builder"} {
		systemSkillDir := filepath.Join(storageRoot, systemSkillOwner, ".claude", "skills", skillID)
		if err := os.MkdirAll(systemSkillDir, 0o755); err != nil {
			t.Fatalf("create system skill dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(systemSkillDir, "SKILL.md"), []byte("---\nname: "+skillID+"\n---\n"), 0o644); err != nil {
			t.Fatalf("write system skill: %v", err)
		}
	}

	sessionDao := dao.NewSession(db)
	messageDao := dao.NewMessage(db)
	sessionLogic := NewSessionLogic(sessionDao)
	runner := &fakeRunner{}
	messageLogic := NewMessageLogic(
		sessionDao,
		messageDao,
		dao.NewSessionReference(db),
		&stubRunnerFactory{runner: runner},
		NewAgentModelLogic(dao.NewAgentModel(db)),
		nil,
		NewSkillLogic(storageRoot),
		nil,
	)

	session, err := sessionLogic.CreateSession(testUserContext(), &aiagent.CreateSessionReq{SessionType: int32(model.SessionTypeFunction)})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	err = messageLogic.StreamMessage(testUserContext(), &aiagent.StreamMessageReq{
		SessionId:              session.GetId(),
		Content:                "生成页面",
		AgentSessionConfigType: "func_generation",
		UserId:                 "user-a",
	}, func(*aiagent.StreamMessageResp) error {
		return nil
	})
	if err != nil {
		t.Fatalf("stream message: %v", err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("unexpected runner calls: %#v", runner.calls)
	}
	if len(runner.calls[0].skills) != 1 || runner.calls[0].skills[0] != "generated-app-builder" {
		t.Fatalf("unexpected skills: %#v", runner.calls[0].skills)
	}
	if len(runner.calls[0].addDirs) != 1 || runner.calls[0].addDirs[0] != filepath.Join(storageRoot, systemSkillOwner) {
		t.Fatalf("unexpected add dirs: %#v", runner.calls[0].addDirs)
	}

	session, err = sessionLogic.CreateSession(testUserContext(), &aiagent.CreateSessionReq{SessionType: int32(model.SessionTypeFunction)})
	if err != nil {
		t.Fatalf("create technical session: %v", err)
	}
	err = messageLogic.StreamMessage(testUserContext(), &aiagent.StreamMessageReq{
		SessionId:              session.GetId(),
		Content:                "生成研发文档",
		AgentSessionConfigType: "func_technical",
		UserId:                 "user-a",
	}, func(*aiagent.StreamMessageResp) error {
		return nil
	})
	if err != nil {
		t.Fatalf("stream technical message: %v", err)
	}
	if len(runner.calls) != 2 {
		t.Fatalf("unexpected runner calls: %#v", runner.calls)
	}
	if len(runner.calls[1].skills) != 2 || runner.calls[1].skills[0] != "technical-doc-builder" || runner.calls[1].skills[1] != "generated-app-builder" {
		t.Fatalf("unexpected technical skills: %#v", runner.calls[1].skills)
	}
	if len(runner.calls[1].addDirs) != 1 || runner.calls[1].addDirs[0] != filepath.Join(storageRoot, systemSkillOwner) {
		t.Fatalf("unexpected technical add dirs: %#v", runner.calls[1].addDirs)
	}
}

func TestMessageLogicStreamUsesExtraSkills(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Session{}, &model.Message{}, &model.SessionReference{}, &model.AgentModel{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}

	sessionDao := dao.NewSession(db)
	messageDao := dao.NewMessage(db)
	sessionLogic := NewSessionLogic(sessionDao)
	runner := &fakeRunner{}
	messageLogic := NewMessageLogic(
		sessionDao,
		messageDao,
		dao.NewSessionReference(db),
		&stubRunnerFactory{runner: runner},
		NewAgentModelLogic(dao.NewAgentModel(db)),
		nil,
		nil,
		nil,
	)

	ctx := testUserContext()
	session, err := sessionLogic.CreateSession(ctx, &aiagent.CreateSessionReq{})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	err = messageLogic.StreamMessage(ctx, &aiagent.StreamMessageReq{
		SessionId: session.GetId(),
		Content:   "润色一下",
		MessageParts: []*aiagent.MessagePart{
			{Type: "text", Text: "请"},
			{Type: "skill", SkillId: "humanizer-zh", Label: "中文去 AI 味"},
			{Type: "text", Text: "润色"},
			{Type: "mcp", McpId: "filesystem", Label: "文件系统"},
		},
	}, func(*aiagent.StreamMessageResp) error {
		return nil
	})
	if err != nil {
		t.Fatalf("stream message: %v", err)
	}
	if len(runner.calls) != 1 || len(runner.calls[0].skills) != 1 || runner.calls[0].skills[0] != "humanizer-zh" {
		t.Fatalf("unexpected runner skills: %#v", runner.calls)
	}
	if runner.calls[0].prompt != "请[技能:中文去 AI 味](humanizer-zh)润色[MCP:文件系统](filesystem)" {
		t.Fatalf("unexpected runner prompt: %q", runner.calls[0].prompt)
	}

	messages, _, _, err := messageLogic.ListMessages(ctx, &aiagent.GetMessageReq{SessionId: session.GetId()})
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	var userMessage *aiagent.Message
	for _, message := range messages {
		if message.GetRole() == model.RoleUser {
			userMessage = message
			break
		}
	}
	if userMessage == nil {
		t.Fatalf("user message was not persisted: %#v", messages)
	}
	parts := userMessage.GetParts()
	if len(parts) != 4 {
		t.Fatalf("expected structured user parts, got %#v", parts)
	}
	if parts[0].GetType() != "text" || parts[0].GetText() != "请" {
		t.Fatalf("unexpected first text part: %#v", parts[0])
	}
	if parts[1].GetType() != "skill" || parts[1].GetSkillId() != "humanizer-zh" || parts[1].GetLabel() != "中文去 AI 味" {
		t.Fatalf("unexpected skill part: %#v", parts[1])
	}
	if parts[3].GetType() != "mcp" || parts[3].GetMcpId() != "filesystem" {
		t.Fatalf("unexpected mcp part: %#v", parts[3])
	}
	if len(userMessage.GetExtra()) != 2 || userMessage.GetExtra()[0].GetIndex() != 1 || userMessage.GetExtra()[1].GetIndex() != 3 {
		t.Fatalf("unexpected user message extra: %#v", userMessage.GetExtra())
	}

	err = messageLogic.StreamMessage(ctx, &aiagent.StreamMessageReq{
		SessionId: session.GetId(),
		Content:   "继续",
	}, func(*aiagent.StreamMessageResp) error {
		return nil
	})
	if err != nil {
		t.Fatalf("stream message without explicit skill: %v", err)
	}
	if len(runner.calls) != 2 || len(runner.calls[1].skills) != 1 || runner.calls[1].skills[0] != "humanizer-zh" {
		t.Fatalf("expected persisted skill reference on next turn, got %#v", runner.calls)
	}
}

func TestMessageLogicStreamUsesMCPServers(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Session{}, &model.Message{}, &model.SessionReference{}, &model.AgentModel{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}

	sessionDao := dao.NewSession(db)
	messageDao := dao.NewMessage(db)
	sessionLogic := NewSessionLogic(sessionDao)
	runner := &fakeRunner{}
	mcpLogic := NewMCPLogic(t.TempDir())
	messageLogic := NewMessageLogic(
		sessionDao,
		messageDao,
		dao.NewSessionReference(db),
		&stubRunnerFactory{runner: runner},
		NewAgentModelLogic(dao.NewAgentModel(db)),
		nil,
		nil,
		mcpLogic,
	)

	ctx := testUserContext()
	if _, err := mcpLogic.CreateMCPServer(ctx, &aiagent.SaveMCPServerReq{
		UserId: "user-a",
		Server: &aiagent.AgentMCPServer{
			Id:      "filesystem",
			Name:    "文件系统",
			Type:    "stdio",
			Enabled: true,
			Command: "npx",
			Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
		},
	}); err != nil {
		t.Fatalf("create mcp server: %v", err)
	}
	session, err := sessionLogic.CreateSession(ctx, &aiagent.CreateSessionReq{})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	err = messageLogic.StreamMessage(ctx, &aiagent.StreamMessageReq{
		SessionId: session.GetId(),
		UserId:    "user-a",
		Content:   "列一下文件",
		Extra: []*aiagent.MessageExtra{
			{Type: "mcp", Id: "filesystem", Name: "文件系统"},
		},
	}, func(*aiagent.StreamMessageResp) error {
		return nil
	})
	if err != nil {
		t.Fatalf("stream message: %v", err)
	}
	if len(runner.calls) != 1 || runner.calls[0].mcps != 1 {
		t.Fatalf("unexpected runner MCP servers: %#v", runner.calls)
	}

	err = messageLogic.StreamMessage(ctx, &aiagent.StreamMessageReq{
		SessionId: session.GetId(),
		UserId:    "user-a",
		Content:   "继续",
	}, func(*aiagent.StreamMessageResp) error {
		return nil
	})
	if err != nil {
		t.Fatalf("stream message without explicit mcp: %v", err)
	}
	if len(runner.calls) != 2 || runner.calls[1].mcps != 1 {
		t.Fatalf("expected persisted MCP reference on next turn, got %#v", runner.calls)
	}
}

func TestMessageLogicListMessagesPagination(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Session{}, &model.Message{}, &model.SessionReference{}, &model.AgentModel{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}

	sessionDao := dao.NewSession(db)
	messageDao := dao.NewMessage(db)
	sessionLogic := NewSessionLogic(sessionDao)
	messageLogic := newTestMessageLogic(db, sessionDao, messageDao, &fakeRunner{})

	ctx := testUserContext()
	session, err := sessionLogic.CreateSession(ctx, &aiagent.CreateSessionReq{Title: "Paged chat"})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	for _, content := range []string{"first", "second", "third"} {
		if _, err := messageLogic.addMessage(ctx, session.GetId(), model.RoleUser, content, defaultParts(content)); err != nil {
			t.Fatalf("add message %q: %v", content, err)
		}
	}

	firstPage, hasMore, nextBefore, err := messageLogic.ListMessages(ctx, &aiagent.GetMessageReq{
		SessionId: session.GetId(),
		Limit:     2,
	})
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if len(firstPage) != 2 || !hasMore || nextBefore == "" {
		t.Fatalf("unexpected first page: messages=%#v hasMore=%v nextBefore=%q", firstPage, hasMore, nextBefore)
	}
	if firstPage[0].GetContent() != "third" || firstPage[1].GetContent() != "second" {
		t.Fatalf("unexpected first page order: %#v", firstPage)
	}

	secondPage, hasMore, _, err := messageLogic.ListMessages(ctx, &aiagent.GetMessageReq{
		SessionId: session.GetId(),
		Limit:     2,
		Before:    nextBefore,
	})
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if len(secondPage) != 1 || hasMore || secondPage[0].GetContent() != "first" {
		t.Fatalf("unexpected second page: messages=%#v hasMore=%v", secondPage, hasMore)
	}
}

func TestMessageLogicStreamValidatesContent(t *testing.T) {
	messageLogic := newTestMessageLogic(nil, nil, nil, &fakeRunner{})
	err := messageLogic.StreamMessage(testUserContext(), &aiagent.StreamMessageReq{
		SessionId: "s1",
		Content:   "  ",
	}, func(*aiagent.StreamMessageResp) error {
		return nil
	})
	if err == nil || !errors.Is(err, errContentRequired) {
		t.Fatalf("expected content required error, got %v", err)
	}
}
