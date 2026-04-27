package gateway

import (
	"encoding/json"
)

// CodeWhisperer API Request/Response structures
// These represent the internal CodeWhisperer format

// Message represents a message in CodeWhisperer format
type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // Can be string or ContentBlock[]
}

// ContentBlock represents a block of content in CodeWhisperer response
type ContentBlock struct {
	Type      string                 `json:"type"` // "text", "tool_use", etc
	Text      string                 `json:"text,omitempty"`
	ID        string                 `json:"id,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Input     json.RawMessage        `json:"input,omitempty"`
	ToolUseID string                 `json:"tool_use_id,omitempty"`
	Content   interface{}            `json:"content,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// CodeWhispererRequest represents a request to CodeWhisperer API
type CodeWhispererRequest struct {
	Model            string                   `json:"model"`
	Messages         []Message                `json:"messages"`
	MaxTokens        int                      `json:"maxTokens,omitempty"`
	Temperature      float32                  `json:"temperature,omitempty"`
	TopP             float32                  `json:"topP,omitempty"`
	TopK             int                      `json:"topK,omitempty"`
	StopSequences    []string                 `json:"stopSequences,omitempty"`
	Tools            []map[string]interface{} `json:"tools,omitempty"`
	ToolChoice       interface{}              `json:"toolChoice,omitempty"`
	System           interface{}              `json:"system,omitempty"` // Can be string or SystemBlock[]
	SystemPrompt     string                   `json:"systemPrompt,omitempty"`
	AdditionalFields map[string]interface{}   `json:"-"` // For any extra fields
}

// CodeWhispererResponse represents a response from CodeWhisperer API
type CodeWhispererResponse struct {
	ID                string         `json:"id"`
	Type              string         `json:"type"` // "message"
	Role              string         `json:"role"` // "assistant"
	Content           []ContentBlock `json:"content"`
	Model             string         `json:"model"`
	StopReason        string         `json:"stopReason"`
	StopSequence      string         `json:"stopSequence,omitempty"`
	Usage             UsageBlock     `json:"usage"`
	Error             *ErrorBlock    `json:"error,omitempty"`
	CitationMetadata  interface{}    `json:"citationMetadata,omitempty"`
	ToolResults       interface{}    `json:"toolResults,omitempty"`
	ContinuationToken string         `json:"continuationToken,omitempty"`
}

// UsageBlock represents token usage information
type UsageBlock struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
}

// ErrorBlock represents an error response
type ErrorBlock struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}

// StreamingMessage represents a chunk from streaming CodeWhisperer response
type StreamingMessage struct {
	Type         string                 `json:"type"` // "message", "contentBlockStart", "contentBlockDelta", "contentBlockStop", "messageDelta", "messageStop"
	Index        int                    `json:"index,omitempty"`
	ContentBlock *ContentBlock          `json:"contentBlock,omitempty"`
	Delta        *DeltaBlock            `json:"delta,omitempty"`
	Message      *CodeWhispererResponse `json:"message,omitempty"`
	StopReason   string                 `json:"stopReason,omitempty"`
	Usage        *UsageBlock            `json:"usage,omitempty"`
}

// DeltaBlock represents incremental changes in streaming response
type DeltaBlock struct {
	Type    string `json:"type"` // "textDelta", "inputJsonDelta"
	Text    string `json:"text,omitempty"`
	Partial string `json:"partial,omitempty"` // For JSON parsing
}

// ModelInfo represents model metadata
type ModelInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

// SupportedModels returns the public Anthropic model ids available through the proxy.
func SupportedModels() []ModelInfo {
	return []ModelInfo{
		{ID: "claude-opus-4.7", Name: "Claude Opus 4.7", Provider: "anthropic"},
		{ID: "claude-sonnet-4.5-20250514", Name: "Claude Sonnet 4.5", Provider: "anthropic"},
		{ID: "claude-3-5-haiku-20241022", Name: "Claude 3.5 Haiku", Provider: "anthropic"},
		{ID: "claude-sonnet-4-20240229", Name: "Claude Sonnet 4", Provider: "anthropic"},
		{ID: "claude-3-5-sonnet-20241022", Name: "Claude 3.5 Sonnet", Provider: "anthropic"},
		{ID: "claude-3-haiku-20240307", Name: "Claude 3 Haiku", Provider: "anthropic"},
	}
}

