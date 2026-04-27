package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/yourusername/kiro-claude/internal/config"
	"github.com/yourusername/kiro-claude/internal/gateway"
)

// MockClient returns deterministic Anthropic-compatible responses for local development.
type MockClient struct {
	scenario string
}

type mockConversationState struct {
	toolUses    []mockToolUse
	toolResults []mockToolResult
}

type mockToolUse struct {
	ID   string
	Name string
}

type mockToolResult struct {
	ToolUseID string
	Text      string
	IsError   bool
}

// NewMockClient creates a mock backend.
func NewMockClient(cfg *config.Config) *MockClient {
	scenario := cfg.Runtime.MockScenario
	if scenario == "" {
		scenario = "default"
	}
	return &MockClient{scenario: scenario}
}

func (c *MockClient) SendRequest(ctx context.Context, req *gateway.CodeWhispererRequest) (*gateway.CodeWhispererResponse, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return c.responseForRequest(req)
}

func (c *MockClient) SendStreamRequest(ctx context.Context, req *gateway.CodeWhispererRequest) (<-chan *gateway.CodeWhispererStreamChunk, <-chan error, error) {
	chunks := make(chan *gateway.CodeWhispererStreamChunk, 8)
	errs := make(chan error, 1)

	go func() {
		defer close(chunks)
		defer close(errs)

		select {
		case <-ctx.Done():
			errs <- ctx.Err()
			return
		default:
		}

		resp, err := c.responseForRequest(req)
		if err != nil {
			errs <- err
			return
		}

		c.streamResponse(ctx, resp, chunks, errs)
	}()

	return chunks, errs, nil
}

func (c *MockClient) Models() []gateway.ModelInfo {
	return gateway.SupportedModels()
}

func (c *MockClient) Close() error {
	return nil
}

func (c *MockClient) responseForRequest(req *gateway.CodeWhispererRequest) (*gateway.CodeWhispererResponse, error) {
	state := inspectConversation(req.Messages)

	switch c.scenario {
	case "tool-chain":
		return c.toolChainResponse(req, state)
	case "tool-result-error":
		return c.toolRetryResponse(req, state)
	case "default", "tool-use":
		return c.defaultScenarioResponse(req, state)
	default:
		return c.defaultScenarioResponse(req, state)
	}
}

func (c *MockClient) streamResponse(ctx context.Context, resp *gateway.CodeWhispererResponse, chunks chan<- *gateway.CodeWhispererStreamChunk, errs chan<- error) {
	send := func(chunk *gateway.CodeWhispererStreamChunk) bool {
		select {
		case <-ctx.Done():
			errs <- ctx.Err()
			return false
		case chunks <- chunk:
			return true
		}
	}

	if !send(&gateway.CodeWhispererStreamChunk{
		Type: "messageStart",
		Message: &gateway.CodeWhispererResponse{
			ID:    resp.ID,
			Type:  "message",
			Role:  "assistant",
			Model: resp.Model,
			Usage: gateway.UsageBlock{InputTokens: resp.Usage.InputTokens},
		},
	}) {
		return
	}

	for i, block := range resp.Content {
		switch block.Type {
		case "tool_use":
			startBlock := gateway.ContentBlock{
				Type:  block.Type,
				ID:    block.ID,
				Name:  block.Name,
				Input: json.RawMessage(`{}`),
			}
			if !send(&gateway.CodeWhispererStreamChunk{
				Type:              "contentBlockStart",
				ContentBlockIndex: i,
				ContentBlock:      &startBlock,
			}) {
				return
			}
			for _, partial := range splitForStream(string(block.Input)) {
				if partial == "" {
					continue
				}
				if !send(&gateway.CodeWhispererStreamChunk{
					Type:              "contentBlockDelta",
					ContentBlockIndex: i,
					Delta: &gateway.DeltaBlock{
						Type:    "inputJsonDelta",
						Partial: partial,
					},
				}) {
					return
				}
			}
		default:
			if !send(&gateway.CodeWhispererStreamChunk{
				Type:              "contentBlockStart",
				ContentBlockIndex: i,
				ContentBlock: &gateway.ContentBlock{
					Type: "text",
					Text: "",
				},
			}) {
				return
			}
			for _, partial := range splitForStream(block.Text) {
				if partial == "" {
					continue
				}
				if !send(&gateway.CodeWhispererStreamChunk{
					Type:              "contentBlockDelta",
					ContentBlockIndex: i,
					Delta: &gateway.DeltaBlock{
						Type: "textDelta",
						Text: partial,
					},
				}) {
					return
				}
			}
		}

		if !send(&gateway.CodeWhispererStreamChunk{
			Type:              "contentBlockStop",
			ContentBlockIndex: i,
		}) {
			return
		}
	}

	if !send(&gateway.CodeWhispererStreamChunk{
		Type:       "messageDelta",
		StopReason: resp.StopReason,
		Usage:      &resp.Usage,
	}) {
		return
	}
	send(&gateway.CodeWhispererStreamChunk{Type: "messageStop"})
}

