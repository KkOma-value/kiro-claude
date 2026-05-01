package api

import (
	"encoding/json"
	"fmt"

	"github.com/yourusername/kiro-claude/internal/gateway"
	"github.com/yourusername/kiro-claude/internal/model"
)

// package-level resolver for model name resolution
var modelResolver = model.NewResolver()

// Converter handles bidirectional conversion between Anthropic and CodeWhisperer formats

// ToCodeWhispererRequest converts an Anthropic message request to CodeWhisperer format
func ToCodeWhispererRequest(req *MessageRequest) (*gateway.CodeWhispererRequest, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	// Map model name from Anthropic to CodeWhisperer internal ID using resolver
	resolution := modelResolver.Resolve(req.Model)
	cwModel := resolution.InternalID

	// Convert messages
	cwMessages := make([]gateway.Message, len(req.Messages))
	for i, msg := range req.Messages {
		cwMsg := gateway.Message{
			Role: msg.Role,
		}

		// Handle content conversion
		switch content := msg.Content.(type) {
		case string:
			cwMsg.Content = content
		case []interface{}:
			// Convert content blocks
			blocks, err := convertAnthropicContentBlocks(content)
			if err != nil {
				return nil, fmt.Errorf("failed to convert content: %w", err)
			}
			cwMsg.Content = blocks
		case []ContentBlock:
			// Already content blocks, convert to gateway format
			blocks := make([]gateway.ContentBlock, len(content))
			for j, cb := range content {
				blocks[j] = ContentBlockToGateway(cb)
			}
			cwMsg.Content = blocks
		default:
			return nil, fmt.Errorf("unsupported content type: %T", msg.Content)
		}

		cwMessages[i] = cwMsg
	}

	// Convert tools
	var cwTools []map[string]interface{}
	for _, tool := range req.Tools {
		cwTools = append(cwTools, map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		})
	}

	// Convert system prompt
	var cwSystem interface{}
	var cwSystemPrompt string
	if req.System != nil {
		switch system := req.System.(type) {
		case string:
			cwSystemPrompt = system
		default:
			cwSystem = req.System
		}
	}

	cwReq := &gateway.CodeWhispererRequest{
		Model:         cwModel,
		Messages:      cwMessages,
		MaxTokens:     req.MaxTokens,
		Temperature:   req.Temperature,
		TopP:          req.TopP,
		TopK:          req.TopK,
		StopSequences: req.StopSequences,
		Tools:         cwTools,
		ToolChoice:    normalizeToolChoice(req.ToolChoice),
		System:        cwSystem,
		SystemPrompt:  cwSystemPrompt,
	}

	return cwReq, nil
}

// ToAnthropicResponse converts a CodeWhisperer response to Anthropic format
func ToAnthropicResponse(cwResp *gateway.CodeWhispererResponse) (*MessageResponse, error) {
	if cwResp == nil {
		return nil, fmt.Errorf("response is nil")
	}

	// Map model name back from CodeWhisperer to Anthropic using resolver
	anthropicModel := modelResolver.ReverseResolve(cwResp.Model)

	// Convert content blocks
	content := make([]ContentBlock, len(cwResp.Content))
	for i, cwBlock := range cwResp.Content {
		content[i] = ContentBlockFromGateway(cwBlock)
	}

	// Map stop reason
	stopReason := normalizeStopReason(cwResp.StopReason)
	if stopReason == "" {
		stopReason = "end_turn"
	}

	return &MessageResponse{
		ID:           cwResp.ID,
		Type:         "message",
		Role:         "assistant",
		Content:      content,
		Model:        anthropicModel,
		StopReason:   stopReason,
		StopSequence: cwResp.StopSequence,
		Usage: UsageBlock{
			InputTokens:  cwResp.Usage.InputTokens,
			OutputTokens: cwResp.Usage.OutputTokens,
		},
	}, nil
}