// CodeWhispererStreamChunk represents a single line from SSE stream
// After decoding from "data: {json}"
type CodeWhispererStreamChunk struct {
	Type              string                 `json:"type"`
	ContentBlock      *ContentBlock          `json:"contentBlock,omitempty"`
	ContentBlockIndex int                    `json:"contentBlockIndex,omitempty"`
	Delta             *DeltaBlock            `json:"delta,omitempty"`
	Message           *CodeWhispererResponse `json:"message,omitempty"`
	StopReason        string                 `json:"stopReason,omitempty"`
	Usage             *UsageBlock            `json:"usage,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// ModelNameMapping maps public Anthropic model names to CodeWhisperer internal IDs
var ModelNameMapping = map[string]string{
	"claude-opus-4.7":            "CLAUDE_OPUS_4_7_20250219_V1_0",
	"claude-opus-4-20250805":     "CLAUDE_OPUS_4_20250805_V1_0",
	"claude-sonnet-4.5-20250514": "CLAUDE_SONNET_4_5_20250514_V1_0",
	"claude-sonnet-4-20250514":   "CLAUDE_SONNET_4_20250514_V1_0",
	"claude-sonnet-4-20250102":   "CLAUDE_SONNET_4_20250102_V1_0",
	"claude-sonnet-4-20240229":   "CLAUDE_SONNET_4_20240229_V1_0",
	"claude-3-5-sonnet-20241022": "CLAUDE_3_5_SONNET_20241022_V1_0",
	"claude-3-5-sonnet-20240620": "CLAUDE_3_5_SONNET_20240620_V1_0",
	"claude-3-sonnet-20240229":   "CLAUDE_3_SONNET_20240229_V1_0",
	"claude-3-5-haiku-20241022":  "CLAUDE_3_5_HAIKU_20241022_V1_0",
	"claude-3-haiku-20240307":    "CLAUDE_3_HAIKU_20240307_V1_0",
	"claude-instant-1-2":         "CLAUDE_INSTANT_1_2_V1_0",
	// Add more as needed; defaults to model name if not found
}

// ReverseModelMapping maps CodeWhisperer IDs back to public names (for response conversion)
var ReverseModelMapping = map[string]string{
	"CLAUDE_OPUS_4_7_20250219_V1_0":   "claude-opus-4.7",
	"CLAUDE_OPUS_4_20250805_V1_0":     "claude-opus-4-20250805",
	"CLAUDE_SONNET_4_5_20250514_V1_0": "claude-sonnet-4.5-20250514",
	"CLAUDE_SONNET_4_20250514_V1_0":   "claude-sonnet-4-20250514",
	"CLAUDE_SONNET_4_20250102_V1_0":   "claude-sonnet-4-20250102",
	"CLAUDE_SONNET_4_20240229_V1_0":   "claude-sonnet-4-20240229",
	"CLAUDE_3_5_SONNET_20241022_V1_0": "claude-3-5-sonnet-20241022",
	"CLAUDE_3_5_SONNET_20240620_V1_0": "claude-3-5-sonnet-20240620",
	"CLAUDE_3_SONNET_20240229_V1_0":   "claude-3-sonnet-20240229",
	"CLAUDE_3_5_HAIKU_20241022_V1_0":  "claude-3-5-haiku-20241022",
	"CLAUDE_3_HAIKU_20240307_V1_0":    "claude-3-haiku-20240307",
	"CLAUDE_INSTANT_1_2_V1_0":         "claude-instant-1-2",
}
