package backend

import (
	"context"
	"fmt"

	"github.com/yourusername/kiro-claude/internal/config"
	"github.com/yourusername/kiro-claude/internal/gateway"
)

// MockClient returns deterministic responses for local development.
type MockClient struct {
	scenario string
}

// NewMockClient creates a mock backend.
func NewMockClient(cfg *config.Config) *MockClient {
	scenario := cfg.Runtime.MockScenario
	if scenario == "" {
		scenario = "default"
	}
	return &MockClient{scenario: scenario}
}

func (c *MockClient) SendRequest(ctx context.Context, req *gateway.KiroRequest) (string, []gateway.KiroStreamEvent, error) {
	select {
	case <-ctx.Done():
		return "", nil, ctx.Err()
	default:
	}

	content := req.ConversationState.CurrentMessage.UserInputMessage.Content
	text := fmt.Sprintf("Mock Kiro backend reply. Your message: %s", content)
	return text, nil, nil
}

func (c *MockClient) SendStreamRequest(ctx context.Context, req *gateway.KiroRequest) (<-chan gateway.KiroStreamEvent, <-chan error, error) {
	eventCh := make(chan gateway.KiroStreamEvent, 8)
	errCh := make(chan error, 1)

	go func() {
		defer close(eventCh)
		defer close(errCh)

		select {
		case <-ctx.Done():
			errCh <- ctx.Err()
			return
		default:
		}

		content := req.ConversationState.CurrentMessage.UserInputMessage.Content
		text := fmt.Sprintf("Mock Kiro backend reply. Your message: %s", content)

		eventCh <- gateway.KiroStreamEvent{Content: text}
	}()

	return eventCh, errCh, nil
}

func (c *MockClient) Models() []gateway.ModelInfo {
	return gateway.SupportedModels()
}

func (c *MockClient) Close() error {
	return nil
}