func (c *MockClient) defaultScenarioResponse(req *gateway.CodeWhispererRequest, state mockConversationState) (*gateway.CodeWhispererResponse, error) {
	if len(req.Tools) == 0 {
		return c.textResponse(req), nil
	}

	lastResult, hasResult := state.lastToolResult()
	if !hasResult {
		return c.toolUseResponse(req, 0, nil)
	}
	if lastResult.IsError {
		return c.toolFailureResponse(req, state), nil
	}
	return c.toolSummaryResponse(req, state), nil
}

func (c *MockClient) toolChainResponse(req *gateway.CodeWhispererRequest, state mockConversationState) (*gateway.CodeWhispererResponse, error) {
	if len(req.Tools) == 0 {
		return c.textResponse(req), nil
	}

	lastResult, hasResult := state.lastToolResult()
	if !hasResult {
		return c.toolUseResponse(req, 0, nil)
	}
	if lastResult.IsError {
		return c.toolFailureResponse(req, state), nil
	}
	if len(state.toolResults) < len(req.Tools) && len(state.toolResults) < 2 {
		return c.toolUseResponse(req, len(state.toolResults), &lastResult)
	}
	return c.toolSummaryResponse(req, state), nil
}

func (c *MockClient) toolRetryResponse(req *gateway.CodeWhispererRequest, state mockConversationState) (*gateway.CodeWhispererResponse, error) {
	if len(req.Tools) == 0 {
		return c.textResponse(req), nil
	}

	lastResult, hasResult := state.lastToolResult()
	if !hasResult {
		return c.toolUseResponse(req, 0, nil)
	}
	if lastResult.IsError {
		return c.toolUseResponse(req, 0, &lastResult)
	}
	return c.toolSummaryResponse(req, state), nil
}

func (c *MockClient) textResponse(req *gateway.CodeWhispererRequest) *gateway.CodeWhispererResponse {
	text := fmt.Sprintf("Mock Kiro backend reply. Last user message: %s", extractLastUserText(req.Messages))
	return &gateway.CodeWhispererResponse{
		ID:    "mock-message-1",
		Type:  "message",
		Role:  "assistant",
		Model: req.Model,
		Content: []gateway.ContentBlock{
			{
				Type: "text",
				Text: text,
			},
		},
		StopReason: "endTurn",
		Usage: gateway.UsageBlock{
			InputTokens:  32,
			OutputTokens: 24,
		},
	}
}

func (c *MockClient) toolUseResponse(req *gateway.CodeWhispererRequest, toolIndex int, lastResult *mockToolResult) (*gateway.CodeWhispererResponse, error) {
	if len(req.Tools) == 0 {
		return c.textResponse(req), nil
	}

	if toolIndex < 0 {
		toolIndex = 0
	}
	if toolIndex >= len(req.Tools) {
		toolIndex = len(req.Tools) - 1
	}

	tool := req.Tools[toolIndex]
	toolName := readToolName(tool)
	if toolName == "" {
		toolName = "mock_tool"
	}

	input, err := json.Marshal(c.buildToolInput(tool, extractLastUserText(req.Messages), lastResult))
	if err != nil {
		return nil, err
	}

	return &gateway.CodeWhispererResponse{
		ID:    fmt.Sprintf("mock-tool-message-%d", toolIndex+1),
		Type:  "message",
		Role:  "assistant",
		Model: req.Model,
		Content: []gateway.ContentBlock{
			{
				Type:  "tool_use",
				ID:    fmt.Sprintf("toolu_mock_%d", toolIndex+1),
				Name:  toolName,
				Input: input,
			},
		},
		StopReason: "toolUse",
		Usage: gateway.UsageBlock{
			InputTokens:  32,
			OutputTokens: 12,
		},
	}, nil
}

