package api

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/yourusername/kiro-claude/internal/gateway"
)

// ToKiroRequest converts an Anthropic Messages API request to a Kiro generateAssistantResponse request
func ToKiroRequest(req *MessageRequest) (*gateway.KiroRequest, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	kiroModel := gateway.MapModelToKiro(req.Model)
	conversationID := uuid.New().String()

	// Extract system prompt
	systemPrompt := extractSystemPrompt(req.System)

	// Convert tools to Kiro format
	var kiroTools []gateway.KiroTool
	for _, tool := range req.Tools {
		inputSchema, _ := json.Marshal(tool.InputSchema)
		if tool.Description == "" {
			continue
		}
		desc := tool.Description
		if len(desc) > 9216 {
			desc = desc[:9216] + "..."
		}
		kiroTools = append(kiroTools, gateway.KiroTool{
			ToolSpecification: gateway.ToolSpecification{
				Name:        tool.Name,
				Description: desc,
				InputSchema: gateway.InputSchema{JSON: inputSchema},
			},
		})
	}

	// If no tools, add a placeholder (Kiro API requires at least one)
	if len(kiroTools) == 0 {
		kiroTools = []gateway.KiroTool{{
			ToolSpecification: gateway.ToolSpecification{
				Name:        "no_tool_available",
				Description: "Placeholder tool when no other tools are available.",
				InputSchema: gateway.InputSchema{JSON: json.RawMessage(`{"type":"object","properties":{}}`)},
			},
		}}
	}

	// Build history and current message from Anthropic messages
	history, currentContent, currentToolResults, err := buildHistoryAndCurrent(req.Messages, kiroModel, systemPrompt)
	if err != nil {
		return nil, err
	}

	if currentContent == "" {
		if len(currentToolResults) > 0 {
			currentContent = "Tool results provided."
		} else {
			currentContent = "Continue"
		}
	}

	// Build current message
	userInput := &gateway.UserInputMessage{
		Content: currentContent,
		ModelID: kiroModel,
		Origin:  "AI_EDITOR",
	}

	ctx := &gateway.UserInputMessageContext{}
	hasCtx := false
	if len(currentToolResults) > 0 {
		ctx.ToolResults = currentToolResults
		hasCtx = true
	}
	if len(kiroTools) > 0 {
		ctx.Tools = kiroTools
		hasCtx = true
	}
	if hasCtx {
		userInput.UserInputMessageContext = ctx
	}

	kiroReq := &gateway.KiroRequest{
		ConversationState: gateway.ConversationState{
			AgentTaskType:   "vibe",
			ChatTriggerType: "MANUAL",
			ConversationID:  conversationID,
			CurrentMessage: gateway.CurrentMessage{
				UserInputMessage: userInput,
			},
		},
	}

	if len(history) > 0 {
		kiroReq.ConversationState.History = history
	}

	return kiroReq, nil
}

// buildHistoryAndCurrent converts Anthropic messages into Kiro history entries + current message parts
func buildHistoryAndCurrent(messages []AnthropicMessage, kiroModel, systemPrompt string) ([]gateway.HistoryEntry, string, []gateway.KiroToolResult, error) {
	var history []gateway.HistoryEntry

	if len(messages) == 0 {
		return nil, "", nil, fmt.Errorf("no messages provided")
	}

	// Merge adjacent same-role messages
	merged := mergeAdjacentMessages(messages)

	startIndex := 0

	// Prepend system prompt to first user message or as standalone
	if systemPrompt != "" {
		if len(merged) > 0 && merged[0].Role == "user" {
			firstContent := extractTextFromContent(merged[0].Content)
			history = append(history, gateway.HistoryEntry{
				UserInputMessage: &gateway.UserInputMessage{
					Content: systemPrompt + "\n\n" + firstContent,
					ModelID: kiroModel,
					Origin:  "AI_EDITOR",
				},
			})
			startIndex = 1
		} else {
			history = append(history, gateway.HistoryEntry{
				UserInputMessage: &gateway.UserInputMessage{
					Content: systemPrompt,
					ModelID: kiroModel,
					Origin:  "AI_EDITOR",
				},
			})
		}
	}

	// Process all messages except the last one into history
	for i := startIndex; i < len(merged)-1; i++ {
		msg := merged[i]
		if msg.Role == "user" {
			entry := convertUserToHistory(msg, kiroModel)
			history = append(history, entry)
		} else if msg.Role == "assistant" {
			entry := convertAssistantToHistory(msg)
			history = append(history, entry)
		}
	}

	// Ensure history ends with assistantResponseMessage if needed
	if len(history) > 0 {
		last := history[len(history)-1]
		if last.AssistantResponseMessage == nil && last.UserInputMessage != nil {
			history = append(history, gateway.HistoryEntry{
				AssistantResponseMessage: &gateway.AssistantResponseMessage{
					Content: "Continue",
				},
			})
		}
	}

	// Process last message as current
	lastMsg := merged[len(merged)-1]

	if lastMsg.Role == "assistant" {
		// Move assistant to history, create "Continue" as current
		entry := convertAssistantToHistory(lastMsg)
		history = append(history, entry)
		return history, "Continue", nil, nil
	}

	// Last message is user — extract content and tool results
	currentContent, toolResults := extractUserParts(lastMsg)
	return history, currentContent, toolResults, nil
}

