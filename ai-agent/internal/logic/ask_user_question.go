package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/team-dandelion/ai-dandelion/toolbox/agent"
)

var (
	errQuestionNotPending = errors.New("ask user question is no longer pending")
	errInvalidAnswers     = errors.New("answers must be a JSON object")
)

type askUserQuestionPending struct {
	sessionID string
	toolID    string
	questions any
	answerCh  chan map[string]any
}

// AskUserQuestionBroker owns the live native tool callbacks for this process.
// Claude's own session remains paused until Submit returns an answer to answerCh.
type AskUserQuestionBroker struct {
	mu      sync.Mutex
	pending map[string]*askUserQuestionPending
}

func NewAskUserQuestionBroker() *AskUserQuestionBroker {
	return &AskUserQuestionBroker{pending: make(map[string]*askUserQuestionPending)}
}

func (b *AskUserQuestionBroker) Wait(
	ctx context.Context,
	sessionID string,
	req agent.AskUserQuestionRequest,
	emit func(agent.Event) bool,
) (map[string]any, error) {
	if b == nil {
		return nil, errors.New("ask user question broker is not configured")
	}
	if req.ToolID == "" {
		return nil, errors.New("ask user question tool id is required")
	}
	questions, ok := req.Input["questions"]
	if !ok || !validQuestions(questions) {
		return nil, errors.New("invalid AskUserQuestion input")
	}

	pending := &askUserQuestionPending{
		sessionID: sessionID,
		toolID:    req.ToolID,
		questions: questions,
		answerCh:  make(chan map[string]any, 1),
	}
	b.mu.Lock()
	if _, exists := b.pending[req.ToolID]; exists {
		b.mu.Unlock()
		return nil, errors.New("ask user question is already pending")
	}
	b.pending[req.ToolID] = pending
	b.mu.Unlock()
	defer b.remove(req.ToolID)

	input, _ := json.Marshal(req.Input)
	if !emit(agent.Event{
		Type:      "ask_user_question",
		ToolID:    req.ToolID,
		ToolName:  req.ToolName,
		ToolInput: string(input),
	}) {
		return nil, ctx.Err()
	}

	select {
	case answer := <-pending.answerCh:
		return answer, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (b *AskUserQuestionBroker) Submit(sessionID string, toolID string, answersJSON string, response string) error {
	if b == nil {
		return errors.New("ask user question broker is not configured")
	}
	var answers map[string]any
	if err := json.Unmarshal([]byte(answersJSON), &answers); err != nil || answers == nil {
		return errInvalidAnswers
	}
	b.mu.Lock()
	pending := b.pending[toolID]
	if pending == nil || pending.sessionID != sessionID {
		b.mu.Unlock()
		return errQuestionNotPending
	}
	if err := validateAnswers(pending.questions, answers, response); err != nil {
		b.mu.Unlock()
		return err
	}
	delete(b.pending, toolID)
	b.mu.Unlock()

	updatedInput := map[string]any{
		"questions": pending.questions,
		"answers":   answers,
	}
	if response != "" {
		updatedInput["response"] = response
	}
	pending.answerCh <- updatedInput
	return nil
}

func (b *AskUserQuestionBroker) remove(toolID string) {
	b.mu.Lock()
	delete(b.pending, toolID)
	b.mu.Unlock()
}

func validQuestions(value any) bool {
	questions, ok := value.([]any)
	if !ok || len(questions) == 0 || len(questions) > 4 {
		return false
	}
	for _, raw := range questions {
		question, ok := raw.(map[string]any)
		if !ok || stringValue(question["question"]) == "" || stringValue(question["header"]) == "" {
			return false
		}
		options, ok := question["options"].([]any)
		if !ok || len(options) < 2 || len(options) > 4 {
			return false
		}
	}
	return true
}

func validateAnswers(questions any, answers map[string]any, response string) error {
	if response != "" {
		return nil
	}
	for _, raw := range questions.([]any) {
		question := raw.(map[string]any)
		key := stringValue(question["question"])
		answer, exists := answers[key]
		if !exists || !validAnswerValue(answer, question["multiSelect"] == true) {
			return fmt.Errorf("missing or invalid answer for %q", key)
		}
	}
	return nil
}

func validAnswerValue(value any, multiSelect bool) bool {
	if multiSelect {
		items, ok := value.([]any)
		return ok && len(items) > 0 && allStrings(items)
	}
	return stringValue(value) != ""
}

func allStrings(items []any) bool {
	for _, item := range items {
		if stringValue(item) == "" {
			return false
		}
	}
	return true
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