func (c *MockClient) toolSummaryResponse(req *gateway.CodeWhispererRequest, state mockConversationState) *gateway.CodeWhispererResponse {
	lastResult, _ := state.lastToolResult()
	toolCount := len(state.toolResults)
	text := fmt.Sprintf(
		"Mock Kiro backend received %d tool result(s) and can continue the Claude Code loop. Latest tool result (%s): %s",
		toolCount,
		lastResult.ToolUseID,
		lastResult.Text,
	)
	if toolName := state.toolNameFor(lastResult.ToolUseID); toolName != "" {
		text = fmt.Sprintf(
			"Mock Kiro backend received %d tool result(s). %s returned: %s",
			toolCount,
			toolName,
			lastResult.Text,
		)
	}

	return &gateway.CodeWhispererResponse{
		ID:    fmt.Sprintf("mock-summary-message-%d", toolCount),
		Type:  "message",
		Role:  "assistant",
		Model: req.Model,
		Content: []gateway.ContentBlock{
			{
				Type: "text",
				Text: text,
			},
		},
		StopReason: "endTurn",
		Usage: gateway.UsageBlock{
			InputTokens:  48,
			OutputTokens: 36,
		},
	}
}

func (c *MockClient) toolFailureResponse(req *gateway.CodeWhispererRequest, state mockConversationState) *gateway.CodeWhispererResponse {
	lastResult, _ := state.lastToolResult()
	text := fmt.Sprintf(
		"Mock Kiro backend saw a tool_result error for %s. Claude Code should treat this as a failed tool invocation and decide whether to retry. Error output: %s",
		lastResult.ToolUseID,
		lastResult.Text,
	)
	if toolName := state.toolNameFor(lastResult.ToolUseID); toolName != "" {
		text = fmt.Sprintf(
			"Mock Kiro backend saw %s fail. Claude Code can retry or recover. Error output: %s",
			toolName,
			lastResult.Text,
		)
	}

	return &gateway.CodeWhispererResponse{
		ID:    "mock-tool-error-message-1",
		Type:  "message",
		Role:  "assistant",
		Model: req.Model,
		Content: []gateway.ContentBlock{
			{
				Type: "text",
				Text: text,
			},
		},
		StopReason: "endTurn",
		Usage: gateway.UsageBlock{
			InputTokens:  40,
			OutputTokens: 30,
		},
	}
}

func (c *MockClient) buildToolInput(tool map[string]interface{}, query string, lastResult *mockToolResult) map[string]interface{} {
	input := map[string]interface{}{}
	schema, _ := tool["inputSchema"].(map[string]interface{})
	properties, _ := schema["properties"].(map[string]interface{})
	if len(properties) == 0 {
		input["query"] = query
		if lastResult != nil && lastResult.Text != "" {
			input["context"] = lastResult.Text
		}
		return input
	}

	keys := make([]string, 0, len(properties))
	for key := range properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	toolName := strings.ToLower(readToolName(tool))
	for _, key := range keys {
		schemaValue, _ := properties[key].(map[string]interface{})
		input[key] = defaultMockValue(toolName, key, schemaValue, query, lastResult)
	}

	return input
}

func inspectConversation(messages []gateway.Message) mockConversationState {
	state := mockConversationState{}
	for _, msg := range messages {
		for _, block := range contentBlocksFromMessage(msg.Content) {
			switch block.Type {
			case "tool_use":
				state.toolUses = append(state.toolUses, mockToolUse{
					ID:   block.ID,
					Name: block.Name,
				})
			case "tool_result":
				state.toolResults = append(state.toolResults, mockToolResult{
					ToolUseID: block.ToolUseID,
					Text:      extractToolResultText(block),
					IsError:   readMockIsError(block.Metadata),
				})
			}
		}
	}
	return state
}

func (s mockConversationState) lastToolResult() (mockToolResult, bool) {
	if len(s.toolResults) == 0 {
		return mockToolResult{}, false
	}
	return s.toolResults[len(s.toolResults)-1], true
}

