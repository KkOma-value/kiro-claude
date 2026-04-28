package backend

import (
	"context"

	"github.com/yourusername/kiro-claude/internal/gateway"
)

// Client defines the backend contract used by the Anthropic-facing API handler.
type Client interface {
	// SendRequest sends a non-streaming request and returns the full response text + tool call events.
	SendRequest(ctx context.Context, req *gateway.KiroRequest) (string, []gateway.KiroStreamEvent, error)
	// SendStreamRequest sends a streaming request and returns channels for events and errors.
	SendStreamRequest(ctx context.Context, req *gateway.KiroRequest) (<-chan gateway.KiroStreamEvent, <-chan error, error)
	Models() []gateway.ModelInfo
	Close() error
}
