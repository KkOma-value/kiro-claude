package backend

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/yourusername/kiro-claude/internal/config"
	"github.com/yourusername/kiro-claude/internal/gateway"
)

func TestMockClientDefaultWithoutToolsReturnsText(t *testing.T) {
	client := NewMockClient(config.DefaultConfig())

	resp, err := client.SendRequest(context.Background(), &gateway.CodeWhispererRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []gateway.Message{
			{Role: "user", Content: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("SendRequest returned error: %v", err)
	}

	if resp.StopReason != "endTurn" {
		t.Fatalf("unexpected stop reason: %s", resp.StopReason)
	}
	if len(resp.Content) != 1 || resp.Content[0].Type != "text" {
		t.Fatalf("unexpected content: %#v", resp.Content)
	}
	if !strings.Contains(resp.Content[0].Text, "hello") {
		t.Fatalf("expected response to mention last user message, got %q", resp.Content[0].Text)
	}
}

func TestMockClientDefaultStartsToolLoop(t *testing.T) {
	client := NewMockClient(config.DefaultConfig())

	resp, err := client.SendRequest(context.Background(), &gateway.CodeWhispererRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []gateway.Message{
			{Role: "user", Content: "list the repository files"},
		},
		Tools: []map[string]interface{}{
			{
				"name": "list_files",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{"type": "string"},
						"recursive": map[string]interface{}{"type": "boolean"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("SendRequest returned error: %v", err)
	}

	if resp.StopReason != "toolUse" {
		t.Fatalf("unexpected stop reason: %s", resp.StopReason)
	}
	if len(resp.Content) != 1 || resp.Content[0].Type != "tool_use" {
		t.Fatalf("unexpected content: %#v", resp.Content)
	}
	if resp.Content[0].Name != "list_files" {
		t.Fatalf("unexpected tool name: %s", resp.Content[0].Name)
	}

	var input map[string]interface{}
	if err := json.Unmarshal(resp.Content[0].Input, &input); err != nil {
		t.Fatalf("failed to decode tool input: %v", err)
	}
	if input["path"] != "." {
		t.Fatalf("expected path input '.', got %#v", input["path"])
	}
	if input["recursive"] != true {
		t.Fatalf("expected recursive input true, got %#v", input["recursive"])
	}
}

func TestMockClientDefaultConsumesToolResult(t *testing.T) {
	client := NewMockClient(config.DefaultConfig())

	resp, err := client.SendRequest(context.Background(), &gateway.CodeWhispererRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []gateway.Message{
			{Role: "user", Content: "inspect the repo"},
			{
				Role: "assistant",
				Content: []gateway.ContentBlock{
					{Type: "tool_use", ID: "toolu_mock_1", Name: "list_files"},
				},
			},
			{
				Role: "user",
				Content: []gateway.ContentBlock{
					{
						Type:      "tool_result",
						ToolUseID: "toolu_mock_1",
						Content: []interface{}{
							map[string]interface{}{"type": "text", "text": "found README.md"},
						},
					},
				},
			},
		},
		Tools: []map[string]interface{}{
			{"name": "list_files"},
		},
	})
	if err != nil {
		t.Fatalf("SendRequest returned error: %v", err)
	}

	if resp.StopReason != "endTurn" {
		t.Fatalf("unexpected stop reason: %s", resp.StopReason)
	}
	if len(resp.Content) != 1 || resp.Content[0].Type != "text" {
		t.Fatalf("unexpected content: %#v", resp.Content)
	}
	if !strings.Contains(resp.Content[0].Text, "list_files returned: found README.md") {
		t.Fatalf("unexpected summary text: %q", resp.Content[0].Text)
	}
}

func TestMockClientToolChainRequestsSecondTool(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Runtime.MockScenario = "tool-chain"
	client := NewMockClient(cfg)

	resp, err := client.SendRequest(context.Background(), &gateway.CodeWhispererRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []gateway.Message{
			{Role: "user", Content: "inspect the repo"},
			{
				Role: "assistant",
				Content: []gateway.ContentBlock{
					{Type: "tool_use", ID: "toolu_mock_1", Name: "list_files"},
				},
			},
			{
				Role: "user",
				Content: []gateway.ContentBlock{
					{
						Type:      "tool_result",
						ToolUseID: "toolu_mock_1",
						Content: []interface{}{
							map[string]interface{}{"type": "text", "text": "README.md\nmain.go"},
						},
					},
				},
			},
		},
		Tools: []map[string]interface{}{
			{
				"name": "list_files",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{"type": "string"},
					},
				},
			},
			{
				"name": "read_file",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"file_path": map[string]interface{}{"type": "string"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("SendRequest returned error: %v", err)
	}

	if resp.StopReason != "toolUse" {
		t.Fatalf("unexpected stop reason: %s", resp.StopReason)
	}
	if resp.Content[0].Name != "read_file" {
		t.Fatalf("expected second tool to be requested, got %s", resp.Content[0].Name)
	}

	var input map[string]interface{}
	if err := json.Unmarshal(resp.Content[0].Input, &input); err != nil {
		t.Fatalf("failed to decode tool input: %v", err)
	}
	if input["file_path"] != "README.md" {
		t.Fatalf("expected follow-up file path, got %#v", input["file_path"])
	}
}

func TestMockClientToolResultErrorScenarioRetriesTool(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Runtime.MockScenario = "tool-result-error"
	client := NewMockClient(cfg)

	resp, err := client.SendRequest(context.Background(), &gateway.CodeWhispererRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []gateway.Message{
			{Role: "user", Content: "run a safe command"},
			{
				Role: "assistant",
				Content: []gateway.ContentBlock{
					{Type: "tool_use", ID: "toolu_mock_1", Name: "run_command"},
				},
			},
			{
				Role: "user",
				Content: []gateway.ContentBlock{
					{
						Type:      "tool_result",
						ToolUseID: "toolu_mock_1",
						Content: []interface{}{
							map[string]interface{}{"type": "text", "text": "permission denied"},
						},
						Metadata: map[string]interface{}{"isError": true},
					},
				},
			},
		},
		Tools: []map[string]interface{}{
			{
				"name": "run_command",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"command": map[string]interface{}{"type": "string"},
						"input":   map[string]interface{}{"type": "string"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("SendRequest returned error: %v", err)
	}

	if resp.StopReason != "toolUse" {
		t.Fatalf("unexpected stop reason: %s", resp.StopReason)
	}
	if resp.Content[0].Name != "run_command" {
		t.Fatalf("expected retry against same tool, got %s", resp.Content[0].Name)
	}

	var input map[string]interface{}
	if err := json.Unmarshal(resp.Content[0].Input, &input); err != nil {
		t.Fatalf("failed to decode retry input: %v", err)
	}
	if input["command"] != "pwd" {
		t.Fatalf("expected safe retry command, got %#v", input["command"])
	}
	if input["input"] != "permission denied" {
		t.Fatalf("expected retry context to include prior error, got %#v", input["input"])
	}
}

func TestMockClientStreamingToolUseIncludesInputJsonDelta(t *testing.T) {
	client := NewMockClient(config.DefaultConfig())

	chunks, errs, err := client.SendStreamRequest(context.Background(), &gateway.CodeWhispererRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []gateway.Message{
			{Role: "user", Content: "list files"},
		},
		Tools: []map[string]interface{}{
			{
				"name": "list_files",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{"type": "string"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("SendStreamRequest returned error: %v", err)
	}

	var sawStart bool
	var sawInputDelta bool
	for chunk := range chunks {
		if chunk.Type == "contentBlockStart" && chunk.ContentBlock != nil && chunk.ContentBlock.Type == "tool_use" {
			sawStart = true
		}
		if chunk.Type == "contentBlockDelta" && chunk.Delta != nil && chunk.Delta.Type == "inputJsonDelta" && strings.Contains(chunk.Delta.Partial, "\"path\":\".\"") {
			sawInputDelta = true
		}
	}
	for streamErr := range errs {
		if streamErr != nil {
			t.Fatalf("stream returned error: %v", streamErr)
		}
	}

	if !sawStart {
		t.Fatal("expected tool_use content block start in stream")
	}
	if !sawInputDelta {
		t.Fatal("expected inputJsonDelta in stream")
	}
}