func convertUserToHistory(msg AnthropicMessage, kiroModel string) gateway.HistoryEntry {
	content, toolResults := extractUserParts(msg)
	if content == "" {
		if len(toolResults) > 0 {
			content = "Tool results provided."
		} else {
			content = "Continue"
		}
	}

	entry := gateway.HistoryEntry{
		UserInputMessage: &gateway.UserInputMessage{
			Content: content,
			ModelID: kiroModel,
			Origin:  "AI_EDITOR",
		},
	}

	if len(toolResults) > 0 {
		// Deduplicate by toolUseId
		seen := map[string]bool{}
		var unique []gateway.KiroToolResult
		for _, tr := range toolResults {
			if !seen[tr.ToolUseID] {
				seen[tr.ToolUseID] = true
				unique = append(unique, tr)
			}
		}
		entry.UserInputMessage.UserInputMessageContext = &gateway.UserInputMessageContext{
			ToolResults: unique,
		}
	}

	return entry
}

func convertAssistantToHistory(msg AnthropicMessage) gateway.HistoryEntry {
	var textParts []string
	var toolUses []gateway.KiroToolUse
	var thinkingText string

	blocks := contentToBlocks(msg.Content)
	for _, block := range blocks {
		switch block.Type {
		case "text":
			textParts = append(textParts, block.Text)
		case "thinking":
			if block.Text != "" {
				thinkingText += block.Text
			}
		case "tool_use":
			var input interface{}
			if len(block.Input) > 0 {
				_ = json.Unmarshal(block.Input, &input)
			}
			toolUses = append(toolUses, gateway.KiroToolUse{
				Input:     input,
				Name:      block.Name,
				ToolUseID: block.ID,
			})
		}
	}

	content := strings.Join(textParts, "")
	if thinkingText != "" {
		if content != "" {
			content = "<thinking>" + thinkingText + "</thinking>\n\n" + content
		} else {
			content = "<thinking>" + thinkingText + "</thinking>"
		}
	}
	if content == "" {
		content = "Continue"
	}

	entry := gateway.HistoryEntry{
		AssistantResponseMessage: &gateway.AssistantResponseMessage{
			Content: content,
		},
	}
	if len(toolUses) > 0 {
		entry.AssistantResponseMessage.ToolUses = toolUses
	}

	return entry
}

func extractUserParts(msg AnthropicMessage) (string, []gateway.KiroToolResult) {
	var textParts []string
	var toolResults []gateway.KiroToolResult

	blocks := contentToBlocks(msg.Content)
	for _, block := range blocks {
		switch block.Type {
		case "text":
			textParts = append(textParts, block.Text)
		case "tool_result":
			resultText := extractToolResultText(block)
			toolResults = append(toolResults, gateway.KiroToolResult{
				Content:   []gateway.KiroToolResultContent{{Text: resultText}},
				Status:    "success",
				ToolUseID: block.ToolUseID,
			})
		}
	}

	return strings.Join(textParts, ""), toolResults
}

func extractToolResultText(block ContentBlock) string {
	if block.Text != "" {
		return block.Text
	}
	// Content can be string or []ContentBlock
	switch c := block.Content.(type) {
	case string:
		return c
	case []interface{}:
		var parts []string
		for _, item := range c {
			if m, ok := item.(map[string]interface{}); ok {
				if t, ok := m["text"].(string); ok {
					parts = append(parts, t)
				}
			}
		}
		return strings.Join(parts, " ")
	}
	if block.Content != nil {
		data, _ := json.Marshal(block.Content)
		return string(data)
	}
	return "tool result"
}

// --- Response conversion: Kiro stream events → Anthropic SSE ---

// StreamState tracks state across streaming events for building Anthropic-format SSE
type StreamState struct {
	Model           string
	ContentIndex    int
	TextBuffer      strings.Builder
	CurrentToolCall *ToolCallState
	ToolCalls       []ToolCallState
	InputTokens     int
	OutputTokens    int
}

type ToolCallState struct {
	ID        string
	Name      string
	InputJSON strings.Builder
}

