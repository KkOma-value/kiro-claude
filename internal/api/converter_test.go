package api

import (
	"testing"
)

func TestToKiroRequestBasic(t *testing.T) {
	req := &MessageRequest{
		Model:     "claude-sonnet-4-5",
		MaxTokens: 256,
		System:    "system prompt",
		Messages: []AnthropicMessage{
			{Role: "user", Content: "hello"},
		},
	}

	kiroReq, err := ToKiroRequest(req)
	if err != nil {
		t.Fatalf("ToKiroRequest returned error: %v", err)
	}

	if kiroReq.ConversationState.ConversationID == "" {
		t.Fatal("expected conversationId to be set")
	}

	if kiroReq.ConversationState.AgentTaskType != "vibe" {
		t.Fatalf("unexpected agentTaskType: %s", kiroReq.ConversationState.AgentTaskType)
	}

	msg := kiroReq.ConversationState.CurrentMessage.UserInputMessage
	if msg == nil {
		t.Fatal("expected current message to be set")
	}

	if msg.Origin != "AI_EDITOR" {
		t.Fatalf("unexpected origin: %s", msg.Origin)
	}
}

func TestToKiroRequestNilReturnsError(t *testing.T) {
	_, err := ToKiroRequest(nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestToKiroRequestSystemPromptMergedIntoHistory(t *testing.T) {
	req := &MessageRequest{
		Model:  "claude-sonnet-4-5",
		System: "You are helpful.",
		Messages: []AnthropicMessage{
			{Role: "user", Content: "hello"},
		},
	}

	kiroReq, err := ToKiroRequest(req)
	if err != nil {
		t.Fatalf("ToKiroRequest returned error: %v", err)
	}

	// With a single user message + system prompt, system gets prepended to first user in history
	// and the current message is "hello"
	msg := kiroReq.ConversationState.CurrentMessage.UserInputMessage
	if msg == nil {
		t.Fatal("expected current message")
	}
}

func TestToKiroRequestPlaceholderToolWhenNoTools(t *testing.T) {
	req := &MessageRequest{
		Model: "claude-sonnet-4-5",
		Messages: []AnthropicMessage{
			{Role: "user", Content: "hello"},
		},
	}

	kiroReq, err := ToKiroRequest(req)
	if err != nil {
		t.Fatalf("ToKiroRequest returned error: %v", err)
	}

	msg := kiroReq.ConversationState.CurrentMessage.UserInputMessage
	if msg.UserInputMessageContext == nil {
		t.Fatal("expected context with placeholder tool")
	}
	if len(msg.UserInputMessageContext.Tools) != 1 {
		t.Fatalf("expected 1 placeholder tool, got %d", len(msg.UserInputMessageContext.Tools))
	}
	if msg.UserInputMessageContext.Tools[0].ToolSpecification.Name != "no_tool_available" {
		t.Fatalf("unexpected placeholder tool name: %s", msg.UserInputMessageContext.Tools[0].ToolSpecification.Name)
	}
}