func (s mockConversationState) toolNameFor(toolUseID string) string {
	for i := len(s.toolUses) - 1; i >= 0; i-- {
		if s.toolUses[i].ID == toolUseID {
			return s.toolUses[i].Name
		}
	}
	return ""
}

func contentBlocksFromMessage(content interface{}) []gateway.ContentBlock {
	switch v := content.(type) {
	case []gateway.ContentBlock:
		return v
	case []interface{}:
		blocks := make([]gateway.ContentBlock, 0, len(v))
		for _, raw := range v {
			data, err := json.Marshal(raw)
			if err != nil {
				continue
			}
			var block gateway.ContentBlock
			if err := json.Unmarshal(data, &block); err != nil {
				continue
			}
			blocks = append(blocks, block)
		}
		return blocks
	default:
		return nil
	}
}

func extractToolResultText(block gateway.ContentBlock) string {
	if block.Text != "" {
		return block.Text
	}

	switch content := block.Content.(type) {
	case string:
		return content
	case []gateway.ContentBlock:
		var parts []string
		for _, nested := range content {
			if nested.Text != "" {
				parts = append(parts, nested.Text)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, " ")
		}
	case []interface{}:
		var parts []string
		for _, raw := range content {
			if blockMap, ok := raw.(map[string]interface{}); ok {
				if text, _ := blockMap["text"].(string); text != "" {
					parts = append(parts, text)
				}
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, " ")
		}
	}

	data, err := json.Marshal(block.Content)
	if err != nil || string(data) == "null" {
		return "mock tool result"
	}
	return string(data)
}

func readMockIsError(metadata map[string]interface{}) bool {
	value, ok := metadata["isError"]
	if !ok {
		value, ok = metadata["is_error"]
	}
	flag, ok := value.(bool)
	return ok && flag
}

func readToolName(tool map[string]interface{}) string {
	name, _ := tool["name"].(string)
	return name
}

func defaultMockValue(toolName, propertyName string, schema map[string]interface{}, query string, lastResult *mockToolResult) interface{} {
	lowerProperty := strings.ToLower(propertyName)

	if enumValues, ok := schema["enum"].([]interface{}); ok && len(enumValues) > 0 {
		return enumValues[0]
	}

	switch {
	case strings.Contains(lowerProperty, "path"):
		if strings.Contains(lowerProperty, "file") {
			return "README.md"
		}
		return "."
	case strings.Contains(lowerProperty, "command"):
		return "pwd"
	case strings.Contains(lowerProperty, "pattern"):
		return "TODO"
	case strings.Contains(lowerProperty, "query"), strings.Contains(lowerProperty, "search"):
		if query != "" {
			return query
		}
		return "mock query"
	case strings.Contains(lowerProperty, "content"), strings.Contains(lowerProperty, "text"), strings.Contains(lowerProperty, "prompt"), strings.Contains(lowerProperty, "input"):
		if lastResult != nil && lastResult.Text != "" {
			return lastResult.Text
		}
		if query != "" {
			return query
		}
		return fmt.Sprintf("mock input for %s", toolName)
	}

	typeName, _ := schema["type"].(string)
	switch typeName {
	case "integer", "number":
		return 1
	case "boolean":
		return true
	case "array":
		return []interface{}{}
	case "object":
		return map[string]interface{}{}
	default:
		if query != "" {
			return query
		}
		return fmt.Sprintf("mock_%s", propertyName)
	}
}

func splitForStream(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if len(value) <= 24 {
		return []string{value}
	}

	mid := len(value) / 2
	return []string{value[:mid], value[mid:]}
}

func extractLastUserText(messages []gateway.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != "user" {
			continue
		}

		switch content := messages[i].Content.(type) {
		case string:
			if content != "" {
				return content
			}
		case []gateway.ContentBlock:
			var parts []string
			for _, block := range content {
				if block.Text != "" {
					parts = append(parts, block.Text)
				}
			}
			if len(parts) > 0 {
				return strings.Join(parts, " ")
			}
		case []interface{}:
			var parts []string
			for _, raw := range content {
				if block, ok := raw.(map[string]interface{}); ok {
					text, _ := block["text"].(string)
					if text != "" {
						parts = append(parts, text)
					}
				}
			}
			if len(parts) > 0 {
				return strings.Join(parts, " ")
			}
		}
	}

	return "hello from Claude Code"
}
