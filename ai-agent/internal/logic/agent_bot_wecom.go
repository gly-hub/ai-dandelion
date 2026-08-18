package logic

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gly-hub/ai-dandelion/ai-agent/internal/dao"
	"github.com/gly-hub/ai-dandelion/ai-agent/internal/model"
	aiagent "github.com/gly-hub/ai-dandelion/proto/ai-agent"
	"github.com/gly-hub/ai-dandelion/toolbox/agent"
	"github.com/gly-hub/ai-dandelion/toolbox/authctx"
	"github.com/gly-hub/ai-dandelion/toolbox/wecomaibot"
	"github.com/gly-hub/quickgo/logger"
)

const wecomStreamUpdateInterval = 800 * time.Millisecond

type wecomAgentBotWorker struct {
	client *wecomaibot.Client
}

func newWeComAgentBotWorker(
	runtime *AgentBotRuntime,
	item dao.AgentBotAggregate,
	channel model.AgentBotChannel,
	engineConfig AgentEngineRunConfig,
) (agentBotChannelWorker, string, error) {
	if strings.TrimSpace(channel.ExternalBotID) == "" || strings.TrimSpace(channel.Secret) == "" {
		return nil, "", errors.New("missing wecom bot id or secret")
	}
	handler := &wecomRuntimeHandler{
		runtime:      runtime,
		bot:          item,
		channel:      channel,
		engineConfig: engineConfig,
	}
	config := wecomaibot.DecodeConfig(channel.ConfigJSON)
	client := wecomaibot.NewClient(wecomaibot.Options{
		BotID:                channel.ExternalBotID,
		Secret:               channel.Secret,
		WSURL:                channel.EndpointURL,
		HeartbeatInterval:    time.Duration(wecomaibot.ConfigInt(config, "heartbeatInterval", 30000)) * time.Millisecond,
		MaxReconnectAttempts: wecomaibot.ConfigInt(config, "maxReconnectAttempts", -1),
	}, handler)
	key := channel.ID
	if key == "" {
		key = fmt.Sprintf("%s:%s", item.Bot.ID, channel.ExternalBotID)
	}
	return &wecomAgentBotWorker{client: client}, key, nil
}

func (w *wecomAgentBotWorker) Start(ctx context.Context) error {
	if w == nil || w.client == nil {
		return nil
	}
	return w.client.Start(ctx)
}

func (w *wecomAgentBotWorker) Stop() {
	if w == nil || w.client == nil {
		return
	}
	w.client.Stop()
}

type wecomRuntimeHandler struct {
	runtime      *AgentBotRuntime
	bot          dao.AgentBotAggregate
	channel      model.AgentBotChannel
	engineConfig AgentEngineRunConfig
}

func (h *wecomRuntimeHandler) HandleEvent(ctx context.Context, client *wecomaibot.Client, event wecomaibot.Event) {
	switch event.Type {
	case "message.text":
		h.handleText(ctx, client, event.Frame)
	case "event.enter_chat":
		if h.bot.Bot.WelcomeMessage != "" {
			if err := client.ReplyWelcome(event.Frame, h.bot.Bot.WelcomeMessage); err != nil {
				logger.Error(ctx, "reply wecom welcome failed: %v", err)
			}
		}
	}
}

func (h *wecomRuntimeHandler) HandleError(ctx context.Context, err error) {
	logger.Error(ctx, "wecom bot runtime error: %v", err)
}

func (h *wecomRuntimeHandler) handleText(ctx context.Context, client *wecomaibot.Client, frame map[string]any) {
	content := wecomaibot.TextContent(frame)
	if content == "" {
		return
	}
	sessionID := fmt.Sprintf("channel:wecom:%s:%s", h.channel.ID, wecomaibot.ConversationKey(frame))
	sessionCtx := authctx.ContextWithUser(ctx, authctx.User{ID: "channel:" + h.channel.ID})
	if _, _, err := h.runtime.sessionLogic.EnsureSession(sessionCtx, &aiagent.EnsureSessionReq{
		Id:          sessionID,
		Title:       h.bot.Bot.Name,
		SessionType: int32(model.SessionTypeChannel),
	}); err != nil {
		logger.Error(ctx, "ensure wecom session failed: %v", err)
		return
	}

	streamID := wecomaibot.ReqID("stream")
	accumulated := strings.Builder{}
	lastPushAt := time.Time{}
	if err := client.ReplyStream(frame, streamID, "正在思考...", false); err != nil {
		logger.Error(ctx, "reply wecom thinking failed: %v", err)
	}
	req := &agentBotRunRequest{
		SessionID:      sessionID,
		Content:        content,
		EngineConfig:   h.engineConfig,
		AgentSessionID: "",
	}
	session, err := h.runtime.messageLogic.sessionDao.Get(sessionCtx, "channel:"+h.channel.ID, sessionID)
	if err == nil {
		req.AgentSessionID = session.AgentSessionId
	} else {
		logger.Error(ctx, "get wecom session failed: %v", err)
	}
	err = h.runtime.streamAgentBotMessage(sessionCtx, req, func(event agentBotStreamEvent) error {
		text := formatWeComAgentEvent(event.Event)
		if text != "" {
			accumulated.WriteString(text)
			now := time.Now()
			if !lastPushAt.IsZero() && now.Sub(lastPushAt) < wecomStreamUpdateInterval {
				return nil
			}
			lastPushAt = now
			return client.ReplyStream(frame, streamID, accumulated.String(), false)
		}
		if event.Done {
			finalText := accumulated.String()
			if finalText == "" {
				finalText = "已完成。"
			}
			return client.ReplyStream(frame, streamID, finalText, true)
		}
		return nil
	})
	if err != nil {
		logger.Error(ctx, "stream wecom message failed: %v", err)
		_ = client.ReplyStream(frame, streamID, "处理失败，请稍后重试。", true)
	}
}

func formatWeComAgentEvent(event agent.Event) string {
	switch event.Type {
	case "text_delta":
		return event.Text
	case "thinking_start":
		return "\n\n[思考]\n"
	case "thinking_delta":
		return event.Text
	case "thinking_stop":
		return "\n[/思考]\n"
	case "tool_start":
		name := strings.TrimSpace(event.ToolName)
		if name == "" {
			name = strings.TrimSpace(event.ToolID)
		}
		if name == "" {
			name = "unknown"
		}
		return fmt.Sprintf("\n\n[工具调用] %s\n", name)
	case "tool_delta":
		return ""
	case "tool_stop":
		return "[工具调用完成]\n"
	case "tool_result":
		if event.IsError {
			result := strings.TrimSpace(event.ResultText)
			if result == "" {
				result = "无错误详情"
			}
			return fmt.Sprintf("[工具结果:失败]\n%s\n", result)
		}
		return "[工具结果:成功]\n"
	default:
		return ""
	}
}
