package gateway

import (
	"encoding/json"
	"fmt"
)

// --- Kiro generateAssistantResponse request/response types ---

// KiroRequest is the top-level request to generateAssistantResponse
type KiroRequest struct {
	ConversationState ConversationState `json:"conversationState"`
	ProfileArn        string            `json:"profileArn,omitempty"`
}

type ConversationState struct {
	AgentTaskType   string          `json:"agentTaskType"`
	ChatTriggerType string          `json:"chatTriggerType"`
	ConversationID  string          `json:"conversationId"`
	CurrentMessage  CurrentMessage  `json:"currentMessage"`
	History         []HistoryEntry  `json:"history,omitempty"`
}

type CurrentMessage struct {
	UserInputMessage *UserInputMessage `json:"userInputMessage,omitempty"`
}

type UserInputMessage struct {
	Content                 string                  `json:"content"`
	ModelID                 string                  `json:"modelId"`
	Origin                  string                  `json:"origin"`
	Images                  []KiroImage             `json:"images,omitempty"`
	UserInputMessageContext *UserInputMessageContext `json:"userInputMessageContext,omitempty"`
}

type UserInputMessageContext struct {
	ToolResults []KiroToolResult `json:"toolResults,omitempty"`
	Tools       []KiroTool       `json:"tools,omitempty"`
}

type KiroTool struct {
	ToolSpecification ToolSpecification `json:"toolSpecification"`
}

type ToolSpecification struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	JSON json.RawMessage `json:"json"`
}

type KiroToolResult struct {
	Content   []KiroToolResultContent `json:"content"`
	Status    string                  `json:"status"`
	ToolUseID string                  `json:"toolUseId"`
}

type KiroToolResultContent struct {
	Text string `json:"text"`
}

type KiroImage struct {
	Format string      `json:"format"`
	Source ImageSource `json:"source"`
}

type ImageSource struct {
	Bytes string `json:"bytes"`
}

// HistoryEntry is either a userInputMessage or assistantResponseMessage
type HistoryEntry struct {
	UserInputMessage         *UserInputMessage         `json:"userInputMessage,omitempty"`
	AssistantResponseMessage *AssistantResponseMessage  `json:"assistantResponseMessage,omitempty"`
}

type AssistantResponseMessage struct {
	Content  string         `json:"content"`
	ToolUses []KiroToolUse  `json:"toolUses,omitempty"`
}

type KiroToolUse struct {
	Input     interface{} `json:"input"`
	Name      string      `json:"name"`
	ToolUseID string      `json:"toolUseId"`
}

// --- Kiro SSE response event types ---

// KiroStreamEvent represents a parsed event from the Kiro SSE stream.
// The Kiro API returns events in format: :message-typeevent{json}
type KiroStreamEvent struct {
	Content        string `json:"content,omitempty"`
	FollowupPrompt string `json:"followupPrompt,omitempty"`

	// Tool call fields
	Name      string `json:"name,omitempty"`
	ToolUseID string `json:"toolUseId,omitempty"`
	Input     string `json:"input,omitempty"`
	Stop      bool   `json:"stop,omitempty"`
}

// --- Model mapping ---

// ModelMapping maps public Anthropic model names to Kiro internal model IDs
var ModelMapping = map[string]string{
	"claude-opus-4-7":              "claude-opus-4.7",
	"claude-opus-4-6":              "claude-opus-4.6",
	"claude-opus-4-5":              "claude-opus-4.5",
	"claude-opus-4-5-20251101":     "claude-opus-4.5",
	"claude-sonnet-4-6":            "claude-sonnet-4.6",
	"claude-sonnet-4-5":            "claude-sonnet-4.5",
	"claude-sonnet-4-5-20250514":   "claude-sonnet-4.5",
	"claude-sonnet-4-5-20250929":   "claude-sonnet-4.5",
	"claude-haiku-4-5":             "claude-haiku-4.5",
	"claude-haiku-4-5-20251001":    "claude-haiku-4.5",
	"claude-3-5-sonnet-20241022":   "claude-sonnet-4.5",
	"claude-3-5-haiku-20241022":    "claude-haiku-4.5",
	"claude-3-haiku-20240307":      "claude-haiku-4.5",
	"claude-sonnet-4-20250514":     "claude-sonnet-4.5",
}

// ReverseModelMapping maps Kiro model IDs back to public Anthropic names
var ReverseModelMapping = map[string]string{
	"claude-opus-4.7":   "claude-opus-4-7",
	"claude-opus-4.6":   "claude-opus-4-6",
	"claude-opus-4.5":   "claude-opus-4-5",
	"claude-sonnet-4.6": "claude-sonnet-4-6",
	"claude-sonnet-4.5": "claude-sonnet-4-5",
	"claude-haiku-4.5":  "claude-haiku-4-5",
}

func MapModelToKiro(anthropicModel string) string {
	if kiroModel, ok := ModelMapping[anthropicModel]; ok {
		return kiroModel
	}
	return anthropicModel
}

func MapModelFromKiro(kiroModel string) string {
	if anthropicModel, ok := ReverseModelMapping[kiroModel]; ok {
		return anthropicModel
	}
	return kiroModel
}

// ModelInfo represents model metadata
type ModelInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

func SupportedModels() []ModelInfo {
	return []ModelInfo{
		{ID: "claude-opus-4-7", Name: "Claude Opus 4.7", Provider: "anthropic"},
		{ID: "claude-opus-4-6", Name: "Claude Opus 4.6", Provider: "anthropic"},
		{ID: "claude-opus-4-5", Name: "Claude Opus 4.5", Provider: "anthropic"},
		{ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6", Provider: "anthropic"},
		{ID: "claude-sonnet-4-5", Name: "Claude Sonnet 4.5", Provider: "anthropic"},
		{ID: "claude-sonnet-4-5-20250514", Name: "Claude Sonnet 4.5 (May 2025)", Provider: "anthropic"},
		{ID: "claude-haiku-4-5", Name: "Claude Haiku 4.5", Provider: "anthropic"},
		{ID: "claude-3-5-sonnet-20241022", Name: "Claude 3.5 Sonnet", Provider: "anthropic"},
		{ID: "claude-3-5-haiku-20241022", Name: "Claude 3.5 Haiku", Provider: "anthropic"},
	}
}

// APIError wraps upstream HTTP failures with status information.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("Kiro API error: HTTP %d", e.StatusCode)
	}
	return e.Message
}