// ProcessKiroEvent converts a single Kiro stream event into zero or more Anthropic SSE events
func (s *StreamState) ProcessKiroEvent(evt gateway.KiroStreamEvent) []StreamEvent {
	var events []StreamEvent

	if evt.Name != "" && evt.ToolUseID != "" {
		// Tool call event
		if s.CurrentToolCall == nil || s.CurrentToolCall.ID != evt.ToolUseID {
			// Flush any pending text
			events = append(events, s.flushText()...)

			// Start new tool call
			s.CurrentToolCall = &ToolCallState{
				ID:   evt.ToolUseID,
				Name: evt.Name,
			}

			// content_block_start for tool_use
			events = append(events, StreamEvent{
				Type:  "content_block_start",
				Index: s.ContentIndex,
				ContentBlock: &ContentBlock{
					Type:  "tool_use",
					ID:    evt.ToolUseID,
					Name:  evt.Name,
					Input: json.RawMessage(`{}`),
				},
			})
		}

		if evt.Input != "" {
			s.CurrentToolCall.InputJSON.WriteString(evt.Input)
			events = append(events, StreamEvent{
				Type:  "content_block_delta",
				Index: s.ContentIndex,
				Delta: &StreamDelta{
					Type:    "input_json_delta",
					Partial: evt.Input,
				},
			})
		}

		if evt.Stop {
			events = append(events, StreamEvent{
				Type:  "content_block_stop",
				Index: s.ContentIndex,
			})
			s.ContentIndex++
			s.ToolCalls = append(s.ToolCalls, *s.CurrentToolCall)
			s.CurrentToolCall = nil
		}
	} else if evt.Content != "" {
		// Text content event
		content := strings.ReplaceAll(evt.Content, `\n`, "\n")

		if s.TextBuffer.Len() == 0 {
			// First text chunk — emit content_block_start
			events = append(events, StreamEvent{
				Type:  "content_block_start",
				Index: s.ContentIndex,
				ContentBlock: &ContentBlock{
					Type: "text",
					Text: "",
				},
			})
		}

		s.TextBuffer.WriteString(content)
		events = append(events, StreamEvent{
			Type:  "content_block_delta",
			Index: s.ContentIndex,
			Delta: &StreamDelta{
				Type: "text_delta",
				Text: content,
			},
		})
	}

	return events
}

func (s *StreamState) flushText() []StreamEvent {
	if s.TextBuffer.Len() == 0 {
		return nil
	}
	events := []StreamEvent{{
		Type:  "content_block_stop",
		Index: s.ContentIndex,
	}}
	s.ContentIndex++
	s.TextBuffer.Reset()
	return events
}

// Finalize returns the closing SSE events
func (s *StreamState) Finalize() []StreamEvent {
	var events []StreamEvent

	// Flush pending text
	events = append(events, s.flushText()...)

	// Determine stop reason
	stopReason := "end_turn"
	if len(s.ToolCalls) > 0 {
		stopReason = "tool_use"
	}

	events = append(events, StreamEvent{
		Type:       "message_delta",
		StopReason: stopReason,
		Usage: &UsageBlock{
			InputTokens:  s.InputTokens,
			OutputTokens: s.OutputTokens,
		},
	})

	events = append(events, StreamEvent{
		Type: "message_stop",
	})

	return events
}

// BuildMessageStart returns the initial message_start event
func (s *StreamState) BuildMessageStart(requestModel string) StreamEvent {
	return StreamEvent{
		Type: "message_start",
		Message: &MessageResponse{
			ID:    "msg_" + uuid.New().String()[:8],
			Type:  "message",
			Role:  "assistant",
			Model: requestModel,
			Usage: UsageBlock{InputTokens: 0, OutputTokens: 0},
		},
	}
}

// --- Helper functions ---

func extractSystemPrompt(system interface{}) string {
	if system == nil {
		return ""
	}
	switch s := system.(type) {
	case string:
		return s
	case []interface{}:
		var parts []string
		for _, item := range s {
			if m, ok := item.(map[string]interface{}); ok {
				if t, ok := m["text"].(string); ok {
					parts = append(parts, t)
				}
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

func extractTextFromContent(content interface{}) string {
	switch c := content.(type) {
	case string:
		return c
	case []interface{}:
		var parts []string
		for _, item := range c {
			if m, ok := item.(map[string]interface{}); ok {
				if t, ok := m["text"].(string); ok {
					parts = append(parts, t)
				}
			}
		}
		return strings.Join(parts, "")
	}
	return ""
}

// contentToBlocks normalizes message content (string or []interface{}) into ContentBlock slice
func contentToBlocks(content interface{}) []ContentBlock {
	switch c := content.(type) {
	case string:
		return []ContentBlock{{Type: "text", Text: c}}
	case []interface{}:
		var blocks []ContentBlock
		for _, item := range c {
			data, err := json.Marshal(item)
			if err != nil {
				continue
			}
			var block ContentBlock
			if err := json.Unmarshal(data, &block); err != nil {
				continue
			}
			blocks = append(blocks, block)
		}
		return blocks
	case []ContentBlock:
		return c
	}
	return nil
}

func mergeAdjacentMessages(messages []AnthropicMessage) []AnthropicMessage {
	if len(messages) == 0 {
		return messages
	}

	var merged []AnthropicMessage
	for _, msg := range messages {
		if len(merged) == 0 || merged[len(merged)-1].Role != msg.Role {
			merged = append(merged, msg)
			continue
		}

		// Same role — merge content
		last := &merged[len(merged)-1]
		lastBlocks := contentToBlocks(last.Content)
		newBlocks := contentToBlocks(msg.Content)
		allBlocks := append(lastBlocks, newBlocks...)

		// Convert back to []interface{} for JSON compatibility
		var result []interface{}
		for _, b := range allBlocks {
			data, _ := json.Marshal(b)
			var m interface{}
			_ = json.Unmarshal(data, &m)
			result = append(result, m)
		}
		last.Content = result
	}

	return merged
}
