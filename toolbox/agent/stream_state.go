package agent

import (
	"fmt"
	"strings"

	claudeagentsdk "github.com/gly-hub/claude-agent-sdk-go"
	"github.com/gly-hub/quickgo/json"
)

type streamState struct {
	sawStreamContent  bool
	lastAssistantText string
	lastThinkingText  string
	toolInputs        map[string]*strings.Builder
	toolNames         map[string]string
	toolIndex         map[int]string
	blockTypes        map[int]string
}

func newStreamState() *streamState {
	return &streamState{
		toolInputs: make(map[string]*strings.Builder),
		toolNames:  make(map[string]string),
		toolIndex:  make(map[int]string),
		blockTypes: make(map[int]string),
	}
}

func (s *streamState) eventsFromMessage(msg claudeagentsdk.Message) []Event {
	switch m := msg.(type) {
	case *claudeagentsdk.StreamEvent:
		return s.eventsFromStream(m)
	case *claudeagentsdk.AssistantMessage:
		return s.eventsFromAssistant(m)
	case *claudeagentsdk.UserMessage:
		return s.eventsFromUser(m)
	default:
		return nil
	}
}

func (s *streamState) eventsFromStream(stream *claudeagentsdk.StreamEvent) []Event {
	rawEvent := stream.Event
	if len(rawEvent) == 0 {
		return nil
	}

	switch stringFromAny(rawEvent["type"]) {
	case "content_block_start":
		return s.contentBlockStart(rawEvent)
	case "content_block_delta":
		return s.contentBlockDelta(rawEvent)
	case "content_block_stop":
		return s.contentBlockStop(rawEvent)
	default:
		return nil
	}
}

func (s *streamState) contentBlockStart(rawEvent map[string]any) []Event {
	index := intFromAny(rawEvent["index"])
	block, _ := rawEvent["content_block"].(map[string]any)
	blockType := stringFromAny(block["type"])
	s.blockTypes[index] = blockType

	switch blockType {
	case "text":
		text := stringFromAny(block["text"])
		if text == "" {
			return nil
		}
		s.sawStreamContent = true
		return []Event{{Type: "text_delta", Text: text}}
	case "thinking":
		s.sawStreamContent = true
		events := []Event{{Type: "thinking_start"}}
		text := stringFromAny(block["thinking"])
		if text != "" {
			s.lastThinkingText = text
			events = append(events, Event{Type: "thinking_delta", Text: text})
		}
		return events
	case "tool_use":
		s.sawStreamContent = true
		toolID := stringFromAny(block["id"])
		if toolID == "" {
			toolID = fmt.Sprintf("tool-%d", index)
		}
		s.toolIndex[index] = toolID
		s.toolInputs[toolID] = &strings.Builder{}
		s.toolNames[toolID] = stringFromAny(block["name"])
		return []Event{{
			Type:     "tool_start",
			ToolID:   toolID,
			ToolName: stringFromAny(block["name"]),
		}}
	default:
		return nil
	}
}

func (s *streamState) contentBlockDelta(rawEvent map[string]any) []Event {
	delta, _ := rawEvent["delta"].(map[string]any)
	switch stringFromAny(delta["type"]) {
	case "text_delta":
		text := stringFromAny(delta["text"])
		if text == "" {
			return nil
		}
		s.sawStreamContent = true
		return []Event{{Type: "text_delta", Text: text}}
	case "thinking_delta":
		text := firstNonEmpty(stringFromAny(delta["thinking"]), stringFromAny(delta["text"]))
		if text == "" {
			return nil
		}
		s.sawStreamContent = true
		return []Event{{Type: "thinking_delta", Text: text}}
	case "input_json_delta":
		index := intFromAny(rawEvent["index"])
		toolID := s.toolIndex[index]
		inputDelta := stringFromAny(delta["partial_json"])
		if toolID == "" || inputDelta == "" {
			return nil
		}
		if s.toolInputs[toolID] == nil {
			s.toolInputs[toolID] = &strings.Builder{}
		}
		s.toolInputs[toolID].WriteString(inputDelta)
		return []Event{{
			Type:      "tool_delta",
			ToolID:    toolID,
			ToolInput: inputDelta,
		}}
	default:
		return nil
	}
}

func (s *streamState) contentBlockStop(rawEvent map[string]any) []Event {
	index := intFromAny(rawEvent["index"])
	blockType := s.blockTypes[index]
	delete(s.blockTypes, index)
	if blockType == "thinking" {
		s.lastThinkingText = ""
		return []Event{{Type: "thinking_stop"}}
	}

	toolID := s.toolIndex[index]
	if toolID == "" {
		return nil
	}
	input := ""
	if builder := s.toolInputs[toolID]; builder != nil {
		input = prettyJSON(builder.String())
	}
	delete(s.toolIndex, index)
	return []Event{{
		Type:      "tool_stop",
		ToolID:    toolID,
		ToolInput: input,
	}}
}

func (s *streamState) eventsFromAssistant(message *claudeagentsdk.AssistantMessage) []Event {
	if s.sawStreamContent {
		return nil
	}

	events := make([]Event, 0)
	for _, block := range message.Content {
		switch b := block.(type) {
		case claudeagentsdk.TextBlock:
			if b.Text == "" {
				continue
			}
			delta := b.Text
			if strings.HasPrefix(b.Text, s.lastAssistantText) {
				delta = strings.TrimPrefix(b.Text, s.lastAssistantText)
			}
			s.lastAssistantText = b.Text
			if delta != "" {
				events = append(events, Event{Type: "text_delta", Text: delta})
			}
		case claudeagentsdk.ThinkingBlock:
			if b.Thinking == "" {
				continue
			}
			delta := b.Thinking
			if strings.HasPrefix(b.Thinking, s.lastThinkingText) {
				delta = strings.TrimPrefix(b.Thinking, s.lastThinkingText)
			}
			s.lastThinkingText = b.Thinking
			events = append(events, Event{Type: "thinking_start"})
			if delta != "" {
				events = append(events, Event{Type: "thinking_delta", Text: delta})
			}
			events = append(events, Event{Type: "thinking_stop"})
		case claudeagentsdk.ToolUseBlock:
			s.toolNames[b.ID] = b.Name
			events = append(events, Event{
				Type:      "tool_start",
				ToolID:    b.ID,
				ToolName:  b.Name,
				ToolInput: prettyAny(b.Input),
			})
			events = append(events, Event{
				Type:      "tool_stop",
				ToolID:    b.ID,
				ToolInput: prettyAny(b.Input),
			})
		}
	}
	return events
}

func (s *streamState) eventsFromUser(message *claudeagentsdk.UserMessage) []Event {
	blocks, ok := message.Content.([]claudeagentsdk.ContentBlock)
	if !ok {
		return nil
	}
	events := make([]Event, 0)
	for _, block := range blocks {
		result, ok := block.(claudeagentsdk.ToolResultBlock)
		if !ok {
			continue
		}
		events = append(events, Event{
			Type:       "tool_result",
			ToolID:     result.ToolUseID,
			ToolName:   s.toolNames[result.ToolUseID],
			IsError:    result.IsError,
			ResultText: prettyAny(result.Content),
		})
	}
	return events
}

func prettyAny(value any) string {
	if value == nil {
		return ""
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(data)
}

func prettyJSON(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	var decoded any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return value
	}
	return prettyAny(decoded)
}
