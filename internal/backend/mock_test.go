package backend

import (
	"context"
	"strings"
	"testing"

	"github.com/yourusername/kiro-claude/internal/config"
	"github.com/yourusername/kiro-claude/internal/gateway"
)

func TestMockClientSendRequestReturnsText(t *testing.T) {
	client := NewMockClient(config.DefaultConfig())

	text, toolEvents, err := client.SendRequest(context.Background(), &gateway.KiroRequest{
		ConversationState: gateway.ConversationState{
			AgentTaskType:   "vibe",
			ChatTriggerType: "MANUAL",
			ConversationID:  "test-conv",
			CurrentMessage: gateway.CurrentMessage{
				UserInputMessage: &gateway.UserInputMessage{
					Content: "hello",
					ModelID: "claude-sonnet-4.5",
					Origin:  "AI_EDITOR",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("SendRequest returned error: %v", err)
	}

	if !strings.Contains(text, "hello") {
		t.Fatalf("expected response to contain user message, got %q", text)
	}

	if len(toolEvents) != 0 {
		t.Fatalf("expected no tool events, got %d", len(toolEvents))
	}
}

func TestMockClientSendStreamRequestReturnsEvents(t *testing.T) {
	client := NewMockClient(config.DefaultConfig())

	eventCh, errCh, err := client.SendStreamRequest(context.Background(), &gateway.KiroRequest{
		ConversationState: gateway.ConversationState{
			AgentTaskType:   "vibe",
			ChatTriggerType: "MANUAL",
			ConversationID:  "test-conv",
			CurrentMessage: gateway.CurrentMessage{
				UserInputMessage: &gateway.UserInputMessage{
					Content: "hello",
					ModelID: "claude-sonnet-4.5",
					Origin:  "AI_EDITOR",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("SendStreamRequest returned error: %v", err)
	}

	var gotContent bool
	for evt := range eventCh {
		if evt.Content != "" {
			gotContent = true
		}
	}
	for streamErr := range errCh {
		if streamErr != nil {
			t.Fatalf("stream returned error: %v", streamErr)
		}
	}

	if !gotContent {
		t.Fatal("expected at least one content event in stream")
	}
}
