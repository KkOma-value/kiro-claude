package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yourusername/kiro-claude/internal/gateway"
	"github.com/yourusername/kiro-claude/internal/logger"
)

type stubGatewayClient struct {
	sendRequest       func(ctx context.Context, req *gateway.CodeWhispererRequest) (*gateway.CodeWhispererResponse, error)
	sendStreamRequest func(ctx context.Context, req *gateway.CodeWhispererRequest) (<-chan *gateway.CodeWhispererStreamChunk, <-chan error, error)
	models            func() []gateway.ModelInfo
}

func (s *stubGatewayClient) SendRequest(ctx context.Context, req *gateway.CodeWhispererRequest) (*gateway.CodeWhispererResponse, error) {
	return s.sendRequest(ctx, req)
}

func (s *stubGatewayClient) SendStreamRequest(ctx context.Context, req *gateway.CodeWhispererRequest) (<-chan *gateway.CodeWhispererStreamChunk, <-chan error, error) {
	return s.sendStreamRequest(ctx, req)
}

func (s *stubGatewayClient) Models() []gateway.ModelInfo {
	if s.models != nil {
		return s.models()
	}
	return gateway.SupportedModels()
}

type flushRecorder struct {
	*httptest.ResponseRecorder
}

func (r *flushRecorder) Flush() {}

func TestHandleMessagesReturnsAnthropicErrorEnvelope(t *testing.T) {
	handler := NewHandler(&stubGatewayClient{
		sendRequest: func(ctx context.Context, req *gateway.CodeWhispererRequest) (*gateway.CodeWhispererResponse, error) {
			return nil, &gateway.APIError{StatusCode: http.StatusUnauthorized, Message: "bad token"}
		},
		sendStreamRequest: func(ctx context.Context, req *gateway.CodeWhispererRequest) (<-chan *gateway.CodeWhispererStreamChunk, <-chan error, error) {
			return nil, nil, errors.New("unexpected")
		},
	}, logger.NewSimpleLogger(logger.LevelError))

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{
		"model":"claude-3-5-sonnet-20241022",
		"messages":[{"role":"user","content":"hello"}],
		"max_tokens":32
	}`))
	rec := httptest.NewRecorder()

	handler.HandleMessages(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status code: %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"type":"error"`) || !strings.Contains(body, `"invalid_request_error"`) {
		t.Fatalf("unexpected error body: %s", body)
	}
}

func TestHandleMessagesStreamingWritesAnthropicSSE(t *testing.T) {
	handler := NewHandler(&stubGatewayClient{
		sendRequest: func(ctx context.Context, req *gateway.CodeWhispererRequest) (*gateway.CodeWhispererResponse, error) {
			return nil, errors.New("unexpected")
		},
		sendStreamRequest: func(ctx context.Context, req *gateway.CodeWhispererRequest) (<-chan *gateway.CodeWhispererStreamChunk, <-chan error, error) {
			chunks := make(chan *gateway.CodeWhispererStreamChunk, 1)
			errs := make(chan error, 1)
			chunks <- &gateway.CodeWhispererStreamChunk{
				Type:              "messageStart",
				ContentBlockIndex: 0,
			}
			close(chunks)
			close(errs)
			return chunks, errs, nil
		},
	}, logger.NewSimpleLogger(logger.LevelError))

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{
		"model":"claude-3-5-sonnet-20241022",
		"messages":[{"role":"user","content":"hello"}],
		"max_tokens":32,
		"stream":true
	}`))
	rec := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}

	handler.HandleMessages(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "event: message_start") {
		t.Fatalf("missing SSE event line: %s", body)
	}
	if !strings.Contains(body, `"type":"message_start"`) {
		t.Fatalf("missing Anthropic payload type: %s", body)
	}
}