// ToAnthropicStreamEvent converts a CodeWhisperer stream chunk to Anthropic format
func ToAnthropicStreamEvent(chunk *gateway.CodeWhispererStreamChunk) (*StreamEvent, error) {
	if chunk == nil {
		return nil, fmt.Errorf("chunk is nil")
	}

	event := &StreamEvent{
		Type:  normalizeStreamEventType(chunk.Type),
		Index: chunk.ContentBlockIndex,
	}

	// Convert content block if present
	if chunk.ContentBlock != nil {
		cb := ContentBlockFromGateway(*chunk.ContentBlock)
		event.ContentBlock = &cb
	}

	// Convert delta if present
	if chunk.Delta != nil {
		event.Delta = &StreamDelta{
			Type:    normalizeDeltaType(chunk.Delta.Type),
			Text:    chunk.Delta.Text,
			Partial: chunk.Delta.Partial,
		}
	}

	// Convert message if present
	if chunk.Message != nil {
		msg, err := ToAnthropicResponse(chunk.Message)
		if err != nil {
			return nil, fmt.Errorf("failed to convert message: %w", err)
		}
		event.Message = msg
	}

	if chunk.StopReason != "" {
		event.StopReason = normalizeStopReason(chunk.StopReason)
	}

	if chunk.Usage != nil {
		event.Usage = &UsageBlock{
			InputTokens:  chunk.Usage.InputTokens,
			OutputTokens: chunk.Usage.OutputTokens,
		}
	}

	return event, nil
}

// --- Helper conversion functions ---

// ContentBlockToGateway converts an Anthropic content block to CodeWhisperer format
func ContentBlockToGateway(cb ContentBlock) gateway.ContentBlock {
	var metadata map[string]interface{}
	if cb.IsError {
		metadata = map[string]interface{}{"isError": true}
	}

	// Note: cb.Content can be interface{}, not assigning to gwBlock.Content which is also interface{}
	return gateway.ContentBlock{
		Type:      cb.Type,
		Text:      cb.Text,
		ID:        cb.ID,
		Name:      cb.Name,
		Input:     cb.Input,
		ToolUseID: cb.ToolUseID,
		Content:   cb.Content,
		Metadata:  metadata,
	}
}

// ContentBlockFromGateway converts a CodeWhisperer content block to Anthropic format
func ContentBlockFromGateway(cwBlock gateway.ContentBlock) ContentBlock {
	return ContentBlock{
		Type:      cwBlock.Type,
		Text:      cwBlock.Text,
		ID:        cwBlock.ID,
		Name:      cwBlock.Name,
		Input:     cwBlock.Input,
		ToolUseID: cwBlock.ToolUseID,
		Content:   cwBlock.Content,
		IsError:   readIsError(cwBlock.Metadata),
	}
}

// convertAnthropicContentBlocks converts interface{} content to ContentBlock slice
func convertAnthropicContentBlocks(content []interface{}) ([]gateway.ContentBlock, error) {
	var blocks []gateway.ContentBlock

	for _, c := range content {
		// Try to marshal and unmarshal to convert between types
		jsonData, err := json.Marshal(c)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal content: %w", err)
		}

		var cb gateway.ContentBlock
		if err := json.Unmarshal(jsonData, &cb); err != nil {
			return nil, fmt.Errorf("failed to unmarshal content block: %w", err)
		}

		blocks = append(blocks, cb)
	}

	return blocks, nil
}

func normalizeToolChoice(choice interface{}) interface{} {
	switch v := choice.(type) {
	case nil:
		return nil
	case string:
		if v == "" {
			return nil
		}
		return map[string]interface{}{"type": v}
	case map[string]interface{}:
		return v
	default:
		return v
	}
}

func normalizeStopReason(reason string) string {
	switch reason {
	case "", "end_turn", "tool_use", "max_tokens", "stop_sequence":
		return reason
	case "endTurn":
		return "end_turn"
	case "toolUse":
		return "tool_use"
	case "maxTokens":
		return "max_tokens"
	case "stopSequence":
		return "stop_sequence"
	default:
		return reason
	}
}

func normalizeStreamEventType(eventType string) string {
	switch eventType {
	case "messageStart":
		return "message_start"
	case "contentBlockStart":
		return "content_block_start"
	case "contentBlockDelta":
		return "content_block_delta"
	case "contentBlockStop":
		return "content_block_stop"
	case "messageDelta":
		return "message_delta"
	case "messageStop":
		return "message_stop"
	default:
		return eventType
	}
}

func normalizeDeltaType(deltaType string) string {
	switch deltaType {
	case "textDelta":
		return "text_delta"
	case "inputJsonDelta":
		return "input_json_delta"
	default:
		return deltaType
	}
}

func readIsError(metadata map[string]interface{}) bool {
	if metadata == nil {
		return false
	}

	value, ok := metadata["isError"]
	if !ok {
		value, ok = metadata["is_error"]
	}
	if !ok {
		return false
	}

	flag, ok := value.(bool)
	return ok && flag
}
