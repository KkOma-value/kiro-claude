package backend

import (
	"context"

	"github.com/yourusername/kiro-claude/internal/gateway"
)

// Client defines the backend contract used by the Anthropic-facing API handler.
type Client interface {
	SendRequest(ctx context.Context, req *gateway.CodeWhispererRequest) (*gateway.CodeWhispererResponse, error)
	SendStreamRequest(ctx context.Context, req *gateway.CodeWhispererRequest) (<-chan *gateway.CodeWhispererStreamChunk, <-chan error, error)
	Models() []gateway.ModelInfo
	Close() error
}
