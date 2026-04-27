package api

import (
	"encoding/json"
)

// Anthropic API types (Claude Code native format)
// Reference: https://docs.anthropic.com/en/api/messages

// MessageRequest represents a message creation request (Anthropic format)
type MessageRequest struct {
	Model         string                 `json:"model"`
	Messages      []AnthropicMessage     `json:"messages"`
	MaxTokens     int                    `json:"max_tokens"`
	Temperature   float32                `json:"temperature,omitempty"`
	TopP          float32                `json:"top_p,omitempty"`
	TopK          int                    `json:"top_k,omitempty"`
	StopSequences []string               `json:"stop_sequences,omitempty"`
	Tools         []ToolDefinition       `json:"tools,omitempty"`
	ToolChoice    interface{}            `json:"tool_choice,omitempty"`
	System        interface{}            `json:"system,omitempty"` // Can be string or SystemBlock[]
	Stream        bool                   `json:"stream,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// AnthropicMessage represents a message in Anthropic format
type AnthropicMessage struct {
	Role    string      `json:"role"`    // "user" or "assistant"
	Content interface{} `json:"content"` // Can be string or ContentBlock[]
}

// ContentBlock represents a content block in Anthropic format
type ContentBlock struct {
	Type      string          `json:"type"` // "text", "tool_use", "tool_result", "image"
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   interface{}     `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

// ToolDefinition defines a tool available to the model
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// MessageResponse represents a successful message response (Anthropic format)
type MessageResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"` // "message"
	Role         string         `json:"role"` // "assistant"
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason"` // "end_turn", "tool_use", "max_tokens", "stop_sequence"
	StopSequence string         `json:"stop_sequence,omitempty"`
	Usage        UsageBlock     `json:"usage"`
	Error        *ErrorResponse `json:"error,omitempty"`
}

// UsageBlock represents token usage
type UsageBlock struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ErrorResponse represents an error from the API
type ErrorResponse struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// StreamEvent represents a streaming event (Anthropic format)
type StreamEvent struct {
	Type         string           `json:"type"` // "message_start", "content_block_start", "content_block_delta", "content_block_stop", "message_delta", "message_stop"
	Index        int              `json:"index,omitempty"`
	ContentBlock *ContentBlock    `json:"content_block,omitempty"`
	Delta        *StreamDelta     `json:"delta,omitempty"`
	Message      *MessageResponse `json:"message,omitempty"`
	StopReason   string           `json:"stop_reason,omitempty"`
	Usage        *UsageBlock      `json:"usage,omitempty"`
}

// StreamDelta represents incremental changes in a stream
type StreamDelta struct {
	Type    string `json:"type"` // "text_delta", "input_json_delta"
	Text    string `json:"text,omitempty"`
	Partial string `json:"partial_json,omitempty"`
}

// ModelsResponse represents the response from the models endpoint
type ModelsResponse struct {
	Data   []ModelData `json:"data"`
	Object string      `json:"object"`
}

// ModelData represents a single model in the models list
type ModelData struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}
