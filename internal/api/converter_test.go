package api

import (
	"encoding/json"
	"testing"

	"github.com/yourusername/kiro-claude/internal/gateway"
)

func TestToCodeWhispererRequestNormalizesFields(t *testing.T) {
	req := &MessageRequest{
		Model:      "claude-3-5-sonnet-20241022",
		MaxTokens:  256,
		System:     "system prompt",
		ToolChoice: "auto",
		Messages: []AnthropicMessage{
			{
				Role: "user",
				Content: []ContentBlock{
					{
						Type:    "tool_result",
						Content: []map[string]string{{"type": "text", "text": "ok"}},
						IsError: true,
					},
				},
			},
		},
	}

	cwReq, err := ToCodeWhispererRequest(req)
	if err != nil {
		t.Fatalf("ToCodeWhispererRequest returned error: %v", err)
	}

	if cwReq.Model != "CLAUDE_3_5_SONNET_20241022_V1_0" {
		t.Fatalf("unexpected model mapping: %s", cwReq.Model)
	}

	if cwReq.SystemPrompt != "system prompt" {
		t.Fatalf("expected system prompt to be preserved, got %q", cwReq.SystemPrompt)
	}

	toolChoice, ok := cwReq.ToolChoice.(map[string]interface{})
	if !ok || toolChoice["type"] != "auto" {
		t.Fatalf("unexpected tool choice: %#v", cwReq.ToolChoice)
	}

	blocks, ok := cwReq.Messages[0].Content.([]gateway.ContentBlock)
	if !ok {
		t.Fatalf("expected converted content blocks, got %T", cwReq.Messages[0].Content)
	}

	if len(blocks) != 1 || !readIsError(blocks[0].Metadata) {
		t.Fatalf("expected tool_result error metadata to be preserved: %#v", blocks)
	}
}

func TestToAnthropicStreamEventNormalizesEventTypes(t *testing.T) {
	toolInput, _ := json.Marshal(map[string]string{"city": "tokyo"})

	event, err := ToAnthropicStreamEvent(&gateway.CodeWhispererStreamChunk{
		Type:              "contentBlockDelta",
		ContentBlockIndex: 2,
		Delta: &gateway.DeltaBlock{
			Type: "textDelta",
			Text: "hello",
		},
		ContentBlock: &gateway.ContentBlock{
			Type:  "tool_use",
			ID:    "tool-1",
			Name:  "weather",
			Input: toolInput,
		},
		StopReason: "toolUse",
		Usage: &gateway.UsageBlock{
			InputTokens:  12,
			OutputTokens: 34,
		},
	})
	if err != nil {
		t.Fatalf("ToAnthropicStreamEvent returned error: %v", err)
	}

	if event.Type != "content_block_delta" {
		t.Fatalf("unexpected event type: %s", event.Type)
	}

	if event.Delta == nil || event.Delta.Type != "text_delta" {
		t.Fatalf("unexpected delta: %#v", event.Delta)
	}

	if event.StopReason != "tool_use" {
		t.Fatalf("unexpected stop reason: %s", event.StopReason)
	}

	if event.ContentBlock == nil || event.ContentBlock.Name != "weather" {
		t.Fatalf("unexpected content block: %#v", event.ContentBlock)
	}
}
